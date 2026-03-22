# Rebost — Claude Code Context

## Project Overview

Rebost is a distributed, peer-to-peer object storage system (inspired by MogileFS, MongoDB, ElasticSearch). There is **no master node** — each node is master of its own objects and knows where replicas live. Nodes discover each other via a gossip protocol (hashicorp/memberlist). Adding a node is just starting it.

Default replica count: 3. Files are content-addressed via SHA1 (deduplication). Multiple keys can point to the same content.

## Build & Test Commands

```bash
make test         # go test ./...
make staticcheck  # go tool staticcheck ./...
make generate     # go generate ./... (regenerates mocks)
make ci           # staticcheck + test
```

## Package Map

| Package       | Purpose                                                                                                    |
| ------------- | ---------------------------------------------------------------------------------------------------------- |
| `cmd/`        | CLI entry (cobra/viper). `serve.go` wires everything together.                                             |
| `storing/`    | Main service layer — Service interface, background replication loops, membership interface.                |
| `storing/transport/http/` | HTTP transport — S3-compatible handlers, auth middleware, XML error responses.      |
| `volume/`     | Local volume ops — file CRUD, replica queue, TTL loop. Wraps BoltDB + filesystem.                          |
| `membership/` | Cluster discovery via gossip. Tracks nodes, volume IDs, downtime.                                          |
| `client/`     | HTTP client for remote nodes; implements `volume.Volume` interface.                                        |
| `boltdb/`     | BoltDB implementations of all repositories + Unit of Work transaction manager.                             |
| `uow/`        | Unit of Work interface + `StartUnitOfWork` function type.                                                  |
| `fs/`         | Filesystem-level UoW tracker (afero). Tracks Create/Remove/Rename for rollback.                            |
| `file/`       | `File` data model (keys, signature, replicas, TTL, size).                                                  |
| `replica/`    | `Replica` data model (pending replication job).                                                            |
| `state/`      | `State` data model (volume disk usage, downtime tracking).                                                 |
| `idxkey/`     | Index: user key → file signature.                                                                          |
| `idxttl/`     | Index: expiration time → file signatures.                                                                  |
| `idxvolume/`  | Index: volume ID → file signatures (tracks remote copies).                                                 |
| `config/`     | Config struct, viper-backed.                                                                               |
| `dashboard/`  | Web UI — lists nodes and volume states.                                                                    |
| `mock/`       | Auto-generated mocks (golang/mock). Never edit by hand.                                                    |

## Architecture

```
HTTP Request
  → storing/transport/http  (S3-compatible handlers, gorilla/mux)
  → storing.Service         (business logic — local or remote dispatch)
      → volume.Local        (local volumes: BoltDB + afero filesystem)
          → UnitOfWork      (boltdb tx wrapping all repositories + fs)
      → client.Client       (remote volumes via HTTP)
  → membership              (gossip cluster: node/volume discovery)
```

### HTTP Routes (storing/transport/http/transport.go)

**S3-compatible object routes** (XML responses, auth-protected when credentials configured):

- `PUT    /{bucket}/{key}` — PutObject (CreateFile internally; key = `bucket/key`)
- `GET    /{bucket}/{key}` — GetObject
- `DELETE /{bucket}/{key}` — DeleteObject
- `HEAD   /{bucket}/{key}` — HeadObject
- `POST   /{bucket}/{key}` — Multipart stub → 501
- `PUT    /{bucket}` — CreateBucket (no-op, bucket-as-prefix)
- `DELETE /{bucket}` — DeleteBucket (no-op)
- `GET    /{bucket}` — ListObjects → 501 (no cluster-wide list index)

**Internal inter-node routes** (JSON responses, always bypass auth):

- `PUT    /replicas/{key}` — CreateReplica
- `PATCH  /replicas/{key}` — UpdateFileReplica
- `GET    /config` — node config

**Bucket-as-prefix:** `PUT /mybucket/photo.jpg` maps to internal key `mybucket/photo.jpg`. No bucket state is stored. Gorilla/mux registration order ensures `/replicas/` and `/config` are matched before `/{bucket}/{key:.*}`.

