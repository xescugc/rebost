package idxkey

//go:generate mockgen -destination=../mock/idxkey_repository.go -mock_names=Repository=IDXKeyRepository -package=mock github.com/xescugc/rebost/idxkey Repository

type Repository interface {
	CreateOrReplace(ik *IDXKey) error
	FindByKey(key string) (*IDXKey, error)
	DeleteByKey(key string) error
}
