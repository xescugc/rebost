package storing

import (
	"io"
	"math/rand"
	"time"

	"github.com/xescugc/rebost/volume"
)

//go:generate mockgen -destination=../mock/storing.go -mock_names=Service=Storing -package=mock github.com/xescugc/rebost/storing Service

// Service is the interface of used to for the storing,
// it's the one that will be used when defining a client
// to consume the API
type Service interface {
	CreateFile(key string, reader io.Reader) error

	GetFile(key string) (io.Reader, error)

	HasFile(key string) (bool, error)

	DeleteFile(key string) error
}

type service struct {
	localVolumes []volume.Volume
}

// New returns an implementation of the Service with
// the given parameters
func New(lv []volume.Volume) Service {
	return &service{
		localVolumes: lv,
	}
}

func (s *service) CreateFile(k string, r io.Reader) error {
	_, err := s.localVolumes[0].CreateFile(k, r)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) GetFile(k string) (io.Reader, error) {
	v, err := s.getVolume(k)
	if err != nil {
		return nil, err
	}
	r, err := v.GetFile(k)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s *service) DeleteFile(k string) error {
	v, err := s.getVolume(k)
	if err != nil {
		return err
	}
	return v.DeleteFile(k)
}

func (s *service) HasFile(k string) (bool, error) {
	for _, v := range s.localVolumes {
		ok, err := v.HasFile(k)
		if err != nil {
			return false, err
		}
		if ok {
			return ok, nil
		}
	}
	return false, nil
}

func (s *service) getVolume(k string) (volume.Volume, error) {
	for _, v := range s.localVolumes {
		ok, err := v.HasFile(k)
		if err != nil {
			return nil, err
		}
		if ok {
			return v, nil
		}
	}
	rand.Seed(time.Now().UTC().UnixNano())
	return s.localVolumes[rand.Intn(len(s.localVolumes))], nil
}
