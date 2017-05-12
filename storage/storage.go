package storage

import (
	"io"

	"github.com/xescugc/rebost/config"
)

type Storage struct {
	volumes []*volume
}

func New(c *config.Config) *Storage {
	s := &Storage{}

	s.volumes = make([]*volume, 1)

	if len(c.Volumes) == 0 {
		c.Volumes = []string{"./data"}
	}

	for _, v := range c.Volumes {
		s.volumes = append(s.volumes, NewVolume(v))
	}

	return s
}

func (s *Storage) Add(key string, reader io.Reader) (*File, error) {
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
