package replica

import "context"

//go:generate mockgen -destination=../mock/pendent_repository.go -mock_names=PendentRepository=ReplicaPendentRepository -package=mock github.com/xescugc/rebost/replica PendentRepository

// PendentRepository is the interface that defines which actions
// can be done to the Pendent struct
type PendentRepository interface {
	Create(ctx context.Context, p *Pendent) error
}
