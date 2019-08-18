package membership

// metadata is the metadata that it's send for each node
type metadata struct {
	// Port is the port in which the storing.Service is
	// raised, needed to connect to the client
	Port int `json:"port"`

	// VolumeIDs list of all VolumeIDs of the Node
	VolumeIDs []string `json:"volume_ids"`
}
