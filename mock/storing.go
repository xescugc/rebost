package mock

import "io"

type Storing struct {
	CreateFileFn      func(key string, reader io.Reader) error
	CreateFileInvoked bool

	GetFileFn      func(key string) (io.Reader, error)
	GetFileInvoked bool

	DeleteFileFn      func(key string) error
	DeleteFileInvoked bool
}

func (s *Storing) CreateFile(k string, r io.Reader) error {
	s.CreateFileInvoked = true
	return s.CreateFileFn(k, r)
}

func (s *Storing) GetFile(k string) (io.Reader, error) {
	s.GetFileInvoked = true
	return s.GetFileFn(k)
}

func (s *Storing) DeleteFile(k string) error {
	s.DeleteFileInvoked = true
	return s.DeleteFileFn(k)
}
