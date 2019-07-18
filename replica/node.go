package replica

import (
	"context"
)

//go:generate mockgen -destination=../mock/replica_node.go -mock_names=Node=ReplicaNode -package=mock github.com/xescugc/rebost/replica Node

// Node represents the functions that a Node have
// to has to implement replication
type Node interface {
	// CreateReplicaPendent tires to create a replica.Pendent
	// if it successes at some point the Node will get File
	// to replicate
	CreateReplicaPendent(ctx context.Context, rp Pendent) error

	// HasReplicaPendent asks if the ID is in the list of
	// replicas todo
	HasReplicaPendent(ctx context.Context, ID string) (bool, error)
}
