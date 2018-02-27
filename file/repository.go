package file

//go:generate mockgen -destination=../mock/file_repository.go -mock_names=Repository=FileRepository -package=mock github.com/xescugc/rebost/file Repository

type Repository interface {
	CreateOrReplace(f *File) error
	FindBySignature(sig string) (*File, error)
	DeleteBySignature(sig string) error
}
