package storing

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"sync"
	"time"

	kitlog "github.com/go-kit/kit/log"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/xescugc/rebost/client"
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

	cache *lru.ARCCache[string, string]

	ctx    context.Context
	cancel context.CancelFunc

	logger kitlog.Logger
}

// New returns an implementation of the Node with
// the given parameters
func New(cfg *config.Config, m Membership, logger kitlog.Logger) (Service, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cache, err := lru.NewARC[string, string](cfg.Cache.Size)
	if err != nil {
		cancel()
		return nil, err
	}
	s := &service{
		members: m,
		cfg:     cfg,

		cache: cache,

		ctx:    ctx,
		cancel: cancel,

		logger: kitlog.With(logger, "src", "storing", "name", cfg.Name),
	}

	if s.cfg.Replica != -1 {
		go s.loopVolumesReplicas()
		go s.loopRemovedVolumeDIs()
	}

	return s, nil
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
	_, v, err := s.getVolume(ctx, k)
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
	_, v, err := s.getVolume(ctx, k)
	if err != nil {
		return err
	}
	err = v.DeleteFile(ctx, k)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) HasFile(ctx context.Context, k string) (string, bool, error) {
	vid, v, err := s.findVolume(ctx, localVolumesToVolumes(s.members.LocalVolumes()), k)
	if err != nil && err.Error() != "not found" {
		return "", false, err
	}

	if v != nil {
		return vid, true, nil
	}

	// If we do not have the file we try to find if someone we know has it
	// so we use the cache to see if we know who has it.
	// This will return the vid but false as we do not have it
	if vid, ok := s.cache.Get(k); ok {
		return vid, false, nil
	}

	return "", false, nil
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

	_, v, err := s.findVolume(ctx, localVolumesToVolumes(s.members.LocalVolumes()), key)
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

// getVolume returns a volume and the volumeID that may have k in his index. It tries first with
// the LocalVolumes and then with the Nodes
func (s *service) getVolume(ctx context.Context, k string) (string, volume.Volume, error) {
	if vid, ok := s.cache.Get(k); ok {
		n, err := s.members.GetNodeWithVolumeByID(vid)
		if err != nil {
			return "", nil, err
		}
		return vid, n, nil
	}

	vid, v, err := s.findVolume(ctx, localVolumesToVolumes(s.members.LocalVolumes()), k)
	if err != nil && err.Error() != "not found" {
		return "", nil, err
	}

	if v != nil {
		return vid, v, nil
	}

	vid, v, err = s.findVolume(ctx, clientsToVolumes(s.members.Nodes()), k)
	if err != nil && err.Error() != "not found" {
		return "", nil, err
	}

	if v != nil {
		// We only cache the remove ones because the local ones are faster and easier to access
		// but also because the list of nodes does not include the current node so if where
		// to cache the local volumes when we do the GetNodeWithVolumeByID would return a
		// not found.
		// Also the intention of the cache is to avoid to query HasFile to the other nodes
		s.cache.Add(k, vid)
		return vid, v, nil
	}

	return "", nil, errors.New("not found")
}

type msg struct {
	v   volume.Volume
	vid string
}

// findVolume finds the volume and the ID that has the key k within the volumes vls in parallel
func (s *service) findVolume(ctx context.Context, vls []volume.Volume, k string) (string, volume.Volume, error) {
	var wg sync.WaitGroup
	cctx, cfn := context.WithCancel(ctx)

	// doneC is used to notify which is the volume found
	doneC := make(chan msg)

	wg.Add(len(vls))

	for _, v := range vls {
		go func(v volume.Volume) {
			defer wg.Done()
			vid, ok, err := v.HasFile(cctx, k)
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
				case doneC <- msg{v, vid}:
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
		m   msg
		err error
	)
	select {
	case m = <-doneC:
		if m.v == nil {
			// If it's done without a value, means
			// that the doneC has ben closed and that
			// no volume was found
			err = errors.New("not found")
		}
		// Cancel all the possible still running
		// request to the volumes
		cfn()
	}

	return m.vid, m.v, err
}

// localVolumesToVolumes convert []volume.Local to []volume.Volume
func localVolumesToVolumes(lvs []volume.Local) []volume.Volume {
	rvs := make([]volume.Volume, 0, len(lvs))
	for _, v := range lvs {
		rvs = append(rvs, v)
	}
	return rvs
}

// clientsToVolumes convert []*client.Client to []volume.Volume
func clientsToVolumes(cs []*client.Client) []volume.Volume {
	rvs := make([]volume.Volume, 0, len(cs))
	for _, c := range cs {
		rvs = append(rvs, c)
	}
	return rvs
}
