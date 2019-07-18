package replica

import "context"

//go:generate mockgen -destination=../mock/retry_repository.go -mock_names=RetryRepository=ReplicaRetryRepository -package=mock github.com/xescugc/rebost/replica RetryRepository

// RetryRepository is the interface that defines which actions
// can be done to the Pendent struct
type RetryRepository interface {
	// Create stores the Retry
	Create(ctx context.Context, r *Retry) error

	// First gets the first element
	First(ctx context.Context) (*Retry, error)

	// Delete removes the Retry
	Delete(ctx context.Context, r *Retry) error
}
