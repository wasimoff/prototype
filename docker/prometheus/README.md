# wasimoff: prometheus

This directory contains an example Docker Compose configuration to start [Prometheus](https://prometheus.io/) and [Grafana](https://grafana.com/) in containers to collect metrics from a running Wasimoff instance. You'll probably want to tweak networking, volumes and the `GF_SECURITY_ADMIN_PASSWORD` for a proper deployment.

The included `dashboard.json` can be imported Grafana to get started with your dashboards.