package mock

import "github.com/xescugc/rebost/idxkey"

type IDXKeyRepository struct {
	CreateOrReplaceFn      func(ik *idxkey.IDXKey) error
	CreateOrReplaceInvoked bool

	FindByKeyFn      func(key string) (*idxkey.IDXKey, error)
	FindByKeyInvoked bool

	DeleteByKeyFn      func(key string) error
	DeleteByKeyInvoked bool
}

func (r *IDXKeyRepository) CreateOrReplace(ik *idxkey.IDXKey) error {
	r.CreateOrReplaceInvoked = true
	return r.CreateOrReplaceFn(ik)
}

func (r *IDXKeyRepository) FindByKey(key string) (*idxkey.IDXKey, error) {
	r.FindByKeyInvoked = true
	return r.FindByKeyFn(key)
}

func (r *IDXKeyRepository) DeleteByKey(key string) error {
	r.DeleteByKeyInvoked = true
	return r.DeleteByKeyFn(key)
}
