package storage

import (
	"io"

	"github.com/xescugc/rebost/config"
)

type Storage struct {
	volumes []*volume
}

func New(c *config.Config) *Storage {
}

func (s *Storage) Add(key string, reader io.Reader) *File {
	return s.getVolume().Add(key, reader)
}

func (s *Storage) Get() {
}

func (s *Storage) Exists() {
}

func (s *Storage) Remove() {
}

func (s *Storage) getVolume() *volume {
	return s.volumes[0]
}
