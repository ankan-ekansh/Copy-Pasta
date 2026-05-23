FROM grafana/grafana:11.1.0

ARG PROVISIONING_DIR=provisioning

# Bake in provisioning configs and dashboards
COPY ${PROVISIONING_DIR} /etc/grafana/provisioning
COPY dashboards /var/lib/grafana/dashboards
