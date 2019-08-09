package replica

// Replica is the struct holding a pending file to replicate
type Replica struct {
	// ID is the identifier of the replica
	ID string

	// OriginalCount it's the original numer of replicas
	OriginalCount int

	// Count it's the number of missing replicas
	Count int

	// Key is the key to replicate
	Key string

	// Signature is the signature of the file
	Signature string

	// VolumeID that stored the original replica
	VolumeID string

	// VolumeReplicaID represents the unique ID of the replica
	// inside the Volume. It's used to index in a
	// uinique incress order on the DB
	VolumeReplicaID []byte
}
