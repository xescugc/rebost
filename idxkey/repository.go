package idxkey

type Repository interface {
	CreateOrReplace(ik *IDXKey) error
	FindByKey(key string) (*IDXKey, error)
	DeleteByKey(key string) error
}
