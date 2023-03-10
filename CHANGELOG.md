# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
