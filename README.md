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

### usage

```
$ prometheus-net-discovery --help
Usage of prometheus-net-discovery:
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

Generated environment variables:
   CONFIG_FILESDPATH
   CONFIG_INTERVAL
   CONFIG_LOG_FORMAT
   CONFIG_LOG_FORMATTER
   CONFIG_LOG_LEVEL
   CONFIG_NETWORKS

```

## Compile

```
 Example:docker build -t prometheus-net-discovery-9103 -f Dockerfile .
docker run -it prometheus-net-discovery-9103:latest /bin/sh
another terminal
```
## Add to the prome container
```
## copy the prometheus-net-discovery file to the local disk
docker cp <container-id>:/opt/net-discovery/prometheus-net-discovery .

## Added to the prometheus and create a new docker image
docker build -t prometheus_discovery_9103-nodeexp:0.0.2 -f Dockerfile_Discover-with-Prom .
```