## Key Design Patterns

### HTTP handler pattern

Each service has `service.go` (interface + implementation) and `transport/http/transport.go` (HTTP handlers). Handlers are separate top-level functions returning `http.HandlerFunc`, taking the service as argument. Logging uses `log/slog`. The `client/` package makes direct `net/http` calls with a `request()` helper for JSON endpoints.

### Unit of Work (UoW) / Transaction

`uow.StartUnitOfWork` is a function type passed around. BoltDB transactions wrap all repository operations atomically. `fs.UOWWithFs` adds filesystem rollback on top. Start a UoW with all needed repositories as arguments; commit on success, rollback on error.

### Mock generation

All interfaces have mocks in `mock/`. Regenerate with `make generate`. Never edit mock files directly. The `go:generate` directives live in the interface source files.

### File deduplication

Files are stored at `{fileDir}/XX/YY/XXYY{rest}` where the path is derived from the SHA1 signature. Multiple user-facing keys (`.Keys`) can point to the same `File` record. Deleting a key only removes the key; the file is deleted when the last key is removed.

**`file.File.VolumeIDs`** tracks **all** volume IDs that hold a copy of this file's content, including the local volume's own ID. The local ID is appended in `volume/volume.go` `createFile` via `f.VolumeIDs = append(f.VolumeIDs, l.ID())`. When building a list of _remote_ targets, always filter out `l.ID()` first.

**`file.File.VolumeIDs` ordering is significant:** `VolumeIDs[0]` is the file's owner/master — the volume responsible for ensuring replication. When creating a file locally, the local volume ID is appended first (see `volume/volume.go` `createFile`). When choosing which node should create new Replica jobs, always check `VolumeIDs[0]`. See `SynchronizeReplicas` (volume/volume.go) and the background queue pattern.

### Distributed philosophy — no broadcasts

Never enumerate `Nodes()` to find who might have a file. Use `file.File.VolumeIDs` to know **exactly** which volumes have a copy. `GetNodeWithVolumeByID(vid)` resolves a volumeID to the owning node. This is the only correct way to target remote operations.

### Background queue pattern (`storing/replica.go` → `loopVolumes`)

All async cross-node operations follow the same pattern:

1. A local operation (e.g. `DeleteFile`, `CreateFile`) writes a job to a BoltDB queue inside the UoW.
2. `loopVolumes` polls all local volumes each tick; calls `processNextXxx(v)` for each job type.
3. The processor resolves VolumeIDs → nodes via `GetNodeWithVolumeByID`, performs the HTTP call, then removes the job.

To add a new background cross-node operation: create a `deletion/`-style package, add a BoltDB bucket + Repository, wire into UoW, push jobs from the local operation, add a `processNextXxx` helper in `storing/replica.go`.

**`deleteFile` (internal, `volume/volume.go`) is local-only** — it removes the file from the current volume only. It pushes remote VolumeIDs to the deletion queue but does not cascade directly. `storing.Service.DeleteFile` handles the async propagation via `loopVolumes`.

### LRU cache in `storing.Service`

`service.cache` is an LRU ARC cache mapping file key → remote volumeID. It avoids repeated `HasFile` queries to peers. Only remote volumeIDs are cached (local volumes are cheap to query directly). **Must be invalidated (`cache.Remove(k)`) when a file is deleted**, otherwise stale entries cause `HasFile` to return a volumeID for files that no longer exist.

### Recovery after node down (`loopRemovedVolumeDIs`)

Waits `VolumeDowntime` (default 2 min) before re-replicating files that were on the downed node. Uses `SynchronizeReplicas()` on the first surviving volume.

### TTL expiration (`volume/ttl.go`)

`loopTTL()` runs every second per volume. Filters expired `idxttl` entries, deletes files and their keys, also cleans up replica queue entries.

## Inter-Node Communication

Node communication uses two distinct layers:

### Gossip Layer (hashicorp/memberlist)

Used for cluster topology only — no file data travels over gossip.

