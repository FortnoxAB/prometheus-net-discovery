package main

import (
	"context"
	"encoding/json"
	"errors"
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
type ExporterConfig []struct {
	port     string
	filename string
	path     string
}

var exporterConfig = ExporterConfig{
	// We only support exporter_exporter at the moment
	{
		port:     "",
		filename: "",
	},
}

var mutex sync.Mutex

func main() {
	config := &Config{}
	multiconfig.MustLoad(&config)
	exporterConfig[0].port = config.ExpoterExporterPort

	fnxlogrus.Init(config.Log, logrus.StandardLogger())

	networks := strings.Split(config.Networks, ",")

	interval, err := time.ParseDuration(config.Interval)
	if err != nil {
		logrus.Errorf("invalid duration %s: %s", config.Interval, err.Error())
		return
	}

	logrus.Infof("Running with interval %s", interval)

	ticker := time.NewTicker(interval)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	defer cancel()

	go func() {
		startWs(config, ctx)
		ticker.Stop()
		log.Println("Shutting down")
	}()

	runDiscovery(ctx, config, networks)
	for {
		select {
		case <-ticker.C:
			runDiscovery(ctx, config, networks)
		case <-ctx.Done():
			return
		}
	}
}

func runDiscovery(parentCtx context.Context, config *Config, networks []string) {
	logrus.Info("Running discovery")

	job := make(chan func(context.Context))
	exporter := make(chan *Address)
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i < 128; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			for {
				select {
				case fn, ok := <-job:
					if !ok {
						logrus.Debugf("worker %d finished", i)
						return
					}
					fn(ctx)
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		for _, v := range networks {
			if ctx.Err() != nil {
				return
			}
			network := strings.TrimSpace(v)
			if network == "" {
				continue
			}
			discoverNetwork(network, job, exporter)
		}
		close(job)
	}()

	exporters := make(Exporters)

	go func() {
		for {
			select {
			case address, ok := <-exporter:
				if !ok {
					return
				}
				exporters[address.Exporter] = append(exporters[address.Exporter], *address)
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()

	saveConfigs(ctx, config, exporters)
	logrus.Info("discovery done")
}

func saveConfigs(ctx context.Context, config *Config, exporters Exporters) {
	mutex.Lock()
	defer mutex.Unlock()
	for name, addresses := range exporters {
		if ctx.Err() != nil {
			return
		}
		err := writeFileSDConfig(config, name, addresses)
		if err != nil {
			logrus.Error(err)
			continue
		}
	}
}

var vipRegexp = regexp.MustCompile(`^.+-vip(\d+)?\.`)

func isVip(name string) bool {
	return vipRegexp.MatchString(name)
}

func discoverNetwork(network string, queue chan func(context.Context), exporter chan *Address) {
	networkip, ipnet, err := net.ParseCIDR(network)
	if err != nil {
		log.Fatal("network CIDR could not be parsed:", err)
	}
	for ip := networkip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		network := network
		ip := ip.String()
		queue <- func(ctx context.Context) {
			for _, data := range exporterConfig {
				if ctx.Err() != nil {
					return
				}
				port := data.port
				logrus.Debugf("scanning port: %s:%s", ip, port)
				var exporters []string
				exporters, err = checkExporterExporter(ctx, ip, port)
				if err != nil {
					if !errors.Is(err, context.DeadlineExceeded) && !IsTimeout(err) {
						logrus.Debugf("error fetching from exporter_exporter: %s", err)
					}
					logrus.Debugf("%s:%s was not open", ip, port)
					continue
				}

				logrus.Info(net.JoinHostPort(ip, port), " is alive")
				addr, _ := net.LookupAddr(ip) // #nosec
				hostname := strings.TrimRight(getFirst(addr), ".")
				if hostname == "" {
					logrus.Error("missing reverse record for ", ip)
					continue
				}
				if isVip(hostname) && !strings.HasPrefix(hostname, "k8s-") {
					logrus.Info("skipping vip ", hostname, ip)
					continue
				}

				if len(exporters) > 0 {
					for _, filename := range exporters {
						a := Address{
							IP:       strings.TrimSpace(ip),
							Hostname: strings.TrimSpace(hostname),
							Subnet:   strings.TrimSpace(network),
							Exporter: filename,
							Port:     port,
						}
						exporter <- &a
					}
				}
			}
		}
	}
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
	if errors.Is(err, io.EOF) { // Ignore empty files
		return nil, nil
	}
	return oldGroups, err
}

func writeFileSDConfig(config *Config, exporterName string, addresses []Address) error {
	path := filepath.Join(config.FileSdPath, exporterName+".json")

	if _, err := os.Stat(config.FileSdPath); os.IsNotExist(err) {
		os.MkdirAll(config.FileSdPath, 0755)
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
		if v.Port == config.ExpoterExporterPort {
			group.Labels["__metrics_path__"] = "/proxy"
			group.Labels["__param_module"] = exporterName
		}
		groups = append(groups, group)
	}

	previous, err := getOldGroups(path)
	if err != nil {
		return err
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
