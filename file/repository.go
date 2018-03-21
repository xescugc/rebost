package file

import "context"

//go:generate mockgen -destination=../mock/file_repository.go -mock_names=Repository=FileRepository -package=mock github.com/xescugc/rebost/file Repository

// Repository is the interface that has to be fulfiled to interact with Files
type Repository interface {
	CreateOrReplace(ctx context.Context, f *File) error
	FindBySignature(ctx context.Context, sig string) (*File, error)
	DeleteBySignature(ctx context.Context, sig string) error
}
