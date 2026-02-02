package replica

import "time"

// Replica is the struct holding a pending file to replicate
type Replica struct {
	// ID is the identifier of the replica
	ID string

	// OriginalCount it's the original number of replicas
	OriginalCount int

	// Count it's the number of missing replicas
	Count int

	// Key is the key to replicate
	Key string

	// Signature is the signature of the file
	Signature string

	// VolumeID that stored the original replica
	VolumeID string

	// VolumeIDs list of all the volumes that have the replica
	// excluding the original one
	VolumeIDs []string

	// VolumeReplicaID represents the unique ID of the replica
	// inside the Volume. It's used to index in a
	// unique increase order on the DB
	VolumeReplicaID []byte

	// TTL is the duration the original file has
	TTL time.Duration

	// CreatedAt is the time of creation of the original file
	CreatedAt time.Time
}
