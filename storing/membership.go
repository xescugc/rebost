package storing

import "github.com/xescugc/rebost/volume"

//go:generate mockgen -destination=../mock/membership.go -mock_names=Membership=Membership -package=mock github.com/xescugc/rebost/storing Membership

// Membership is the interface that hides the logic behind the
// cluseter members. In this "domain" (rebost), the members
// are considered volume.Volume.
type Membership interface {
	// Nodes return all the Nodes of the cluster
	Nodes() []Service

	// LocalVolumes returns only the local volumes
	LocalVolumes() []volume.Local

	// GetNodeWithVolumeByID returns the Node that has the
	// vid in his volumes
	GetNodeWithVolumeByID(vid string) (Service, error)

	// RemovedVolumeIDs returns a list of are the volumeIDs
	// that left the cluster
	RemovedVolumeIDs() []string

	// Leave makes it leave the cluster
	Leave()
}
