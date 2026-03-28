# Rebost Documentation

## Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Quick Start — Single Node](#quick-start--single-node)
4. [Multi-Node Cluster](#multi-node-cluster)
5. [Proxy Nodes](#proxy-nodes)
6. [Draining a Node](#draining-a-node)
7. [Configuration Reference](#configuration-reference)
8. [S3 API Reference](#s3-api-reference)
9. [Authentication](#authentication)
10. [Replication](#replication)
11. [TTL & Expiration](#ttl--expiration)
12. [Volume Sizing](#volume-sizing)
13. [Dashboard](#dashboard)
14. [Health & Readiness Endpoints](#health--readiness-endpoints)
15. [Metrics](#metrics)
16. [Tracing](#tracing)
17. [Known Limitations](#known-limitations)

---

## Overview

Rebost is a distributed, peer-to-peer object storage system. Key properties:

- **No master node.** Every node is master of its own objects and knows where replicas live. There is no single point of failure and no coordination overhead.
- **Zero-config clustering.** Adding a node to the cluster is just starting it and pointing it at any existing node with `--remote`. Nodes discover each other via a gossip protocol ([hashicorp/memberlist](https://github.com/hashicorp/memberlist)).
- **Content-addressed deduplication.** Files are stored by SHA1 signature. Multiple keys pointing at the same bytes share one copy on disk.
- **S3-compatible API.** Any AWS S3 library or tool works against Rebost without a dedicated client. Authentication is optional and uses AWS Signature V4.
- **Configurable replication.** Default replica count is 3. Rebost replicates asynchronously in the background and re-replicates automatically after a node recovers from downtime.

---

## Installation

### Docker (recommended)

```bash
docker pull xescugc/rebost
```

The image is published automatically from the `master` branch and on every version tag.

### Build from source

Requires Go 1.25+.

```bash
git clone https://github.com/xescugc/rebost.git
cd rebost
go build -o rebost .
```

---

## Quick Start — Single Node

Start a single node with one local volume:

```bash
docker run -d --name rebost \
  -p 3805:3805 -p 3806:3806 \
  -v $(pwd)/data:/data \
  xescugc/rebost serve --volumes /data --name mynode
```

Upload a file:

```bash
curl -T ./photo.jpg http://localhost:3805/mybucket/photo.jpg
```

Download the file:

```bash
curl -o photo.jpg http://localhost:3805/mybucket/photo.jpg
```

Delete the file:

```bash
curl -X DELETE http://localhost:3805/mybucket/photo.jpg
```

Check whether a file exists (returns `200` with metadata headers, or `404`):

```bash
curl -I http://localhost:3805/mybucket/photo.jpg
```

### Using the AWS CLI

Rebost is S3-compatible. Use `--endpoint-url` to point the CLI at your node and `--no-sign-request` when auth is not enabled:

```bash
# Upload
aws s3 cp ./photo.jpg s3://mybucket/photo.jpg \
  --endpoint-url http://localhost:3805 \
  --no-sign-request

# Download
aws s3 cp s3://mybucket/photo.jpg ./photo.jpg \
  --endpoint-url http://localhost:3805 \
  --no-sign-request

# Delete
aws s3 rm s3://mybucket/photo.jpg \
  --endpoint-url http://localhost:3805 \
  --no-sign-request
```

---

## Multi-Node Cluster

Rebost nodes find each other via gossip. To form a cluster, start the first node normally and point every subsequent node at it with `--remote`. Any existing node URL works — gossip propagates the full topology automatically.

### Local example with Docker

Create a shared network so containers resolve each other by name:

```bash
docker network create rebost
```

Start the first node:

```bash
docker run -d --name node1 --network rebost \
  -p 3805:3805 -p 3806:3806 \
  -v $(pwd)/v1:/data \
  xescugc/rebost serve --volumes /data --name node1
```

Start additional nodes, each pointing at node1:

```bash
docker run -d --name node2 --network rebost \
  -p 2020:2020 \
  -v $(pwd)/v2:/data \
  xescugc/rebost serve --volumes /data --port 2020 --name node2 \
    --remote http://node1:3805 --dashboard.enabled false

docker run -d --name node3 --network rebost \
  -p 3030:3030 \
  -v $(pwd)/v3:/data \
  xescugc/rebost serve --volumes /data --port 3030 --name node3 \
    --remote http://node1:3805 --dashboard.enabled false
```

Once all nodes are up, upload once and read from any node:

```bash
curl -T ./photo.jpg http://localhost:3805/mybucket/photo.jpg

# Any node can serve the file
curl http://localhost:2020/mybucket/photo.jpg
curl http://localhost:3030/mybucket/photo.jpg
```

### How joining works

1. The new node fetches `/config` from the `--remote` URL to get the gossip port.
2. It calls `memberlist.Join` — memberlist handles full state synchronisation.
3. The new node learns about all existing nodes and volumes immediately.

### Scaling out

There is no limit on the number of nodes. Each node can hold one or more local volumes (`--volumes` accepts a comma-separated list or can be specified multiple times). New nodes receive replicas of existing objects over time as the background replication queue drains.

---

## Proxy Nodes

A proxy node is a node that joins the cluster without any local storage volumes. It forwards all operations to peer storage nodes and is useful for building topologies where some nodes sit at the internet boundary while storage nodes remain on a private network.

```
Internet
    │
    ▼
proxy-node  (no local volumes, internet-facing)
    │  gossip + HTTP
    ▼
storage-node1  storage-node2  storage-node3  (private network)
```

### Starting a proxy node

Omit `--volumes` entirely:

```bash
docker run -d --name proxy --network rebost \
  -p 3805:3805 \
  xescugc/rebost serve --name proxy --remote http://storage-node1:3805
```

The node logs `"starting in proxy mode (no local volumes)"` at startup and joins the cluster gossip. All client requests routed to the proxy are forwarded to a storage peer that has capacity or holds the requested object.

### Behaviour

| Operation                | What the proxy does                                            |
| ------------------------ | -------------------------------------------------------------- |
| `PUT /{bucket}/{key}`    | Selects a peer with available capacity and delegates the write |
| `GET /{bucket}/{key}`    | Queries peers to locate the object, then streams it back       |
| `HEAD /{bucket}/{key}`   | Same as GET but returns only headers                           |
| `DELETE /{bucket}/{key}` | Locates the object on a peer and forwards the delete           |
| `PUT /replicas/{key}`    | Returns an error — proxies do not accept replicas              |
| `PATCH /replicas/{key}`  | Returns an error — proxies hold no replica metadata            |

### Notes

- A proxy node participates fully in gossip and is visible in the dashboard, but reports zero volume usage.
- Proxy nodes do not hold data and are therefore not targeted by the background replication loop.
- You can run multiple proxy nodes; each independently forwards requests to the cluster.
- If no storage peer has capacity, the proxy returns the same `InternalError` (cluster full) as any storage node would.

---

## Draining a Node

Draining is the safe way to decommission a node before removing it from the cluster. Unlike abrupt termination (`SIGTERM`/`SIGINT`, which stops the process immediately), a drain ensures that every locally-held file is safely replicated to other nodes before the node leaves.

### How to trigger

Send `SIGQUIT` to the running process:

```bash
# Graceful drain and shutdown
kill -QUIT $(pgrep rebost)
# or: Ctrl+\

# Fast shutdown (no drain)
kill $(pgrep rebost)
```

### What happens

1. For each locally-held file that does not already have enough external replicas, a replication job is created.
2. The node waits for all replication jobs to complete — every file is now safely held by other nodes.
3. Local copies are purged without propagating deletions to remote nodes (remote replicas are preserved).
4. The node leaves the cluster gossip and shuts down.

### Notes

- If the drain fails mid-way (e.g. not enough peers to satisfy the replica count), the node remains in the cluster and retains its local files. The drain can be safely retried by sending `SIGQUIT` again.
- Proxy nodes (started without `--volumes`) drain instantly because they hold no data.
- While draining, the node rejects all `CreateFile` and `CreateReplica` requests with an error. Other cluster members also skip routing writes to a draining node.

---

## Configuration Reference

All options can be provided as CLI flags, environment variables (uppercased with `_` separator, prefixed with `REBOST_`), or a config file.

| Flag                                    | Type     | Default       | Description                                                                                                                                                                       |
| --------------------------------------- | -------- | ------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--port` / `-p`                         | int      | `3805`        | HTTP port the node listens on                                                                                                                                                     |
| `--name`                                | string   | random 7-char | Unique node name in the cluster. Auto-generated if not set.                                                                                                                       |
| `--volumes` / `-v`                      | strings  | _(none)_      | Paths to local storage volumes. Repeat or comma-separate for multiple. Omit entirely to run as a [proxy node](#proxy-nodes). See [Volume Sizing](#volume-sizing) for size limits. |
| `--remote` / `-r`                       | string   | —             | URL of any existing cluster node to join. Omit to start a new single-node cluster.                                                                                                |
| `--replica`                             | int      | `3`           | Default replica count per object. Set to `-1` to disable replication on this node (storage-only mode).                                                                            |
| `--timing.volume-downtime`              | duration | `2m`          | How long a volume can be unreachable before Rebost starts re-replicating its objects to surviving nodes.                                                                          |
| `--timing.scrub-interval`               | duration | `24h`         | How often each volume re-verifies file checksums and auto-repairs corrupt copies from replicas.                                                                                   |
| `--timing.replica-check-interval`       | duration | `1h`          | How often each volume checks for under-replicated files and re-queues replication jobs.                                                                                           |
| `--timing.replica-consistency-interval` | duration | `1h`          | How often non-owner replicas verify they are still expected by the file owner and purge stale local copies.                                                                       |
| `--timing.rebalance-interval`           | duration | `1h`          | How often Rebost checks whether any locally-held primaries have been displaced by a higher-ranking HRW volume and transfers ownership. Disabled when `0` or negative.            |
| `--memberlist.port`                     | int      | `0` (auto)    | UDP/TCP port for gossip. Auto-assigned if `0`. Fix this port if you need deterministic firewall rules.                                                                            |
| `--cache.size`                          | int      | `200`         | Size of the per-node LRU cache that maps object keys to remote volume IDs, avoiding repeated `HEAD` queries to peers.                                                             |
| `--dashboard.port`                      | int      | `3806`        | HTTP port for the dashboard UI.                                                                                                                                                   |
| `--dashboard.enabled`                   | bool     | `true`        | Enable or disable the dashboard on this node.                                                                                                                                     |
| `--s3.access_key`                       | string   | —             | AWS access key ID. Leave empty to disable authentication.                                                                                                                         |
| `--s3.secret_key`                       | string   | —             | AWS secret access key. Required when `--s3.access_key` is set.                                                                                                                   |
| `--s3.auth_mode`                        | string   | `all`         | Authentication scope. `all` requires auth for every request. `write` requires auth only for mutating operations (PUT, DELETE, PATCH, POST); GET and HEAD are public.              |

---

## S3 API Reference

Rebost exposes an S3-compatible HTTP API using path-style addressing: `/{bucket}/{key}`.

**Buckets are key prefixes.** `PUT /photos/cat.jpg` stores the object with internal key `photos/cat.jpg`. No bucket state is stored; `PUT /{bucket}` and `DELETE /{bucket}` are accepted as no-ops for client compatibility.

### Object operations

| Method   | Path              | Description                                                                                       | Success          |
| -------- | ----------------- | ------------------------------------------------------------------------------------------------- | ---------------- |
| `PUT`    | `/{bucket}/{key}` | Upload an object. Optional query params: `?replica=N`, `?ttl=<duration>`, `?created_at=<RFC3339>` | `200 OK`         |
| `GET`    | `/{bucket}/{key}` | Download an object. Returns `Content-Length`, `ETag`, `Last-Modified` headers.                    | `200 OK`         |
| `HEAD`   | `/{bucket}/{key}` | Get object metadata without the body. Returns the same headers as GET plus `X-Rebost-VolumeID`.   | `200 OK`         |
| `DELETE` | `/{bucket}/{key}` | Delete an object. Propagates to all replicas asynchronously.                                      | `204 No Content` |
| `POST`   | `/{bucket}/{key}` | Multipart upload stub — always returns `501 Not Implemented`.                                     | —                |

### Bucket operations

| Method   | Path        | Description                                   | Response         |
| -------- | ----------- | --------------------------------------------- | ---------------- |
| `PUT`    | `/{bucket}` | Create bucket (no-op).                        | `200 OK`         |
| `DELETE` | `/{bucket}` | Delete bucket (no-op).                        | `204 No Content` |
| `GET`    | `/{bucket}` | List objects — returns `501 Not Implemented`. | —                |

### Internal inter-node routes

These routes are used for replication between nodes. They are always exempt from authentication.

| Method  | Path              | Description                                                                              |
| ------- | ----------------- | ---------------------------------------------------------------------------------------- |
| `PUT`   | `/replicas/{key}` | Accept a replica of an object from a peer node.                                          |
| `PATCH` | `/replicas/{key}` | Update replica location metadata after a new replica is placed.                          |
| `GET`   | `/replicas/{key}` | Return the VolumeIDs and replica count for a file (used by the consistency check).       |
| `GET`   | `/config`         | Return this node's configuration (used during cluster join).                             |

### Error responses

All S3 routes return XML-encoded errors:

```xml
<Error>
  <Code>NoSuchKey</Code>
  <Message>The specified key does not exist.</Message>
</Error>
```

Common codes: `NoSuchKey` (404), `InternalError` (500), `AccessDenied` (403), `NotImplemented` (501).

### Upload query parameters

| Parameter    | Type     | Description                                                                                     |
| ------------ | -------- | ----------------------------------------------------------------------------------------------- |
| `replica`    | int      | Override the node's default replica count for this object.                                      |
| `ttl`        | duration | Set an expiration TTL (e.g. `24h`, `30m`). The object is deleted automatically when it expires. |
| `created_at` | RFC3339  | Override the creation timestamp (used for replica synchronisation).                             |

---

## Authentication

Authentication is optional. When enabled, Rebost validates requests using **AWS Signature V4** — the same scheme used by the real AWS S3 API. Any AWS SDK or tool that supports custom endpoints will work.

Internal routes (`/config`, `/replicas/*`) are always exempt from authentication, even when credentials are configured. This lets nodes replicate to each other without credentials.

### Enable auth

```bash
docker run -d -p 3805:3805 -v $(pwd)/data:/data \
  xescugc/rebost serve --volumes /data \
    --s3.access_key myaccesskey \
    --s3.secret_key mysecretkey
```

### Auth modes

| Mode            | Behaviour                                                       |
| --------------- | --------------------------------------------------------------- |
| `all` (default) | Every request requires a valid AWS Signature V4.                |
| `write`         | GET and HEAD are public; PUT, DELETE, PATCH, POST require auth. |

```bash
# Write-only auth — public reads, protected writes
docker run -d -p 3805:3805 -v $(pwd)/data:/data \
  xescugc/rebost serve --volumes /data \
    --s3.access_key myaccesskey \
    --s3.secret_key mysecretkey \
    --s3.auth_mode write
```

### AWS CLI with auth

```bash
AWS_ACCESS_KEY_ID=myaccesskey AWS_SECRET_ACCESS_KEY=mysecretkey \
  aws s3 cp ./photo.jpg s3://mybucket/photo.jpg \
  --endpoint-url http://localhost:3805

AWS_ACCESS_KEY_ID=myaccesskey AWS_SECRET_ACCESS_KEY=mysecretkey \
  aws s3 cp s3://mybucket/photo.jpg ./photo.jpg \
  --endpoint-url http://localhost:3805
```

### Notes

- Region is not validated — use any region string (e.g. `us-east-1`) in your SDK configuration.
- Path-style addressing must be enabled in your SDK if it offers the choice.

---

## Replication

Rebost replicates objects asynchronously in the background.

### How it works

1. When an object is uploaded, the cluster uses **Highest Random Weight (HRW / Rendezvous) hashing** to choose which volume stores the primary copy. Every `(key, volumeID)` pair is scored with `fnv64a(key + volumeID)`; the highest-scoring volume is the natural owner. This makes placement deterministic and predictable: the same key always maps to the same volume as long as the cluster topology is unchanged.
2. If the HRW-winning volume is full, the next-ranked volume is tried automatically.
3. Lookup uses the same ranking: the node walks the top-`Replica` ranked volumes before falling back to a cluster-wide broadcast. This reduces cold-cache lookup from O(n) to O(1) in the common case.
4. The background replication loop picks up jobs, selects peer nodes that do not yet have a copy, transfers the object, and updates all replica holders with the new location.
5. Replica metadata travels with every file: each node that holds a copy knows the volume IDs of all other copies.

### Replica count

The default replica count is `3` (configurable with `--replica`). This means Rebost will try to keep 3 copies of every object across the cluster. You can override per-upload with the `?replica=N` query parameter:

```bash
# Store with 5 replicas
curl -T ./video.mp4 "http://localhost:3805/mybucket/video.mp4?replica=5"

# Store with 1 replica (no replication)
curl -T ./tmp.log "http://localhost:3805/mybucket/tmp.log?replica=1"
```

Set `--replica -1` on a node to make it a storage-only node that never initiates replication.

### HRW rebalancing

When a new node joins the cluster, it may become the HRW winner for keys that were previously owned by an older node. Rebost automatically transfers primary ownership to the new winner.

Each tick, Rebost walks every locally-owned file and checks whether its HRW rank is still the highest. If a newer volume ranks higher, the primary copy is transferred to that volume and the local copy is removed. Replicas on other nodes stay in place.

Rebalancing runs every `1h` by default. To disable it, set `--timing.rebalance-interval 0`. To use a shorter interval:

```bash
rebost serve --volumes /data --timing.rebalance-interval 10m
```

### Node downtime and recovery

Rebost monitors the gossip heartbeat of each node. When a node disappears:

1. All volume IDs owned by that node are stamped with the departure time.
2. After `--timing.volume-downtime` (default 2 minutes), Rebost assumes the node is gone and starts re-replicating the affected objects to surviving nodes.

If the node comes back before the downtime threshold, replication is not triggered.

---

## TTL & Expiration

Objects can be given a time-to-live at upload time. Rebost deletes them automatically once they expire.

```bash
# Expire in 24 hours
curl -T ./session.json "http://localhost:3805/mybucket/session.json?ttl=24h"

# Expire in 30 minutes
curl -T ./tmp.bin "http://localhost:3805/mybucket/tmp.bin?ttl=30m"
```

The TTL is stored with the object and propagated to all replicas. The expiration loop runs every second per volume. When an object expires, its key and all replicas are deleted.

Valid duration units: `s` (seconds), `m` (minutes), `h` (hours). These follow Go's `time.Duration` syntax (e.g. `1h30m`).

---

## Volume Sizing

By default a volume can use all available disk space on the filesystem it lives on. To cap a volume:

```bash
# Limit to 20 GB
docker run -d -p 3805:3805 \
  -v $(pwd)/data:/data \
  xescugc/rebost serve --volumes /data:20G
```

Size suffixes: `K`, `M`, `G`, `T` (powers of 1024). If the volume is full, Rebost rejects uploads with `InternalError`.

Multiple volumes on the same node:

```bash
docker run -d -p 3805:3805 \
  -v $(pwd)/vol1:/vol1 -v $(pwd)/vol2:/vol2 \
  xescugc/rebost serve --volumes /vol1,/vol2
```

Each volume gets its own BoltDB metadata database (`my.db`) stored inside the volume directory.

---

## Dashboard

Each node runs a lightweight web dashboard on port `3806` by default. It shows:

- All nodes currently visible in the cluster.
- Per-node and per-volume disk usage (used vs total, with colour-coded fill percentage).

Access it at `http://localhost:3806` after starting a node.

To disable the dashboard:

```bash
xescugc/rebost serve --volumes /data --dashboard.enabled false
```

To run the dashboard on a different port:

```bash
xescugc/rebost serve --volumes /data --dashboard.port 8080
```

---

## Health & Readiness Endpoints

Three endpoints are always reachable regardless of node readiness state (they bypass the readiness gate used by the S3 API):

| Path | Method | Description |
|------|--------|-------------|
| `/live` | GET | Liveness probe — always returns 200 while the process is running |
| `/health` | GET | Alias for `/live` |
| `/ready` | GET | Readiness probe — 200 when the node is running and all local volumes are healthy, 503 otherwise |

### Response format

Success (200):
```json
{"status": "ok"}
```

Failure (503):
```json
{"status": "not_ready", "reason": "<description>"}
```

Possible `reason` values:
- `"node not running"` — node is starting up or draining
- `"volume <id> unhealthy: <error>"` — a local volume failed its health check

These endpoints are suitable for use as Kubernetes liveness/readiness probes or load-balancer health checks.

---

## Metrics

Rebost exposes a Prometheus-compatible `/metrics` endpoint on the same port as the S3 API. It uses the [OpenTelemetry](https://opentelemetry.io/) SDK with a Prometheus exporter.

### Endpoint

```
GET /metrics
```

Always reachable — no readiness gate. Responds with Prometheus text format.

### Exposed metrics

| Metric | Type | Description |
|--------|------|-------------|
| `http_server_request_duration_seconds` | Histogram | HTTP request latency per method, route, and status code |
| `http_server_active_requests` | Gauge | In-flight HTTP requests |
| `rebost_volume_storage_used_bytes` | Gauge | Bytes used on each local volume (label: `volume_id`) |
| `rebost_volume_storage_total_bytes` | Gauge | Total capacity of each local volume (label: `volume_id`) |
| `rebost_volume_files` | Gauge | Number of file records on each local volume (label: `volume_id`) |
| `rebost_volume_replica_queue_depth` | Gauge | Pending replica jobs per volume (label: `volume_id`) |
| `rebost_volume_deletion_queue_depth` | Gauge | Pending deletion jobs per volume (label: `volume_id`) |
| `rebost_volume_replication_lag_seconds` | Gauge | Age of oldest pending replica job in seconds; 0 if queue is empty (label: `volume_id`) |
| `rebost_db_operation_duration_seconds` | Histogram | BoltDB repository operation latency (labels: `repo`, `method`, `status`) |

### Example Prometheus scrape config

```yaml
scrape_configs:
  - job_name: rebost
    static_configs:
      - targets: ['localhost:3805']
```

---

## Tracing

Rebost emits OpenTelemetry spans for service and volume operations. W3C `traceparent` headers are propagated across inter-node HTTP calls automatically. Spans are exported via OTLP HTTP when `--tracing.otlp-endpoint` is configured; if omitted, context still propagates but nothing is exported.

### Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `--tracing.otlp-endpoint` | _(empty)_ | OTLP HTTP collector endpoint, e.g. `localhost:4318`. Empty disables export. |

### Spans

| Span | Attributes |
|------|------------|
| `storing.CreateFile` | `key`, `replica` |
| `storing.GetFile` | `key` |
| `storing.DeleteFile` | `key` |
| `storing.HasFile` | `key` |
| `storing.StatFile` | `key` |
| `storing.CreateReplica` | `key` |
| `storing.UpdateFileReplica` | `key` |
| `storing.GetReplicaInfo` | `key` |
| `storing.Drain` | — |
| `storing.Ready` | — |
| `volume.CreateFile` | `key`, `volume_id` |
| `volume.GetFile` | `key`, `volume_id` |
| `volume.DeleteFile` | `key`, `volume_id` |
| `volume.StatFile` | `key`, `volume_id` |
| `volume.HasFile` | `key`, `volume_id` |
| `volume.SynchronizeReplicas` | `volume_id`, `removed_volume_id` |
| `volume.NextReplica` | `volume_id` |
| `volume.UpdateReplica` | `volume_id` |

### Example: Grafana Tempo + Grafana

```yaml
# docker-compose.yml
services:
  tempo:
    image: grafana/tempo:latest
    command: ["-config.file=/etc/tempo.yaml"]
    volumes:
      - ./tempo.yaml:/etc/tempo.yaml
    ports:
      - "4318:4318"   # OTLP HTTP receiver
      - "3200:3200"   # Tempo query API

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_DATASOURCES_DEFAULT_URL=http://tempo:3200
```

```yaml
# tempo.yaml
server:
  http_listen_port: 3200
distributor:
  receivers:
    otlp:
      protocols:
        http:
          endpoint: 0.0.0.0:4318
storage:
  trace:
    backend: local
    local:
      path: /tmp/tempo
```

Start Rebost pointing at Tempo:

```bash
rebost serve --name n1 --volumes /tmp/v1 --tracing.otlp-endpoint localhost:4318
rebost serve --name n2 --volumes /tmp/v2 --remote localhost:3805 --tracing.otlp-endpoint localhost:4318
```

Open Grafana at `http://localhost:3000`, add a Tempo datasource (`http://tempo:3200`), then use **Explore → Search** to find traces for service `rebost`.

---

## Known Limitations

| Feature                                   | Status                | Notes                                                                            |
| ----------------------------------------- | --------------------- | -------------------------------------------------------------------------------- |
| List objects (`GET /{bucket}`)            | `501 Not Implemented` | Rebost has no cluster-wide listing index. There is no way to enumerate all keys. |
| Multipart upload (`POST /{bucket}/{key}`) | `501 Not Implemented` | Use a single-part `PUT` for all uploads, regardless of file size.                |
| Object versioning                         | Not supported         | Uploading the same key twice overwrites the previous object.                     |
| Cross-region replication                  | N/A                   | Rebost has no concept of regions.                                                |
