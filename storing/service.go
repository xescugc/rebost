package storing

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"sync"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/volume"
)

const (
	noReplica = 1
)

//go:generate mockgen -destination=../mock/storing.go -mock_names=Service=Storing -package=mock github.com/xescugc/rebost/storing Service

// Service is the interface of used to for the storing,
// it's the one that will be used when defining a client
// to consume the API
type Service interface {
	volume.Volume

	// Config returns the current Service configuration
	Config(context.Context) (*config.Config, error)

	// CreateReplica creates a new File replica
	CreateReplica(ctx context.Context, key string, reader io.ReadCloser) (vID string, err error)
}

type service struct {
	members Membership
	cfg     *config.Config

	ctx    context.Context
	cancel context.CancelFunc

	logger kitlog.Logger
}

// New returns an implementation of the Node with
// the given parameters
func New(cfg *config.Config, m Membership, logger kitlog.Logger) Service {
	ctx, cancel := context.WithCancel(context.Background())
	s := &service{
		members: m,
		cfg:     cfg,

		ctx:    ctx,
		cancel: cancel,

		logger: logger,
	}

	if s.cfg.Replica != -1 {
		go s.loopVolumesReplicas()
		go s.loopRemovedVolumeDIs()
	}

	return s
}

func (s *service) Config(_ context.Context) (*config.Config, error) {
	return s.cfg, nil
}

func (s *service) CreateFile(ctx context.Context, k string, r io.ReadCloser, rep int) error {
	if rep == 0 {
		rep = s.cfg.Replica
	}
	err := s.getLocalVolume(ctx, k).CreateFile(ctx, k, r, rep)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) GetFile(ctx context.Context, k string) (io.ReadCloser, error) {
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
	err = v.DeleteFile(ctx, k)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) HasFile(ctx context.Context, k string) (bool, error) {
	v, err := s.findVolume(ctx, localVolumesToVolumes(s.members.LocalVolumes()), k)
	if err != nil && err.Error() != "not found" {
		return false, err
	}

	if v != nil {
		return true, nil
	}

	return false, nil
}

func (s *service) CreateReplica(ctx context.Context, key string, reader io.ReadCloser) (string, error) {
	if s.cfg.Replica == -1 {
		return "", errors.New("can not store replicas")
	}
	v := s.getLocalVolume(ctx, key)
	err := v.CreateFile(ctx, key, reader, noReplica)
	if err != nil {
		return "", err
	}

	return v.ID(), nil
}

func (s *service) UpdateFileReplica(ctx context.Context, key string, volumeIDs []string, replica int) error {
	if s.cfg.Replica == -1 {
		return errors.New("can not store replicas")
	}

	v, err := s.findVolume(ctx, localVolumesToVolumes(s.members.LocalVolumes()), key)
	if err != nil {
		return err
	}

	err = v.UpdateFileReplica(ctx, key, volumeIDs, replica)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) getLocalVolume(ctx context.Context, k string) volume.Local {
	vls := s.members.LocalVolumes()

	rand.Seed(time.Now().UTC().UnixNano())
	return vls[rand.Intn(len(vls))]
}

// getVolume returns a volume that may have k in his index. It tries first with
// the LocalVolumes and then with the Nodes
func (s *service) getVolume(ctx context.Context, k string) (volume.Volume, error) {
	v, err := s.findVolume(ctx, localVolumesToVolumes(s.members.LocalVolumes()), k)
	if err != nil && err.Error() != "not found" {
		return nil, err
	}

	if v != nil {
		return v, nil
	}

	v, err = s.findVolume(ctx, servicesToVolumes(s.members.Nodes()), k)
	if err != nil && err.Error() != "not found" {
		return nil, err
	}

	if v != nil {
		return v, nil
	}

	return nil, errors.New("not found")
}

// findVolume finds the volume that has the key k within the volumes vls in parallel
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

// localVolumesToVolumes convert []volume.Local to []volume.Volume
func localVolumesToVolumes(lvs []volume.Local) []volume.Volume {
	rvs := make([]volume.Volume, 0, len(lvs))
	for _, v := range lvs {
		rvs = append(rvs, v)
	}
	return rvs
}

// servicesToVolumes convert []Service to []volume.Volume
func servicesToVolumes(ns []Service) []volume.Volume {
	rvs := make([]volume.Volume, 0, len(ns))
	for _, v := range ns {
		rvs = append(rvs, v)
	}
	return rvs
}
