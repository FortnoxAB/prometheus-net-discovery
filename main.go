package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fortnoxab/fnxlogrus"
	"github.com/koding/multiconfig"
	"github.com/sirupsen/logrus"
)

// ExporterConfig configures ports to scan to what filename to save it to.
// if path is set we will try to make a HTTP get and find # TYPE in the first 10 rows of the response to make sure we know its prometheus metrics.
// TODO make this runtime configurable
type ExporterConfig map[string]struct {
	filename string
	path     string
}

var exporterConfig = ExporterConfig{
	"8081": {
		filename: "php",
		path:     "http://%s/metrics",
	},
	"9100": {
		filename: "node",
	},
	"9108": {
		filename: "elasticsearch",
	},
	"9114": {
		filename: "elasticsearch",
	},
	"9216": {
		filename: "mongodb",
	},
	"9091": {
		filename: "minio",
		path:     "http://%s/minio/prometheus/metrics",
	},
	"9101": {
		filename: "haproxy",
	},
	"9104": {
		filename: "mysql",
	},
	"9113": {
		filename: "nginx",
	},
	"9121": {
		filename: "redis",
	},
	"9150": {
		filename: "memcached",
	},
	"9154": {
		filename: "postfix",
	},
	"9182": {
		filename: "wmi",
	},
	"9187": {
		filename: "postgres",
	},
	"9188": {
		filename: "pgbouncer",
	},
	"9189": {
		filename: "barman",
	},
	"9253": {
		filename: "php-fpm",
	},
	"9308": {
		filename: "kafka",
	},
	"9496": {
		filename: "389ds",
	},
}

func main() {
	config := &Config{}
	multiconfig.MustLoad(&config)

	//logrus.SetReportCaller(true)

	fnxlogrus.Init(config.Log, logrus.StandardLogger())

	//router := gin.New()
	//router.Use(ginlogrus.New(logrus.StandardLogger(), "/health", "/metrics"), gin.Recovery())

	networks := strings.Split(config.Networks, ",")

	interval, err := time.ParseDuration(config.Interval)
	if err != nil {
		logrus.Errorf("invalid duration %s: %s", config.Interval, err.Error())
		return
	}

	logrus.Infof("Running with interval %s", interval)

	ticker := time.NewTicker(interval)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	ctx, cancel := context.WithCancel(context.Background())
	runDiscovery(ctx, config, networks)
	for {
		select {
		case <-ticker.C:
			runDiscovery(ctx, config, networks)
		case <-interrupt:
			cancel()
			ticker.Stop()
			log.Println("Shutting down")
			return
		}
	}
}

func runDiscovery(parentCtx context.Context, config *Config, networks []string) {
	logrus.Info("Running discovery")

	job := make(chan func())
	exporter := make(chan *Address)
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	for i := 0; i < 32; i++ {
		go func() {
			for {
				select {
				case fn := <-job:
					fn()
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	count := make(chan int64)
	go func() {
		var i int64
		for _, v := range networks {
			network := strings.TrimSpace(v)
			if network == "" {
				continue
			}
			i += discoverNetwork(network, job, exporter)
		}
		count <- i
	}()

	exporters := make(Exporters)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var a int64
		var b int64
		for {
			select {
			case address := <-exporter:
				a++
				if address != nil {
					exporters[address.Exporter] = append(exporters[address.Exporter], *address)
				}
			case count := <-count:
				b = count
			}
			if b == a {
				return
			}
		}
	}()

	wg.Wait()

	for name, addresses := range exporters {
		err := writeFileSDConfig(config, name, addresses)
		if err != nil {
			logrus.Error(err)
			continue
		}
	}
	logrus.Info("discovery done")
}

//func calculateIps(network string) []string{
//}

var vipRegexp = regexp.MustCompile(`^.+-vip(\d+)?\.`)

func isVip(name string) bool {
	return vipRegexp.MatchString(name)
}

func discoverNetwork(network string, queue chan func(), exporter chan *Address) int64 {
	networkip, ipnet, err := net.ParseCIDR(network)
	if err != nil {
		log.Fatal("network CIDR could not be parsed:", err)
	}
	i := int64(0)
	for ip := networkip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		network := network
		ip := ip.String()
		for port, data := range exporterConfig {
			i++
			port := port
			data := data
			queue <- func() {
				//TODO also check if http request contains "# TYPE"
				if !alive(ip, port, data.path) {
					exporter <- nil
					return
				}

				logrus.Info(net.JoinHostPort(ip, port), " is alive")
				addr, _ := net.LookupAddr(ip) // #nosec
				hostname := strings.TrimRight(getFirst(addr), ".")
				if hostname == "" {
					logrus.Error("missing reverse record for ", ip)
					exporter <- nil
					return
				}
				if isVip(hostname) {
					logrus.Info("skipping vip ", hostname, ip)
					exporter <- nil
					return
				}

				a := Address{
					IP:       strings.TrimSpace(ip),
					Hostname: strings.TrimSpace(hostname),
					Subnet:   strings.TrimSpace(network),
					Exporter: data.filename,
					Port:     port,
				}

				exporter <- &a
			}
		}
	}
	return i
}

func getOldGroups(path string) ([]Group, error) {
	file, err := os.Open(path) // #nosec
	if os.IsNotExist(err) {
		return nil, nil // ignore if files not found
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	oldGroups := []Group{}
	err = json.NewDecoder(file).Decode(&oldGroups)
	if err == io.EOF { //Ignore empty files
		return nil, nil
	}
	return oldGroups, err
}

func writeFileSDConfig(config *Config, path string, addresses []Address) error {
	path = filepath.Join(config.FileSdPath, path+".json")

	previous, err := getOldGroups(path)
	if err != nil {
		return err
	}

	groups := []Group{}

	for _, v := range addresses {
		group := Group{
			Targets: []string{net.JoinHostPort(v.IP, v.Port)},
			Labels: map[string]string{
				"subnet": v.Subnet,
				"host":   v.Hostname,
			},
		}
		groups = append(groups, group)
	}

	// Dont remove targets if they happened to be down at the moment
	for _, prev := range previous {
		exists := false
		for _, current := range groups {
			if getFirst(current.Targets) == getFirst(prev.Targets) {
				exists = true
			}
		}
		if !exists {
			logrus.Errorf("%s (%s) was removed, keeping it anyway. To remove it delete it from %s manually", prev.Targets[0], prev.Labels["host"], path)
			groups = append(groups, prev)
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "\t")
	return encoder.Encode(groups)
}
