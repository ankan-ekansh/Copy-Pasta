#!/bin/sh
set -eu

# Write metrics token to file for Prometheus authorization config.
# If METRICS_TOKEN is unset, create an empty file (scrape without auth).
echo -n "${METRICS_TOKEN:-}" > /etc/prometheus/metrics-token

# Substitute the scrape target FQDN into the config.
# SCRAPE_TARGET is set by the deploy script (e.g., the app's internal FQDN).
if [ -z "${SCRAPE_TARGET:-}" ]; then
  echo "FATAL: SCRAPE_TARGET is not set. Cannot configure Prometheus." >&2
  exit 1
fi

sed -i "s|SCRAPE_TARGET_PLACEHOLDER|${SCRAPE_TARGET}|g" /etc/prometheus/prometheus.yml

# Verify substitution succeeded
if grep -q "SCRAPE_TARGET_PLACEHOLDER" /etc/prometheus/prometheus.yml; then
  echo "FATAL: Failed to substitute SCRAPE_TARGET_PLACEHOLDER in prometheus.yml" >&2
  exit 1
fi

exec /bin/prometheus \
  --config.file=/etc/prometheus/prometheus.yml \
  --storage.tsdb.path=/prometheus \
  --storage.tsdb.retention.time=15d
