package deletion

import "context"

//go:generate mockgen -destination=../mock/deletion_repository.go -mock_names=Repository=DeletionRepository -package=mock github.com/xescugc/rebost/deletion Repository

// Repository defines the operations for the pending deletion queue.
type Repository interface {
	// Create stores a new pending deletion job.
	Create(ctx context.Context, d *Deletion) error

	// First returns the next pending deletion job.
	First(ctx context.Context) (*Deletion, error)

	// Delete removes a processed deletion job.
	Delete(ctx context.Context, d *Deletion) error
}
