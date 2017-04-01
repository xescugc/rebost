# Ideas for a mogilefs replacement system

## Objective

The main objective of this project is to create a distributed file system inspired by our experience with mogilefs, mongodb and elasticsearch.

## Plan

In contrast to MogileFS, this system should be simple to deploy and operate. Starting with just one node and growth frome there like with elasticsearch. This means that every node on the system should be able to act as a full cluster of MogileFS. This means, nodes should be able to act as:

- Store: webdav interface plus an endpoint to show capacity information to other nodes (trackers)
- Tracker: This node should get info from others and instruct them on what to do to keep the concistency and maximize throghput.
- DB: What in MogileFS is a mysql cluster and "The source of truth" for trackers should be embedded on nodes.
- Proxy: a node talking to the outside and serving files from the cluster. Probably this is the only kind of node which makes sense to deploy without an active store role.

Our plan is to try to have only one tracker per cluster using raft or something similar for tracker election but in case of election every node can act as tracker. Stores will always accept and serve files when clients request so, they don't need a tracker to be elected to perform. This is a rough idea of the full lifecycle of files in the system:

1. A `client` talk to a `node[store]` to `PUT` a file on a `key` and a `class`. The class is known by all nodes and it indicates the replication factor. For this example, our replication factor will be 3.
2. The node[store] accept the file and store it on a temporary space. At this point the key and the temp-name will be stored on the DB with state `uploading`.
3. The file is complete and the client disconnect (when disconnect before completion it will be removed and removed from the DB)
4. Ideally, some kind of unique-checksum (sha-256?) is calculated while the file is being received. Or after reveiving it.
5. The checksum is splitted to create the file-path as a sparce directory to ensure there is no more than 999 files per directory. A file width checksum `aaabbbccc` will be stores as `[store-root]/aaa/bbb/ccc.fid`.
6. Once the file was moved to it's final destination it will be updated on the DB and the state will be `need-replication`. From this point on, the cluster is able to serve this file.
7. The tracker node will be ckecking all nodes periodically to detect failures and to get information from stores, so this node[store] will inform is has 1 file of the given class than need-replication with actual replication of `1`.
8. The tracker will ask for the file checksum and key
9. The tracker will check all other nodes to know if some has the same file and/or has other with the same key. In this case this checksum is not known by any other node.
10. The tracker, using capacity collected on every heartbit will choose 2 nodes[store] to copy the file.
11. Those 2 nodes will start copying the file placing it on its DB with state [replication-in] and the fist one will change state to [replication-out].
12. When each node finish replication will inform the origin node about it and state will change to `stored` with metadata having the other nodes having this file.
13. Origin node will change state to `stored` when replication count reach 3. At this point it will keep the others nodes as metadata and will inform the other nodes to copy/index this metadata.
14. When a client ask for a file to a node, it will check if the key is on its DB, in that case it will be served or an array of nodes having the file will be returned (both api calls will exists).
15. If the node doesn't have the file on it's DB it will ask to all or parts of the nodes till it knows where the file is or that the key is missing (and return an error like 404).
16. This info will be kept on an LRU disk cache that will be updated when timed-out, trackers inform abot deletions and/or replications updates.

Notes:

1. When a file is sent for a key already present on the system it will be accepted and the old one will be removed.
2. When 2 nodes receives different files with the same key, the tracker will kept the last one it get notice of replication. The tracker should deal with this kind of inconcistencies and the rest will always follow.
3. All tracker decicions will be logged in some way (and replicated to all nodes) so it can follows in case of restart and/or tracker re-election.
4. When lots of files need to be copied/moved, the tracker should handle this in small steps to prevent the system to be flood by it's own operations. Auto-rebalancing is decired but should be carefully designed to prevent this kind of problems.
5. When files are being moved as per a rebalancing operation, deletion will be done after copy and the store is able to mark the file with state `to-delete` and do it when it has I/O capacity.
 
## KV Storage options for node internal DB/s

- RocksDB: https://github.com/facebook/rocksdb
- BoltDB: https://github.com/boltdb/bolt
- LevelDB: https://github.com/syndtr/goleveldb
