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

	// Leave makes it leave the cluster
	Leave()
}
