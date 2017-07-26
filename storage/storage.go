package storage

import (
	"io"

	"github.com/xescugc/rebost/config"
)

type Storage struct {
	volumes []Volume
}

func New(c *config.Config) *Storage {
	s := &Storage{}

	s.volumes = make([]Volume, 0, 1)

	if len(c.Volumes) == 0 {
		c.Volumes = []string{"./data"}
	}

	for _, v := range c.Volumes {
		s.volumes = append(s.volumes, NewVolume(v))
	}

	return s
}

func (s *Storage) AddFile(key string, reader io.Reader) (*File, error) {
	return s.getVolume().AddFile(key, reader)
}

func (s *Storage) GetFile(key string) (*File, error) {
	return s.getVolume().GetFile(key)
}

func (s *Storage) Exists(key string) (bool, error) {
	return s.getVolume().ExistsFile(key)
}

func (s *Storage) DeleteFile(key string) error {
	return s.getVolume().DeleteFile(key)
}

func (s *Storage) getVolume() Volume {
	return s.volumes[0]
}
