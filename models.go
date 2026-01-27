package main

import (
	"github.com/fortnoxab/fnxlogrus"
)

// Config is main application configuration.
type Config struct {
	// Comma separated string of networks in 192.168.0.1/24 format
	Networks string
	// Interval is how often to scan. Default 60m
	Interval string `default:"60m"`
	// FileSdPath specifies where to put your generated files. Example /etc/prometheus/file_sd/
	FileSdPath          string
	Log                 fnxlogrus.Config
	Port                string `default:"8080"`
	ExporterExporterPort string `default:"9999"`
	// Workers is the number of concurrent workers for scanning. Default 16
	Workers int `default:"16"`
	// ScanRateLimit is the maximum number of hosts to scan per second. Default 50 to reduce network load
	ScanRateLimit float64 `default:"50"`
	// SkipNetworkBroadcast skips network and broadcast addresses. Default true
	SkipNetworkBroadcast bool `default:"true"`
}

// Exporters is a list of addresses grouped by exporter name.
type Exporters map[string][]Address

// Address represents a host:ip to monitor.
type Address struct {
	IP          string
	Hostname    string
	Subnet      string
	Port        string
	Exporter    string
	MetricsPath string
}

// Group is a prometheus target config. Copied struct from prometheus repo.
type Group struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}
