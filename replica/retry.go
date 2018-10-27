package replica

// Retry is the struct that holds the information of the
// accepted replica that did not happen yet.
type Retry struct {
	Pendent

	// ToVolumeID stores the ID of the Volume
	// that accepted the replica
	ToVolumeID string
}
