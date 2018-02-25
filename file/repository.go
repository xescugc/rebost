package file

type Repository interface {
	CreateOrReplace(file *File) error
	FindBySignature(sig string) (*File, error)
	DeleteBySignature(sig string) error
}
