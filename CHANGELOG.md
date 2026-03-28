# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- HRW (Highest Random Weight / Rendezvous) hashing for deterministic file placement: uploads land on the highest-scored `fnv64a(key+volumeID)` volume, reducing cold-cache lookups from O(n) broadcast to O(Replica) ranked checks. [Issue#206](https://github.com/xescugc/rebost/issues/206)
- New internal `HEAD /local/{key}` endpoint for local-only file presence checks, fixing a replication regression where `processNextReplica` and `processNextScrub` incorrectly believed every peer already had a file because `HEAD /{bucket}/{key}` (now `StatFile`) searches the whole cluster. `client.HasFile` now targets `/local/{key}` so replication and scrub correctly detect missing replicas. [Issue#183](https://github.com/xescugc/rebost/issues/183)
- Health/readiness HTTP endpoints: `GET /live` and `GET /health` (always 200), `GET /ready` (200 when running and all volumes healthy, 503 otherwise). All three bypass the readiness gate so they are reachable during startup and drain. [Issue#155](https://github.com/xescugc/rebost/issues/155)
- Node lifecycle status (`starting` | `running` | `draining`): visible in the dashboard as a badge per node and propagated via gossip so peers skip routing to non-running nodes. [Issue#194](https://github.com/xescugc/rebost/issues/194)
- Node tags: arbitrary `--tag key=value` labels (repeatable) visible in the dashboard as badges and broadcast via gossip for future placement constraints. [Issue#137](https://github.com/xescugc/rebost/issues/137)
- Replica consistency check (`loopConsistencyCheck`): non-owner replicas periodically ask the file owner whether they are still listed in VolumeIDs and silently purge stale local copies if not. Configurable via `--timing.replica-consistency-interval` (default `1h`). [Issue#129](https://github.com/xescugc/rebost/issues/129)

### Changed

- Sort replication candidate nodes by descending free space so replicas prefer nodes with the most headroom [Issue#15](https://github.com/xescugc/rebost/issues/15)

- Timing configuration flags grouped under `timing.*` prefix (`--timing.volume-downtime`, `--timing.scrub-interval`, `--timing.replica-check-interval`, `--timing.replica-consistency-interval`). Config struct now has a `Timing` sub-struct. [Issue#129](https://github.com/xescugc/rebost/issues/129)

- Replica reconciliation on startup: clears stale replica queue and rebuilds from file state after a crash [Issue#129](https://github.com/xescugc/rebost/issues/129)
- Background replica-check loop (`loopReplicaCheck`) that periodically detects and re-queues under-replicated files. Configurable via `--replica-check-interval` (default `1h`). [Issue#129](https://github.com/xescugc/rebost/issues/129)
- Periodic scrubbing: background loop re-verifies file checksums on each volume and auto-repairs corrupt files by fetching a good copy from a remote replica. Configurable via `--scrub-interval` (default `24h`).
  [Issue#130](https://github.com/xescugc/rebost/issues/130)
- Graceful node drain via SIGQUIT: replicates locally-held files to peers, purges local copies, and leaves the cluster before shutdown.
  [Issue#17](https://github.com/xescugc/rebost/issues/17)
- Proxy node support: start a node without `--volumes` to forward all operations to peers.
  [Issue#118](https://github.com/xescugc/rebost/issues/118)
- Storage capacity fallback: when a volume is full, `CreateFile` automatically
  falls back to another local volume, then a remote cluster node.
  [Issue#36](https://github.com/xescugc/rebost/issues/36)

### Fixed

- HEAD handler now correctly finds files on remote nodes, matching GET behaviour; eliminates redundant double-lookup [Issue#183](https://github.com/xescugc/rebost/issues/183)

## Changed

- All the endpoints are now S3 compatible
  [Issue#89](https://github.com/xescugc/rebost/issues/89)

## [0.4.0] - 2026-03-20

### Added

- TTL to the files so they can have an expiration date
  [Issue#71](https://github.com/xescugc/rebost/issues/71)

## Updated

- Refactored the UoW to make it more generic and not require to specify the registry, now it just needs to be initialized with the database and the bucket name and it will work for any bucket
  [Issue#103](https://github.com/xescugc/rebost/issues/103)

### Fixed

- Restarting an app from 0 does no longer tirgger a Reset
  [Issue#102](https://github.com/xescugc/rebost/issues/102)
- Stale replica jobs are now cleaned up when the source file no longer exists, preventing infinite retry loops
  [Issue#100](https://github.com/xescugc/rebost/issues/100)
- Replica deltion was not working properly and was leaving the file in a "deleting" state
  [PR#97](https://github.com/xescugc/rebost/pull/97)

## [0.3.0] - 2023-03-31

### Added

- Client can be initialized with multiple hosts and will request one at a time in order as a load balancer
  [Issue#16](https://github.com/xescugc/rebost/issues/16)
- Visualization of the size of the Nodes (used and total) and the size of the cluster (use and total) to the Dashboard
  [Issue#49](https://github.com/xescugc/rebost/issues/49)
- `--volume-downtime` flag to set a custom time to start replicating after a Volumes goes down as we start replicating
  [Issue#56](https://github.com/xescugc/rebost/issues/56)

### Fixed

- Initializing with a volume with size was causing an error
  [PR#52](https://github.com/xescugc/rebost/pull/52)
- Error check on volume goroutine to recalculate size
  [PR#53](https://github.com/xescugc/rebost/pull/53)
- No longer trying to replicate a file with a node with a replica of the file
  [Issue#61](https://github.com/xescugc/rebost/issues/61)

## [0.2.0] - 2023-03-11

### Added

- Cache(LRU) to the logic to fetch an object form another node so we don't have to search for it again once we found it once
  [Issue#35](https://github.com/xescugc/rebost/issues/35)
- Volume fixed size, not the initialization of a volume can have `--vomue /:10G` to fix a maximum size to use
  [Issue#33](https://github.com/xescugc/rebost/issues/33)

## Updated

- Migrated from 'boltdb/bolt' to 'go.etcd.io/bbolt' and also updated all the dependencies [Issue#10](https://github.com/xescugc/rebost/issues/10)
- If the name of the Node is not defined the random one is now human readable instead of random alphanumeric we had [Issue#12](https://github.com/xescugc/rebost/issues/12)
- Changed the `--memberlist-bind-port` to `--memberlist.port` and `--memberlist-name` to `--name` [Issue#41](https://github.com/xescugc/rebost/issues/41)

## [0.1.0] - 2023-02-24

### Added

- First basic implementation of Rebost
- Implemented Replica logic [PR#4](https://github.com/xescugc/rebost/pull/4)
- Changed fmt.Println for go-kit log [PR#6](https://github.com/xescugc/rebost/pull/6)
- New Dashboard service [Issue#9](https://github.com/xescugc/rebost/issues/9)
- The CHANGELOG file [Issue#11](https://github.com/xescugc/rebost/issues/11)
- Version to the cmd [Issue#24](https://github.com/xescugc/rebost/issues/24)

### Changed

- From TravisCI to GitHub Actions (Test && Docker)[Issue#14](https://github.com/xescugc/rebost/issues/14)
