package idxkey

import "context"

//go:generate mockgen -destination=../mock/idxkey_repository.go -mock_names=Repository=IDXKeyRepository -package=mock github.com/xescugc/rebost/idxkey Repository

// Repository is the interface that has to be fulfiled to interact with IDXKeys
type Repository interface {
	CreateOrReplace(ctx context.Context, ik *IDXKey) error
	FindByKey(ctx context.Context, key string) (*IDXKey, error)
	DeleteByKey(ctx context.Context, key string) error
}
