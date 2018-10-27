package replica

import "context"

//go:generate mockgen -destination=../mock/pendent_repository.go -mock_names=PendentRepository=ReplicaPendentRepository -package=mock github.com/xescugc/rebost/replica PendentRepository

// PendentRepository is the interface that defines which actions
// can be done to the Pendent struct
type PendentRepository interface {
	// Create stores the Pendent
	Create(ctx context.Context, p *Pendent) error

	// First gets the first element
	First(ctx context.Context) (*Pendent, error)

	// Delete removes the Pendent
	Delete(ctx context.Context, p *Pendent) error
}
