[![Go](https://github.com/xescugc/rebost/actions/workflows/go.yml/badge.svg)](https://github.com/xescugc/rebost/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/xescugc/rebost.svg)](https://pkg.go.dev/github.com/xescugc/rebost)

<p align="center">
  <img src="doc/11-triangle-mesh.svg" alt="Rebost" width="340" />
</p>

# Rebost (Beta)

**[Full Documentation →](doc/docs.md)**

Rebost is a Distributed Object Storage inspired by our experience with MogileFS, MongoDB and ElasticSearch.

Rebost tries to simplify the management (deploy and operate) of an Object Storage by having an easy setup (barely no configuration required), by basically requiring just the address of one Node of the Rebost cluster (if not it'll start it's own) and the path to the local Volumes (where the objects will be stored).
The implementation also simplifies the management of the cluster as there is no "Master", each Node is Master of his objects and also knows where the replicas of those are in the cluster. So adding a new Node it's just starting it and done. When a file is asked to a Node that does not know where it is, it'll ask it to the other Nodes.

## Example

For this example we'll have 3 directories on the current path: `v1`, `v2` and `v3`.

First create a Docker network so the nodes can reach each other by name:

```bash
docker network create rebost
```

Create the first Node:

```bash
docker run -d --name node1 --network rebost \
  -p 3805:3805 -p 3806:3806 \
  -v $(pwd)/v1:/data \
  xescugc/rebost serve --volumes /data --name node1
```

Create the second Node pointing (`--remote`) to the first one and changing the default `--port` as it's already in use (`3805`) for the first Node:

```bash
docker run -d --name node2 --network rebost \
  -p 2020:2020 \
  -v $(pwd)/v2:/data \
  xescugc/rebost serve --volumes /data --port 2020 --name node2 \
    --remote http://node1:3805 --dashboard.enabled false
```

Do the same thing with the third Node:

```bash
docker run -d --name node3 --network rebost \
  -p 3030:3030 \
  -v $(pwd)/v3:/data \
  xescugc/rebost serve --volumes /data --port 3030 --name node3 \
    --remote http://node1:3805 --dashboard.enabled false
```

After this the 3 Nodes will see each other and connect, for example you could run:

```bash
curl -T YOUR-FILE http://localhost:3805/mybucket/your-file-name
```

Then you can go to your browser and check it (if it's an image) or:

```bash
curl http://localhost:3805/mybucket/your-file-name
```

As the default replica is `3` all the Nodes we've created will have a copy of it so you could do the last command (in fact any of the 2 before) to any Node.

To access the dashboard go to http://localhost:3806, we disabled the Dashboard on the other nodes just because we would need to change also the port, and for a simple example would look too verbose.

## S3 Compatibility

Rebost exposes an S3-compatible API, so any AWS S3 library or tool can be used without a dedicated client. The URL structure follows the S3 path-style convention: `/{bucket}/{key}`.

Buckets are treated as key prefixes — `PUT /mybucket/photo.jpg` stores the object with the internal key `mybucket/photo.jpg`. No explicit bucket creation is required, but `PUT /{bucket}` and `DELETE /{bucket}` are accepted as no-ops for client compatibility.

### Using the AWS CLI

```bash
# Upload a file (no auth)
aws s3 cp ./photo.jpg s3://mybucket/photo.jpg \
  --endpoint-url http://localhost:3805 \
  --no-sign-request

# Download a file
aws s3 cp s3://mybucket/photo.jpg ./photo.jpg \
  --endpoint-url http://localhost:3805 \
  --no-sign-request

# Delete a file
aws s3 rm s3://mybucket/photo.jpg \
  --endpoint-url http://localhost:3805 \
  --no-sign-request
```

### Authentication

Authentication is optional and configured per node. Nodes with `--s3.access_key` and `--s3.secret_key` set will require AWS Signature V4 on all requests (except internal inter-node routes). Nodes without credentials set accept all requests — useful for storage nodes that are not directly internet-accessible.

```bash
# Start a node with auth enabled
docker run -d -p 3805:3805 -v $(pwd)/v1:/data \
  xescugc/rebost serve --volumes /data \
    --s3.access_key mykey --s3.secret_key mysecret

# Use with AWS CLI
AWS_ACCESS_KEY_ID=mykey AWS_SECRET_ACCESS_KEY=mysecret \
  aws s3 cp ./photo.jpg s3://mybucket/photo.jpg \
  --endpoint-url http://localhost:3805
```

### Known Limitations

- **List objects** (`GET /{bucket}`) — returns `501 NotImplemented`. Rebost has no cluster-wide listing index.
- **Multipart upload** — returns `501 NotImplemented`. Use single-part `PUT` for all uploads.

## Beta?

Yes, there are a lot of things missing (most of them optimizations) that need to be implemented, for now it's an MVP to see if the idea made sense (which does hehe). Those changes will mostly be code-wise but some of them may also affect how the Nodes communicate and all those can be breaking changes until we reach v1.
