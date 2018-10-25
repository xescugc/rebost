package replica

// Pendent is the struct holding a pending file to replicate
type Pendent struct {
	// ID is the identifier of the replica
	ID string

	// Replica is the number of replicas that it needs
	Replica int

	// Key is the key to replicate
	Key string

	// Signature is the signature of the file
	Signature string

	// VolumeID that stores the original replica
	VolumeID string
}
