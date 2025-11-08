# Grafana Configuration

## Data Source

The Prometheus data source is automatically provisioned when Grafana starts. You should see it in **Configuration → Data Sources**.

### If you need to add it manually:

**DO NOT use `localhost:9090`** - Grafana runs in a Docker container, so `localhost` refers to the Grafana container itself.

**Use the Docker service name instead:**
- URL: `http://prometheus:9090`
- This uses Docker's internal networking

### Accessing Prometheus from your host machine:

- Prometheus UI: http://localhost:9091 (note: port 9091, not 9090)
- The container exposes Prometheus on port 9091 to avoid conflicts

## Dashboards

The Syncer Telemetry Dashboard is automatically provisioned. You should see it in **Dashboards**.

## Troubleshooting

### Data source connection fails:

1. Check that Prometheus is running: `docker-compose -f docker-compose.dev.yaml ps prometheus`
2. Check Prometheus targets: http://localhost:9091/targets
3. Verify the data source URL is `http://prometheus:9090` (not localhost)
4. Restart Grafana: `docker-compose -f docker-compose.dev.yaml restart grafana`

### No metrics showing:

1. Ensure the syncer container is running and exposing metrics on port 9090
2. Check Prometheus is scraping: http://localhost:9091/targets (should show syncer as UP)
3. Verify metrics are available: `curl http://localhost:9090/metrics`
4. Trigger a sync operation to generate metrics: `curl -X POST http://localhost:8000/sync`

