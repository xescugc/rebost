package storing

import (
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/membership"
	"github.com/xescugc/rebost/volume"
)

//go:generate mockgen -destination=../mock/membership.go -mock_names=Membership=Membership -package=mock github.com/xescugc/rebost/storing Membership

// Membership is the interface that hides the logic behind the
// cluseter members. In this "domain" (rebost), the members
// are considered volume.Volume.
type Membership interface {
	// Nodes return all the Nodes of the cluster except the current one
	Nodes() []*client.Client

	// NodesWithoutVolumeIDs return all the Nodes of the cluster except the current one that
	// do not have any of the provided vids
	NodesWithoutVolumeIDs(vids []string) []*client.Client

	// LocalVolumes returns only the local volumes
	LocalVolumes() []volume.Local

	// GetNodeWithVolumeByID returns the Node that has the
	// vid in his volumes
	GetNodeWithVolumeByID(vid string) (*client.Client, error)

	// GetNodeState returns the Staet of the Node
	GetNodeState(nn string) (*membership.State, error)

	// RemovedVolumeIDs returns a list of are the volumeIDs
	// that left the cluster
	RemovedVolumeIDs() []string

	// Leave makes it leave the cluster
	Leave()
}
