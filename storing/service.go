package storing

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/volume"
)

//go:generate mockgen -destination=../mock/storing.go -mock_names=Service=Storing -package=mock github.com/xescugc/rebost/storing Service

// Service is the interface of used to for the storing,
// it's the one that will be used when defining a client
// to consume the API
type Service interface {
	volume.Volume
	replica.Node

	// Config returns the current Service configuration
	Config(context.Context) (*config.Config, error)
}

type service struct {
	members Membership
	cfg     *config.Config

	replicasPendentCha chan replica.Pendent

	replicasPendentMapLock sync.RWMutex
	replicasPendentMap     map[string]struct{}

	ctx    context.Context
	cancel context.CancelFunc
}

// New returns an implementation of the Node with
// the given parameters
func New(cfg *config.Config, m Membership) Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &service{
		members: m,
		cfg:     cfg,

		replicasPendentCha: make(chan replica.Pendent, cfg.MaxReplicaPendent),
		replicasPendentMap: make(map[string]struct{}),

		ctx:    ctx,
		cancel: cancel,
	}
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
	return v.DeleteFile(ctx, k)
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

func (s *service) CreateReplicaPendent(ctx context.Context, rp replica.Pendent) error {
	select {
	case s.replicasPendentCha <- rp:
		s.replicasPendentMapLock.Lock()
		s.replicasPendentMap[rp.ID] = struct{}{}
		s.replicasPendentMapLock.Unlock()
		return nil
	default:
		return errors.New("too busy to replicate")
	}
}

func (s *service) HasReplicaPendent(ctx context.Context, ID string) (bool, error) {
	s.replicasPendentMapLock.RLock()
	_, ok := s.replicasPendentMap[ID]
	s.replicasPendentMapLock.RUnlock()
	return ok, nil
}

func (s *service) getLocalVolume(ctx context.Context, k string) volume.Volume {
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

	return nil, nil
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
