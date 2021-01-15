package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func getFirst(s []string) string {
	for _, v := range s {
		return v
	}
	return ""
}

var client = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec only cockroach at the moment
	},
}

func checkExporterExporter(parentCtx context.Context, host, port string) ([]string, error) {
	u := fmt.Sprintf("http://%s", net.JoinHostPort(host, port))
	ctx, cancel := context.WithTimeout(parentCtx, time.Second*3)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error http get %s: %w", u, err)
	}

	defer resp.Body.Close()
	var anyJSON map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&anyJSON)
	if err != nil {
		return nil, fmt.Errorf("error decoding json body: %w", err)
	}

	exporters := make([]string, len(anyJSON))
	i := 0
	for k := range anyJSON {
		exporters[i] = k
		i++
	}

	return exporters, nil
}

func alive(parentCtx context.Context, host, port, path string) bool {
	if path != "" {
		u := fmt.Sprintf(path, net.JoinHostPort(host, port))

		ctx, cancel := context.WithTimeout(parentCtx, time.Second*3)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err != nil {
			logrus.Errorf("error creating request: %s", err)
			return false
		}
		resp, err := client.Do(req)
		if err != nil {
			return false
		}
		defer resp.Body.Close()

		r := bufio.NewReader(resp.Body)
		for i := 0; i < 10; i++ {
			line, _, err := r.ReadLine()
			if err != nil {
				return false
			}
			if bytes.Contains(line, []byte("# TYPE")) {
				return true
			}
		}
		return false
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 200*time.Millisecond)
	if err != nil {
		return false
	}

	if conn != nil {
		conn.Close()
		return true
	}
	return false
}
