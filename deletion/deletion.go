package deletion

// Deletion holds a pending remote-replica deletion job.
// When a file's last key is removed from a local volume, a Deletion is created
// so the background loop can propagate the delete to the remote volumes listed
// in VolumeIDs.
type Deletion struct {
	// VolumeDeletionID is the unique BoltDB key assigned on creation.
	VolumeDeletionID []byte

	// Key is the file key to delete from the remote volumes.
	Key string

	// VolumeIDs is the list of remote volume IDs that still hold a copy.
	VolumeIDs []string
}
