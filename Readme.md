[![Build Status](https://travis-ci.org/xescugc/rebost.svg?branch=master)](https://travis-ci.org/xescugc/rebost)

# Rebost (Beta)

Rebost is a Distributed Object Storage inspied by our experiance with MogileFS, MongoDB and ElasticSearch.

Rebost tries to simplify the management (deploy and operate) of an Object Storage by having an easy setup (barely no configuration required), by basically requiring just the address of one Node of the Rebost cluster (if not it'll start it's own)  and the path to the local Volumes (where the objects will be stored).
The implementation also simplifies the management of the cluster as there is no "Master", each Node is Master of his objects and also knows where the replicas of those are in the cluster. When a file is asked to a Node that does not know where it is, it'll ask it to the other NNodes.
