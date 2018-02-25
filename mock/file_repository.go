package mock

import "github.com/xescugc/rebost/file"

type FileRepository struct {
	CreateOrReplaceFn      func(f *file.File) error
	CreateOrReplaceInvoked bool
	CreateOrReplaceTimes   int

	FindBySignatureFn      func(sig string) (*file.File, error)
	FindBySignatureInvoked bool
	FindBySignatureTimes   int

	DeleteBySignatureFn      func(sig string) error
	DeleteBySignatureInvoked bool
	DeleteBySignatureTimes   int
}

func (r *FileRepository) CreateOrReplace(f *file.File) error {
	r.CreateOrReplaceInvoked = true
	r.CreateOrReplaceTimes += 1
	return r.CreateOrReplaceFn(f)
}

func (r *FileRepository) FindBySignature(sig string) (*file.File, error) {
	r.FindBySignatureInvoked = true
	r.FindBySignatureTimes += 1
	return r.FindBySignatureFn(sig)
}

func (r *FileRepository) DeleteBySignature(sig string) error {
	r.DeleteBySignatureInvoked = true
	r.DeleteBySignatureTimes += 1
	return r.DeleteBySignatureFn(sig)
}
