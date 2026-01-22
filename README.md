# prometheus-net-discovery
Scan entire networks for open ports with known prometheus exporters and generate files with targets to be used with `file_sd_config` file based discovery

### install

```
go get -u github.com/fortnoxab/prometheus-net-discovery
```

### example run

```
prometheus-net-discovery -networks "192.168.1.0/24" --filesdpath /tmp/
```

### Configuration options

| Option | Default | Description |
|--------|---------|-------------|
| `-workers` | 16 | Number of concurrent workers |
| `-scanratelimit` | 50 | Maximum hosts to probe per second |
| `-skipnetworkbroadcast` | true | Skip network and broadcast addresses |
| `-exporterexporterport` | 9999 | Port where exporter_exporter is listening |
| `-port` | 8080 | Port for the internal webserver |
| `-interval` | 60m | How often to scan |

#### Example configurations

Default scanning:
```
prometheus-net-discovery -networks "192.168.1.0/24" -filesdpath /tmp/
```

Gentle scanning for sensitive networks:
```
prometheus-net-discovery -networks "192.168.1.0/24" -filesdpath /tmp/ -workers 8 -scanratelimit 25
```

Fast scanning for trusted networks:
```
prometheus-net-discovery -networks "10.0.0.0/16" -filesdpath /tmp/ -workers 32 -scanratelimit 100
```

### usage

```
$ prometheus-net-discovery --help
Usage of prometheus-net-discovery:
  -exporterexporterport
    	Change value of ExporterExporterPort. (default 9999)
  -filesdpath
    	Change value of FileSdPath.
  -interval
    	Change value of Interval. (default 60m)
  -log-format
    	Change value of Log-Format. (default text)
  -log-formatter
    	Change value of Log-Formatter. (default <nil>)
  -log-level
    	Change value of Log-Level.
  -networks
    	Change value of Networks.
  -port
    	Change value of Port. (default 8080)
  -workers
    	Change value of Workers. (default 16)
  -scanratelimit
    	Change value of ScanRateLimit. (default 50)
  -skipnetworkbroadcast
    	Change value of SkipNetworkBroadcast. (default true)

Generated environment variables:
   CONFIG_EXPORTEREXPORTERPORT
   CONFIG_FILESDPATH
   CONFIG_INTERVAL
   CONFIG_LOG_FORMAT
   CONFIG_LOG_FORMATTER
   CONFIG_LOG_LEVEL
   CONFIG_NETWORKS
   CONFIG_PORT
   CONFIG_WORKERS
   CONFIG_SCANRATELIMIT
   CONFIG_SKIPNETWORKBROADCAST

```
