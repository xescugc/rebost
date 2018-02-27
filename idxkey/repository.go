package idxkey

//go:generate mockgen -destination=../mock/idxkey_repository.go -mock_names=Repository=IDXKeyRepository -package=mock github.com/xescugc/rebost/idxkey Repository

// Repository is the interface that has to be fulfiled to interact with IDXKeys
type Repository interface {
	CreateOrReplace(ik *IDXKey) error
	FindByKey(key string) (*IDXKey, error)
	DeleteByKey(key string) error
}
