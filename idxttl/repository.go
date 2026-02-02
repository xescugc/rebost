package idxttl

import (
	"context"
	"time"
)

//go:generate mockgen -destination=../mock/idxttl_repository.go -mock_names=Repository=IDXTTLRepository -package=mock github.com/xescugc/rebost/idxttl Repository

// Repository is the interface that has to be fulfilled to interact with IDXTTL.
// All the 'ea' used as keys will be  converted to RFC3339
type Repository interface {
	CreateOrReplace(ctx context.Context, ik *IDXTTL) error
	// Filter will return all the IDXTTL older than ea
	Filter(ctx context.Context, ea time.Time) ([]*IDXTTL, error)
	Find(ctx context.Context, ea time.Time) (*IDXTTL, error)
	Delete(ctx context.Context, ea time.Time) error
}
