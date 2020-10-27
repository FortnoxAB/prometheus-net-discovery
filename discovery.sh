#!/bin/sh
/bin/prometheus-net-discovery -interval 10m -networks $Subnet --filesdpath /tmp/ &
sleep 5;
/bin/prometheus --config.file=/etc/prometheus/prometheus.yml --storage.tsdb.path=/prometheus --web.console.libraries=/usr/share/prometheus/console_libraries --web.console.templates=/usr/share/prometheus/consoles --web.enable-lifecycle --storage.tsdb.no-lockfile --storage.tsdb.allow-overlapping-blocks --storage.tsdb.retention.time=30d
