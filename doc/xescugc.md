# Gogilefs

## Objective

The objective is to write a Distributed filesystem inspired in MogileFS

## Implementation

The main idea behind it is to be EASY to setup (barely no configuration needed). We plan to implement a Leader/Follower distribution for the Nodes, but in this case the Leaders is a week leader. Each Node has a KV store of the images it knows (not the full DB) and where they are replicated.

Each Object(stored) has a Class/Type/? that defines the replication.

Every Node can serve the Objects without comunication with the Leader.

## Configuration

Te basic configuration is a .gogilefs.(json|yaml|xml) file located by default: __BLANK__ which can have this keys:

* storage: Array of locations in which Gogilefs will sotre the Objects
* name: Name of the cluster
* node_name: Canonical name for the node (readable logs)
* nodes: Array with a list of some of the nodes of the cluster
* classes/types: Map with the 'keys' beeing the names of the class and the values the reaplication.

*Example:*

```json
  {
    storage: ['/data/'],
    name: 'Pepito',
    node_name: 'Palotes',
    nodes: ['127.0.0.1:5000'],
    classes: {
      original: 4,
      thumbnail: 2,
    }
  }
```

## Objects Stored

Object can be anithing, from images to videso to anithing. The way we store them is making a SHAXXX and with the SHA key of length 40 we create subfolders for every X numbers (40/4=10 subfolders)

## Node (follower) Role

A simmple Node by itself can store Objects and Serve Objects to the client.

### Store

When a Object needs to be stored:

* First is stores the object in a `tmp/` location (in case of crashing the server)
* Then it's copied to the location and removed from the `tmp/`
* Finally stores the SHA key to the KV store

If the image needs to be replicated then the Node, in the next heartbeat will comunicate te pending replications.

### Serve
### Status

## Leader Role
### Replication
### Leader Electino


#TODO

* Master replication is a RPC to itself?
