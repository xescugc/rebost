package membership

// Metadata is the metadata that it's send for each node
type Metadata struct {
	// Port is the port in which the storing.Service is
	// raised, needed to connect to the client
	Port int `json:"port"`

	// Volumes list of all VolumeIDs of the Node
	Volumes map[string]struct{} `json:"volumes"`

	// Draining indicates that the node is draining and should not
	// receive new writes from peers
	Draining bool `json:"draining"`
}
