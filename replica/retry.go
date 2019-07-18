package replica

// Retry is the struct that holds the information of the
// accepted replica that did not happen yet.
type Retry struct {
	// ID is the identifier of the replica
	ID string

	// Key is the key to replicate
	Key string

	// Signature is the signature of the file
	Signature string

	// VolumeID that stored the original replica
	VolumeID string

	// NodeName has the name of the
	// Node that has this replica
	NodeName string

	// VolumeReplicaID represents the unique ID of the replica
	// inside the Volume. It's used to index  in a
	// uinique incress order on the DB
	VolumeReplicaID []byte
}

// NewRetryFromPendent returns the initialization of a Retry
// from the Pendent rp and with the NodeName nodeName
func NewRetryFromPendent(rp *Pendent, nodeName string) *Retry {
	return &Retry{
		ID:        rp.ID,
		Key:       rp.Key,
		Signature: rp.Signature,
		VolumeID:  rp.VolumeID,
		NodeName:  nodeName,
	}
}
