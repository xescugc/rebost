package scrub

import "context"

//go:generate go tool mockgen -destination=../mock/scrub_repository.go -mock_names=Repository=ScrubRepository -package=mock github.com/xescugc/rebost/scrub Repository

// Repository defines the operations for the pending scrub-repair queue.
type Repository interface {
	// Create stores a new pending scrub-repair job.
	Create(ctx context.Context, s Scrub) error

	// First returns the next pending scrub-repair job.
	First(ctx context.Context) (*Scrub, error)

	// Delete removes a processed scrub-repair job.
	Delete(ctx context.Context, s *Scrub) error
}
