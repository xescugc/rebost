package storage

import (
	"io"
	"math/rand"
	"time"

	"github.com/xescugc/rebost/config"
)

type Storage struct {
	localVolumes  []Volume
	remoteVolumes []Volume
	term          uint32
	status        string
}

func New(c *config.Config) *Storage {
	s := &Storage{}

	s.localVolumes = make([]Volume, 0, 1)

	if len(c.Volumes) == 0 {
		c.Volumes = []string{"./data"}
	}

	for _, v := range c.Volumes {
		s.localVolumes = append(s.localVolumes, NewVolume(v))
	}

	return s
}

// ExistsFile validates if the key exits on any volume known to this one, remote or local
func (s *Storage) ExistsFile(key string) (bool, error) {
	v := s.getVolumeWithKey(key, true)
	return v != nil, nil
}

// AddFile adds a file to the storage with the provided key and reader
func (s *Storage) AddFile(key string, reader io.Reader) (*File, error) {
	return s.getLocalVolume().AddFile(key, reader)
}

// GetFile returns the resquested File if found
func (s *Storage) GetFile(key string) (*File, error) {
	return s.getVolumeWithKey(key, false).GetFile(key)
}

// HasFile validates if in the localVolumes exists the provided key
func (s *Storage) HasFile(key string) (bool, error) {
	v := s.getVolumeWithKey(key, false)
	return v != nil, nil
}

// DeleteFiles deletes the files with the provided key if found
func (s *Storage) DeleteFile(key string) error {
	return s.getVolumeWithKey(key, false).DeleteFile(key)
}

// Clean cleans all the localVolumes
func (s *Storage) Clean() {
	for _, v := range s.localVolumes {
		v.Clean()
	}
}

// getLocalVolume gets one of the local volumes to perform some operations
func (s *Storage) getLocalVolume() Volume {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return s.localVolumes[r.Intn(len(s.localVolumes))]
}

// getVolume gets one of the volume that knows about the key
func (s *Storage) getVolumeWithKey(key string, propagate bool) Volume {
	volumes := s.localVolumes
	if propagate {
		volumes = append(volumes, s.remoteVolumes...)
	}

	var found Volume
	for _, v := range volumes {
		if ok, _ := v.HasFile(key); ok {
			found = v
		}
	}

	return found
}
