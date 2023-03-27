package replica

import "context"

//go:generate mockgen -destination=../mock/replica_repository.go -mock_names=Repository=ReplicaRepository -package=mock github.com/xescugc/rebost/replica Repository

// Repository is the interface that defines which actions
// can be done to the Pendent struct
type Repository interface {
	// Create stores the Pendent
	Create(ctx context.Context, r *Replica) error

	// First gets the first element
	First(ctx context.Context) (*Replica, error)

	// Delete removes the Pendent
	Delete(ctx context.Context, r *Replica) error

	DeleteAll(ctx context.Context) error
}
