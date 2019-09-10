[![Build Status](https://travis-ci.org/xescugc/rebost.svg?branch=master)](https://travis-ci.org/xescugc/rebost)

# Rebost (Beta)

Rebost is a Distributed Object Storage inspied by our experiance with MogileFS, MongoDB and ElasticSearch.

Rebost tries to simplify the management (deploy and operate) of an Object Storage by having an easy setup (barely no configuration required), by basically requiring just the address of one Node of the Rebost cluster (if not it'll start it's own)  and the path to the local Volumes (where the objects will be stored).
The implementation also simplifies the management of the cluster as there is no "Master", each Node is Master of his objects and also knows where the replicas of those are in the cluster. So adding a new Node it's just starting it and done. When a file is asked to a Node that does not know where it is, it'll ask it to the other Nodes.

## Example

For this example we'll have 3 directories on the current path: `v1`, `v2` and `v3`.

Create the first Node:

```bash
$> rebost serve --volumes v1
```

Create the second Node pointing (`--remote`) to the first one and changing the default `--port` as it's already in use (`3805`) for the first Node.

```bash
$> rebost serve --volumes v2 --port 3030 --remote http://localhost:3805
```

Do the same thing with the third Node.

```bash
$> rebost serve --volumes v4 --port 4040 --remote http://localhost:3805
```

After this the 3 Nodes will see each other and connect, for example you could run:

```bash
$> curl -T YOUR-FILE http://localhost:3805/files/your-file-name
```

Then you can go to your browser and check it (if it's an image) or:

```bash
$> curl http://localhost:3805/files/your-file-name
```

As the default replica is `3` all the Nodes we've created will have a copy of it so you could do the las comman (in fact any of the 2 before) to any Node.

## Beta?

Yes, there are a lot of things missing (most of them optimizations) that need to be implemented, for now it's an MVP to see if the idea made sense (which does hehe). Those changes will mostly be code-wise but some of them may also affect how the Nodes communicate and all those can be breaking changes until we reach v1.
