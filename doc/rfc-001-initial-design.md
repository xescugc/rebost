# RFC — Pre-Implementation Design Documents

Two independent design proposals written before the project was built. The final implementation diverged from both (no leader/follower, pure gossip instead of Raft), but these capture the original thinking.

## Index

- [RFC 1 — xescugc](#rfc-1--xescugc)
  - [Objective](#objective)
  - [Implementation](#implementation)
  - [Configuration](#configuration)
  - [Objects Stored](#objects-stored)
  - [Node (follower) Role](#node-follower-role)
    - [Store](#store)
    - [Serve](#serve)
    - [Status](#status)
    - [Communication](#communication)
    - [Replication](#replication)
  - [Leader Role](#leader-role)
    - [Replication](#replication-1)
    - [Leader Election](#leader-election)
    - [New Node](#new-node)
- [RFC 2 — diegok](#rfc-2--diegok)
  - [Objective](#objective-1)
  - [Plan](#plan)
    - [When a node stops working](#when-a-node-stops-working)
    - [When a leader can't be elected](#when-a-leader-cant-be-elected)
  - [KV Storage options for the internal node DB](#kv-storage-options-for-the-internal-node-db)

---

## RFC 1 — xescugc

*Original file: `xescugc.md`*

### Objective

The objective is to write a Distributed filesystem inspired by MogileFS.

### Implementation

The main idea behind it is to be EASY to setup (barely no configuration needed). We plan to implement a Leader/Follower distribution for the Nodes, but in this case the Leader is a weak Leader. Each Node has a KV store of the Objects it knows (not the full DB) and where they are replicated.

Each Object has a Class/Type that defines the replication.

Every Node can serve the Objects without communication with the Leader.

On the first implementation, the communication between Nodes will be over HTTPS, and later via RPC.

The Followers will not have any direct request to the Leader; they wait until the next heartbeat to inform the Leader of anything that needs to be communicated.

### Configuration

The basic configuration is a `.gogilefs.(json|yaml|xml)` file which can have these keys:

- `storage`: Array of locations in which Gogilefs will store the Objects
- `name`: Name of the cluster
- `node_name`: Canonical name for the node (readable logs)
- `nodes`: Array with a list of some of the nodes of the cluster
- `classes/types`: Map with the keys being the names of the class and the values the replication factor
- much more...

### Objects Stored

An Object can be anything — images, videos, etc. The way we store them is by computing a SHA hash, and with the resulting 40-character key we create subfolders for every N characters (e.g. `40/4 = 10` subfolders).

### Node (follower) Role

A simple Node by itself can store Objects, serve Objects to the client, and obey orders (replication, etc.).

Each Node has an internal KV where it saves the Objects and the replicas on other servers.

Each Node has another internal DB/KV/State Machine to store current jobs, replication status, candidate state, current term, etc.

Each Node has an LRU Cache to store locations of unknown Objects.

#### Store

When an Object needs to be stored:

1. First it stores the object in a `tmp/` location (in case of a crash)
2. Then it is copied to the final location and removed from `tmp/`
3. Finally it stores the SHA key in the KV store

If the Object needs to be replicated, the Node will communicate the pending replications to the Leader on the next heartbeat.

The response to the client can come after the Object is saved on the Node, or — if configured — after `w` replicas acknowledge the write (similar to MongoDB's `w` and `j` options).

#### Serve

If the Object is in the Node, it serves it directly.

If the Object is not in the Node's KV, it asks a "nearby" Node for the Object (and consecutively, if that node doesn't know it either, it asks another, passing along the list of nodes already asked to avoid loops). Once a Node has the Object it responds, all "bypass" nodes cache the key→location mapping. The first Node then proxies the Object from the owner to the Client.

> **Note:** If the cluster is very large, should we stop the "ask nearby node" policy after querying a majority without a match?

#### Status

The node can be in one of these states:

- `Leader`
- `Follower`
- `Candidate`
- `Draining`

This status is persisted to disk so that a restarted node knows what it was doing.

##### Draining

The `Draining` state means the server is about to shut down. All Objects must be replicated elsewhere, and all Nodes must be informed to remove the draining Node from their lists. If no Object on this node lacks a replica elsewhere, it can be shut down safely.

#### Communication

Each heartbeat from the Leader, Followers may respond with:

- IO status
- Node status
- Pending replications
- The answer to any request the heartbeat carried

#### Replication

When a Node receives the order to replicate an Object FROM Node A TO Node B, Node B requests the Object from Node A (with an identifier to prevent duplicate/invalid replications). When replication is complete, the receiver (Node B) notifies all other nodes that the Object has been successfully replicated, so they update their metadata to record that Node B also holds a copy.

### Leader Role

A cluster MUST have exactly ONE Leader.

The main job of the Leader is to track the stats of Followers to trigger replication and storage rebalancing (if one Node is at XX% capacity, the Leader can decide to move some Objects to a less-full Node).

It also communicates to all Nodes when another Node goes down, so that affected Objects can be re-replicated.

#### Replication

When a Follower tells the Leader it has an Object that needs replication, the Leader decides (based on cluster stats) which Node will take it. It then instructs both nodes that Node A must replicate to Node B, and from that point the Nodes handle the transfer themselves.

If more than 2 replicas are needed, all Nodes must record the location of every Node that ends up holding the Object once replication is complete.

> **Note:** If a global KV exists, the master can know the Object is already on a Node and simply communicate that.
> **Note:** Without a global KV, the master must ask Followers. This is relevant when 2 identical Objects are uploaded simultaneously.

#### Leader Election

For leader election, Rebost adapts the approach that Raft follows, minus the Raft log:

> *(Excerpt from the Raft paper)*
>
> Raft uses a heartbeat mechanism to trigger leader election. When servers start up, they begin as followers. A server remains in follower state as long as it receives valid RPCs from a leader or candidate. Leaders send periodic heartbeats to all followers to maintain their authority. If a follower receives no communication over a period of time called the election timeout, then it assumes there is no viable leader and begins an election to choose a new leader.
>
> To begin an election, a follower increments its current term and transitions to candidate state. It then votes for itself and issues RequestVote RPCs in parallel to each of the other servers in the cluster. A candidate continues in this state until one of three things happens: (a) it wins the election, (b) another server establishes itself as leader, or (c) a period of time goes by with no winner.
>
> Raft uses randomized election timeouts to ensure that split votes are rare and resolved quickly.

In summary:

- The Leader sends heartbeats every 0.5–20 ms
- Each Node has a random election timeout of 10–500 ms
- If a Node times out, it enters Candidate mode: votes for itself, asks others to vote for it, and increments the term
- Each Node votes for only one Candidate per term
- To win, a Node must receive the majority of votes (e.g. 3 of 5 in a 5-node cluster)
- If a Candidate receives a heartbeat from a Node claiming to be Leader with a term ≥ its own, it returns to Follower state
- If no one wins, all Candidates restart the election after a random timeout, incrementing the term again

#### New Node

When a new Node joins the cluster, it enters as a Follower and the Leader announces its presence to all other Nodes.

The Leader then begins replicating Objects to the new Node to rebalance the cluster.

---

## RFC 2 — diegok

*Original file: `diegok.md`*

### Objective

The main objective is to create a distributed file system inspired by our experience with MogileFS, MongoDB, and Elasticsearch.

### Plan

In contrast to MogileFS, this system should be simple to deploy and operate — start with just one node and grow from there, like Elasticsearch. Every node should be able to act as a full cluster of MogileFS, meaning nodes can take on the following roles:

- **Store:** WebDAV interface plus an endpoint to expose capacity information to other nodes (trackers)
- **Tracker:** Collects info from others and instructs them on what to do to maintain consistency and maximize throughput
- **DB:** What in MogileFS is a MySQL cluster — "the source of truth" for trackers — should be embedded in each node
- **Proxy:** A node that talks to the outside and serves files from the cluster. Probably the only role that makes sense without an active store role

The plan is to have only one tracker per cluster (using Raft or similar for election), but every node can act as tracker if needed. Stores will always accept and serve files without waiting for a tracker to be elected.

Rough lifecycle of a file:

1. A `client` talks to a `node[store]` to `PUT` a file under a `key` and a `class`. The class defines the replication factor (e.g. 3).
2. The `node[store]` accepts the file and stores it in a temporary space. The key and temp-name are written to the DB with state `uploading`.
3. The file transfer completes. If the client disconnects before completion, the file and DB entry are removed.
4. A unique checksum (SHA-256) is computed while receiving, or after.
5. The checksum is split to create a sparse directory path, ensuring no more than 999 files per directory. A file with checksum `aaabbbccc` is stored as `[store-root]/aaa/bbb/ccc.fid`.
6. Once moved to its final location, the DB state is updated to `need-replication`. The file is now serveable.
7. The tracker checks all nodes periodically. This node reports it has 1 file of that class with `need-replication` and actual replication of `1`.
8. The tracker asks for the file checksum and key.
9. The tracker checks all other nodes for the same checksum or key. In this case no other node knows it.
10. Using capacity data collected on each heartbeat, the tracker selects 2 nodes to replicate the file to.
11. Those 2 nodes start copying the file. Each sets state to `replication-in`; the origin sets state to `replication-out`.
12. When each node finishes replication it informs the origin, which updates its state. The receiving node notifies all other nodes that it now holds a copy.
13. The origin sets state to `stored` when replica count reaches 3. It retains the other nodes as metadata and instructs them to index this metadata.
14. When a client requests a file from a node, the node checks its DB. If it has the key, it serves it or returns the list of nodes that hold it (both API variants exist).
15. If the node doesn't have the key in its DB, it queries all or a subset of nodes until it finds the file or concludes the key is missing (404).
16. This info is kept in an LRU disk cache, updated when timed out or when the tracker reports deletions or replication changes.

**Notes:**

1. When a file is uploaded for a key already present, it is accepted and the old file is removed.
2. When 2 nodes receive different files with the same key, the tracker keeps the last one it receives a replication notice for. The tracker is the authority on such inconsistencies.
3. All tracker decisions are logged and replicated to all nodes so they can be replayed after a restart or re-election.
4. When many files need to be copied/moved, the tracker should process in small steps to avoid flooding the cluster with its own operations. Auto-rebalancing is desirable but must be carefully designed.
5. When files are being moved for rebalancing, deletion happens after the copy. Stores can mark files with state `to-delete` and act on it when I/O capacity allows.

#### When a node stops working

If the leader doesn't receive heartbeat responses from a node after a threshold, it marks the node as `DOWN`. It informs all other nodes, and each computes which keys it shared with the downed node and sets their state to `need-replication`. Over the next heartbeat cycles the leader collects this information and uses the normal replication mechanism to heal the cluster, always prioritizing files with fewer replicas.

#### When a leader can't be elected

If a leader can't be voted in, the cluster freezes — no replication happens — but every node continues to serve known and reachable files. This should be well documented, as it reflects the project's philosophy of favouring throughput and durability over strict consistency.

### KV Storage options for the internal node DB

- **RocksDB:** https://github.com/facebook/rocksdb
- **BoltDB:** https://github.com/boltdb/bolt *(chosen in the final implementation)*
- **LevelDB:** https://github.com/syndtr/goleveldb
