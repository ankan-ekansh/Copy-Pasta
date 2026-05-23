#!/bin/sh
# Write metrics token to file for Prometheus authorization config.
# If METRICS_TOKEN is unset, create an empty file (scrape without auth).
echo -n "${METRICS_TOKEN:-}" > /etc/prometheus/metrics-token

# Substitute the scrape target FQDN into the config.
# SCRAPE_TARGET is set by the deploy script (e.g., the app's internal FQDN).
if [ -n "$SCRAPE_TARGET" ]; then
  sed -i "s|SCRAPE_TARGET_PLACEHOLDER|${SCRAPE_TARGET}|g" /etc/prometheus/prometheus.yml
fi

exec /bin/prometheus \
  --config.file=/etc/prometheus/prometheus.yml \
  --storage.tsdb.path=/prometheus \
  --storage.tsdb.retention.time=15d
