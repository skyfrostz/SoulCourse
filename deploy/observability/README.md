# Public beta observability

Load `prometheus-alerts.yml` through Prometheus `rule_files`, then validate it with `promtool check rules`. Import `grafana-public-beta-dashboard.json` into a Grafana folder whose access is restricted to the operations team and select the Prometheus datasource when prompted.

Prometheus must scrape `GET /metrics` with the production `METRICS_TOKEN` as either `X-Metrics-Token` or a Bearer token. Do not expose that token to Grafana browser clients. Scrape the application through Prometheus and connect Grafana to Prometheus server-side.

The host utilization panels and alerts require node_exporter on the application host. Route `critical` alerts to the release rollback channel and `warning` alerts to the on-call channel. Test each route before public traffic is enabled.
