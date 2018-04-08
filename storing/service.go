package storing

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/xescugc/rebost/membership"
	"github.com/xescugc/rebost/volume"
)

//go:generate mockgen -destination=../mock/storing.go -mock_names=Service=Storing -package=mock github.com/xescugc/rebost/storing Service

// Service is the interface of used to for the storing,
// it's the one that will be used when defining a client
// to consume the API
type Service interface {
	volume.Volume
}

type service struct {
	members membership.Membership
}

// New returns an implementation of the Service with
// the given parameters
func New(m membership.Membership) Service {
	return &service{
		members: m,
	}
}

func (s *service) CreateFile(ctx context.Context, k string, r io.Reader) error {
	err := s.getLocalVolume(ctx, k).CreateFile(ctx, k, r)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) GetFile(ctx context.Context, k string) (io.Reader, error) {
	v, err := s.getVolume(ctx, k)
	if err != nil {
		return nil, err
	}
	r, err := v.GetFile(ctx, k)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s *service) DeleteFile(ctx context.Context, k string) error {
	v, err := s.getVolume(ctx, k)
	if err != nil {
		return err
	}
	return v.DeleteFile(ctx, k)
}

func (s *service) HasFile(ctx context.Context, k string) (bool, error) {
	v, err := s.findVolume(ctx, s.members.LocalVolumes(), k)
	if err != nil && err.Error() != "not found" {
		return false, err
	}

	if v != nil {
		return true, nil
	}

	return false, nil
}

func (s *service) getLocalVolume(ctx context.Context, k string) volume.Volume {
	vls := s.members.LocalVolumes()

	rand.Seed(time.Now().UTC().UnixNano())
	return vls[rand.Intn(len(vls))]
}

func (s *service) getVolume(ctx context.Context, k string) (volume.Volume, error) {
	v, err := s.findVolume(ctx, s.members.LocalVolumes(), k)
	if err != nil && err.Error() != "not found" {
		return nil, err
	}

	if v != nil {
		return v, nil
	}

	v, err = s.findVolume(ctx, s.members.RemoteVolumes(), k)
	if err != nil && err.Error() != "not found" {
		return nil, err
	}

	if v != nil {
		return v, nil
	}

	return nil, nil
}

// findVolume finds the volume that has the key (k) within the volumes (vls) in parallel
func (s *service) findVolume(ctx context.Context, vls []volume.Volume, k string) (volume.Volume, error) {
	var wg sync.WaitGroup
	cctx, cfn := context.WithCancel(ctx)

	// doneC is used to notify which is the volume found
	doneC := make(chan volume.Volume)

	wg.Add(len(vls))

	for _, v := range vls {
		go func(v volume.Volume) {
			defer wg.Done()
			ok, err := v.HasFile(cctx, k)
			if err != nil {
				// TODO: Log the error?
				// remember that when the ctx is canceled, it
				// makes the volume.Volume return "context canceled"
				// if it's a node
				return
			}
			if ok {
				select {
				case <-cctx.Done():
				case doneC <- v:
				}
				return
			}
		}(v)
	}

	go func() {
		wg.Wait()
		close(doneC)
	}()

	var (
		v   volume.Volume
		err error
	)
	select {
	case v = <-doneC:
		if v == nil {
			// If it's done without a value, means
			// that the doneC has ben closed and that
			// no volume was found
			err = errors.New("not found")
		}
		// Cancel all the possible still running
		// request to the volumes
		cfn()
	}

	return v, err
}
