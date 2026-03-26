package membership

// Metadata is the metadata that it's send for each node
type Metadata struct {
	// Port is the port in which the storing.Service is
	// raised, needed to connect to the client
	Port int `json:"port"`

	// Volumes list of all VolumeIDs of the Node
	Volumes map[string]struct{} `json:"volumes"`

	// Status is the lifecycle state of the node
	Status Status `json:"status"`

	// Tags is a map of arbitrary key=value labels for this node,
	// set at startup and propagated via gossip.
	Tags map[string]string `json:"tags"`
}
