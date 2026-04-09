# Full-Stack Example: 6-Node Cluster with Grafana Observability

This example deploys a complete Rebost cluster with full observability using Docker Compose:

- **5 Rebost storage nodes + 1 proxy node** forming a peer-to-peer cluster with 3x replication
- **Prometheus** scraping metrics from all nodes
- **Loki + Promtail** collecting structured JSON logs
- **Tempo** receiving distributed traces via OTLP
- **Grafana** with pre-provisioned datasources and a Rebost dashboard

## Architecture

```
                    ┌──────────┐
                    │  Grafana  │ :3000
                    └────┬─────┘
           ┌─────────────┼─────────────┐
           ▼             ▼             ▼
    ┌────────────┐ ┌──────────┐ ┌──────────┐
    │ Prometheus │ │   Loki   │ │  Tempo   │
    │    :9090   │ │  :3100   │ │  :3200   │
    └─────┬──────┘ └────┬─────┘ └────┬─────┘
          │              │            │
          │ scrape    promtail     OTLP HTTP
          │              │            │
    ┌─────┴──────────────┴────────────┴─────┐
    │                                       │
    │  node1  node2  node3  node4  node5    │
    │  :3901  :3902  :3903  :3904  :3905    │
    │             proxy (no volumes)        │
    │                :3906                  │
    │                                       │
    │         Rebost Cluster (gossip)       │
    └───────────────────────────────────────┘
```

## Prerequisites

- Docker and Docker Compose
- curl (for `make populate` and `make status`)

## Quick Start

```bash
# Start everything
make up

# Wait for nodes to become healthy (~15s)
make status

# Upload 50 random images
make populate

# Open Grafana in the browser
make grafana
```

## Make Targets

Run `make help` to see all available targets:

| Target       | Description                                              |
| ------------ | -------------------------------------------------------- |
| `help`       | Show all available targets                               |
| `up`         | Start all services                                       |
| `down`       | Stop all services and remove volumes                     |
| `logs`       | Follow container logs                                    |
| `status`     | Check node health and print service URLs                 |
| `populate`   | Download and upload `C` random images (default: `C=50`)  |
| `bench`      | Load test: GET images across all nodes                   |
| `grafana`    | Open Grafana dashboard in the browser                    |
| `prometheus` | Open Prometheus UI in the browser                        |
| `dashboard`  | Open node1 Rebost dashboard in the browser               |
| `clean`      | Remove all containers, volumes, and networks             |

The `populate` target accepts a `C` variable to control how many images to upload:

```bash
# Upload 10 images instead of the default 50
make populate C=10
```

The `bench` target fires concurrent GET requests across all 6 nodes (including the proxy), spreading them over the uploaded images. Configurable with `N` (total requests), `CONC` (concurrency), and `C` (image pool size):

```bash
# Default: 500 requests, 10 concurrent, 50 images
make bench

# Heavy load: 2000 requests, 50 concurrent
make bench N=2000 CONC=50

# Quick smoke test
make bench N=50 CONC=5 C=10
```

Open `make grafana` while the bench runs to watch HTTP metrics, traces, and logs in real time.

## Service URLs

| Service     | URL                   |
| ----------- | --------------------- |
| Node 1 API  | http://localhost:3901 |
| Node 2 API  | http://localhost:3902 |
| Node 3 API  | http://localhost:3903 |
| Node 4 API  | http://localhost:3904 |
| Node 5 API  | http://localhost:3905 |
| Proxy API   | http://localhost:3906 |
| Node 1 Dash | http://localhost:3911 |
| Node 2 Dash | http://localhost:3912 |
| Node 3 Dash | http://localhost:3913 |
| Node 4 Dash | http://localhost:3914 |
| Node 5 Dash | http://localhost:3915 |
| Proxy Dash  | http://localhost:3916 |
| Grafana     | http://localhost:3000 |
| Prometheus  | http://localhost:9090 |
| Loki        | http://localhost:3100 |
| Tempo       | http://localhost:3200 |

## S3 API Examples

```bash
# Upload a file
curl -T ./myfile.txt http://localhost:3901/mybucket/myfile.txt

# Download from any node
curl -o myfile.txt http://localhost:3903/mybucket/myfile.txt

# Check if a file exists
curl -I http://localhost:3901/mybucket/myfile.txt

# Delete a file
curl -X DELETE http://localhost:3901/mybucket/myfile.txt

# Upload with TTL (auto-delete after 1 hour)
curl -T ./tmp.txt "http://localhost:3901/mybucket/tmp.txt?ttl=1h"
```

## Proxy Node

The `proxy` service (port 3906) runs without any local volumes. It joins the cluster via gossip and forwards all S3 operations to the 5 storage nodes. This demonstrates Rebost's proxy mode, useful for placing internet-facing nodes in front of private storage nodes.

You can upload through the proxy and download from any storage node (or vice versa):

```bash
# Upload through the proxy
curl -T ./myfile.txt http://localhost:3906/mybucket/myfile.txt

# Download from a storage node
curl -o myfile.txt http://localhost:3902/mybucket/myfile.txt
```

The proxy appears in the dashboard with zero volume usage and participates in metrics/logging/tracing like any other node.

## Exploring Observability

### Logs (Loki)

Open Grafana > Explore > select **Loki** datasource.

Example LogQL queries:

```logql
# All logs from node1
{node="node1"}

# Audit events only
{node=~"node.*"} | json | event=~"audit\\..*"

# Replica events
{node=~"node.*"} | json | event=~"replica\\..*"
```

### Traces (Tempo)

Open Grafana > Explore > select **Tempo** datasource.

Search for traces by service name `rebost`. Click a trace to see the full span tree across nodes. Trace IDs in log lines link directly to Tempo.

### Dashboard

Open Grafana > Dashboards > **Rebost**.

The pre-built dashboard includes:

- **Storage Overview** — volume usage and file counts
- **Replication** — replica/deletion queue depths and replication lag
- **HTTP Performance** — request duration percentiles, active requests, request rate by status code
- **Database** — BoltDB operation duration and rate
- **Logs** — live log stream from all nodes, plus a filtered audit events panel
- **Traces** — recent traces from Tempo, clickable to inspect the full span tree

## Cleanup

```bash
make clean
```

This removes all containers, volumes, and networks created by the example.
