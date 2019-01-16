package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
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
	Timeout: 5 * time.Second,
}

func alive(host, port string, path string) bool {
	if path != "" {
		u := fmt.Sprintf(path, net.JoinHostPort(host, port))
		resp, err := client.Get(u)
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
