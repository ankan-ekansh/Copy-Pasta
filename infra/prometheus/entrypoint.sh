#!/bin/sh
# Write metrics token to file for Prometheus authorization config.
# If METRICS_TOKEN is unset, create an empty file (scrape without auth).
echo -n "${METRICS_TOKEN:-}" > /etc/prometheus/metrics-token

exec /bin/prometheus \
  --config.file=/etc/prometheus/prometheus.yml \
  --storage.tsdb.path=/prometheus \
  --storage.tsdb.retention.time=15d