**Node metadata** (`membership/metadata.go`) — sent once on join/update:

- HTTP service port (so peers can build a `client.Client`)
- Set of volume IDs this node owns

**Node state** (`membership/state.go`) — pushed periodically during gossip rounds:

- Per-volume disk usage (`SystemTotalSize`, `SystemUsedSize`, `VolumeTotalSize`, `VolumeUsedSize`)
- `UpdatedAt` heartbeat timestamp (used to detect downtime)

**Events** (`membership/event_delegate.go`):

- `NotifyJoin` — unpacks metadata, creates `client.Client("host:port")`, stores in `nodes` map, clears any pending removed-volume entries for that node.
- `NotifyLeave` — stamps all of that node's volume IDs in `removedVolumeIDs` with the current time; triggers re-replication after `VolumeDowntime` (default 2 min).
- `NotifyUpdate` — currently delegates to NotifyJoin.

### HTTP Layer (client/client.go)

All actual file operations between nodes go over HTTP using the S3-compatible API. Since keys already contain the bucket prefix (e.g. `mybucket/photo.jpg`), the inter-node URLs are simply `/{key}`:

| Operation           | Method | Route             | Usage                                 |
| ------------------- | ------ | ----------------- | ------------------------------------- |
| `HasFile`           | HEAD   | `/{key}`          | Check before replicating              |
| `GetFile`           | GET    | `/{key}`          | Fetch from remote node                |
| `CreateReplica`     | PUT    | `/replicas/{key}` | Push replica to remote node           |
| `UpdateFileReplica` | PATCH  | `/replicas/{key}` | Notify peers of new replica locations |
| `DeleteFile`        | DELETE | `/{key}`          | Remove from remote node               |
| `Config`            | GET    | `/config`         | Fetch memberlist port when joining    |

### Membership Interface (storing/membership.go)

`storing.Service` only talks to membership through this interface:

```go
Nodes() []*client.Client                           // all peers
NodesWithoutVolumeIDs(vids []string) []*client.Client  // peers missing a file
LocalVolumes() []volume.Local
GetNodeWithVolumeByID(vid string) (*client.Client, error)
GetNodeState(nn string) (*membership.State, error)
RemovedVolumeIDs() []string                        // volumes ready to re-replicate
Leave()
```

### Cluster Bootstrap

On startup, if `--remote` is provided:

1. Node fetches `/config` from the remote to get its memberlist port.
2. Calls `memberlist.Join` with that address — memberlist handles full state sync automatically.

## BoltDB Buckets

One DB file per volume path. Buckets: `files`, `idxkey`, `idxttl`, `idxvolume`, `replica`, `deletion`, `state`.

## Development Conventions

### CHANGELOG format

Always add a `CHANGELOG.md` entry under `## [Unreleased]` for every feature or fix.
Format: `### Added` / `### Changed` / `### Fixed` entries, each line:
`- Description [Issue#NNN](https://github.com/xescugc/rebost/issues/NNN)` or `[PR#NNN](url)`

### Documentation

When adding or changing user-facing behavior (CLI flags, HTTP routes, operational
procedures), also update `doc/docs.md`.

## Current Branch: fg-17

Modified files and intent:

- `membership/metadata.go` — Added `Draining bool` for fast gossip propagation of draining state.
- `membership/membership.go` — Added `draining atomic.Bool`, `SetDraining()`, draining filters in `Nodes()`, `NodesWithoutVolumeIDs()`, `NodesWithCapacity()`.
- `membership/delegate.go` — `NodeMeta()` includes `Draining` state.
- `storing/membership.go` — Added `SetDraining(draining bool)` to interface.
- `storing/service.go` — Added `draining atomic.Bool`; `CreateFile`/`CreateReplica` reject when draining; `Drain()` sets draining at start.
- `volume/volume.go` — `PrepareForDrain` only creates Replica jobs when `VolumeIDs[0] == l.id` (owner check).
- `mock/membership.go` — Regenerated to include `SetDraining`.
- `CLAUDE.md` — Documented `VolumeIDs[0]` ownership rule.

