package scrub

// Scrub holds a pending checksum-repair job.
// When a volume detects that a local file's SHA1 no longer matches its stored
// signature, a Scrub is created so the background loop can fetch a good copy
// from one of the listed remote volumes.
type Scrub struct {
	// ScrubID is the unique BoltDB key assigned on creation.
	ScrubID []byte

	// Key is the file key to repair on this volume.
	Key string

	// Signature is the expected SHA1 of the file content.
	Signature string

	// VolumeIDs is the list of remote volume IDs that hold a known-good copy.
	// The local volume ID has already been filtered out.
	VolumeIDs []string
}
