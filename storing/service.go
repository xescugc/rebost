package storing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"log/slog"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/volume"
)

const (
	noReplica = 1
)

//go:generate go tool mockgen -destination=../mock/storing.go -mock_names=Service=Storing -package=mock github.com/xescugc/rebost/storing Service

// Service is the interface of used to for the storing,
// it's the one that will be used when defining a client
// to consume the API
type Service interface {
	volume.Volume

	// Config returns the current Service configuration
	Config(context.Context) (*config.Config, error)

	// CreateReplica creates a new File replica
	CreateReplica(ctx context.Context, key string, reader io.ReadCloser, ttl time.Duration, ca time.Time) (vID string, err error)

	// Drain gracefully decommissions this node by ensuring all locally-held files
	// have enough replicas on other nodes, then purging local copies and leaving
	// the cluster.
	Drain(ctx context.Context) error

	// GetReplicaInfo returns the VolumeIDs and replica count for a file.
	// It searches local volumes and returns the first match.
	GetReplicaInfo(ctx context.Context, key string) ([]string, int, error)

	// Ready checks whether all local volumes are accessible.
	// Returns nil if all volumes respond to GetState without error.
	Ready(ctx context.Context) error
}

type service struct {
	members  Membership
	cfg      *config.Config
	draining atomic.Bool

	cache *lru.ARCCache[string, string]

	ctx    context.Context
	cancel context.CancelFunc

	logger *slog.Logger
}

// New returns an implementation of the Node with
// the given parameters
func New(cfg *config.Config, m Membership, logger *slog.Logger) (Service, error) {
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

		logger: logger.With("src", "storing", "name", cfg.Name),
	}

	if s.cfg.Replica != -1 {
		go s.loopVolumes()
		go s.loopRemovedVolumeDIs()
		go s.loopConsistencyCheck()
	}

	return s, nil
}

func (s *service) Config(_ context.Context) (*config.Config, error) {
	return s.cfg, nil
}

func (s *service) CreateFile(ctx context.Context, k string, r io.ReadCloser, rep int, ttl time.Duration, ca time.Time) error {
	if s.draining.Load() {
		return errors.New("node is draining")
	}

	if rep == 0 {
		rep = s.cfg.Replica
	}

	// Buffer the incoming stream so content can be replayed on fallback.
	tmpFile, err := os.CreateTemp("", "rebost-buf-*")
	if err != nil {
		return fmt.Errorf("creating buffer file: %w", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	tee := io.NopCloser(io.TeeReader(r, tmpFile))

	// Try local volumes in preferred order. The first gets the tee (streams
	// directly); subsequent ones seek from the tmp buffer.
	for i, v := range s.getLocalVolumes(ctx, k) {
		var reader io.ReadCloser
		if i == 0 {
			reader = tee
		} else {
			if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
				return err
			}
			reader = io.NopCloser(tmpFile)
		}
		if err := v.CreateFile(ctx, k, reader, rep, ttl, ca); err == nil {
			return nil
		} else if !errors.Is(err, volume.ErrNoSpace) {
			return err
		}
	}

	// Drain tee into tmpFile in case no local volumes were tried
	// (len==0) or the tee was not fully consumed by the mock/volume.
	if _, err := io.Copy(io.Discard, tee); err != nil {
		return err
	}

	fi, err := tmpFile.Stat()
	if err != nil {
		return err
	}
	size := fi.Size()

	nodes := s.members.NodesWithCapacity(size)
	for _, node := range nodes {
		if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
			return err
		}
		err := node.CreateFile(ctx, k, io.NopCloser(tmpFile), rep, ttl, ca)
		if err == nil {
			return nil
		}
		if !errors.Is(err, volume.ErrNoSpace) {
			return err
		}
	}

	return volume.ErrClusterFull
}

func (s *service) StatFile(ctx context.Context, k string) (*volume.FileStat, error) {
	_, v, err := s.getVolume(ctx, k)
	if err != nil {
		return nil, err
	}
	return v.StatFile(ctx, k)
}

func (s *service) GetFile(ctx context.Context, k string) (io.ReadCloser, int64, error) {
	_, v, err := s.getVolume(ctx, k)
	if err != nil {
		return nil, -1, err
	}

	return v.GetFile(ctx, k)
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
	s.cache.Remove(k)
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

func (s *service) CreateReplica(ctx context.Context, key string, reader io.ReadCloser, ttl time.Duration, ca time.Time) (string, error) {
	if s.draining.Load() {
		return "", errors.New("node is draining")
	}

	if s.cfg.Replica == -1 {
		return "", errors.New("can not store replicas")
	}
	vls := s.getLocalVolumes(ctx, key)
	if len(vls) == 0 {
		return "", errors.New("no local volumes to store replica")
	}
	v := vls[0]
	err := v.CreateFile(ctx, key, reader, noReplica, ttl, ca)
	if err != nil {
		return "", err
	}

	return v.ID(), nil
}

func (s *service) UpdateFileReplica(ctx context.Context, key string, volumeIDs []string, replica int) error {
	if s.cfg.Replica == -1 {
		return errors.New("can not store replicas")
	}

	if len(s.members.LocalVolumes()) == 0 {
		return errors.New("no local volumes to update replica")
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

func (s *service) Drain(ctx context.Context) error {
	s.draining.Store(true)
	s.members.SetDraining(true)

	lvs := s.members.LocalVolumes()
	if len(lvs) == 0 {
		// proxy mode — nothing to drain
		s.logger.Info("drain: node is in proxy mode, nothing to drain")
		s.members.Leave()
		return nil
	}

	// Step 1: Create replica jobs for all under-replicated files
	s.logger.Info("drain: preparing replica jobs for all local files")
	for _, v := range lvs {
		if err := v.PrepareForDrain(ctx); err != nil {
			// Leave() is deliberately NOT called on failure — the caller (SIGQUIT handler)
			// decides what to do next; the node remains a cluster member so it can retry
			// or be manually shut down.
			return fmt.Errorf("drain: prepare failed: %w", err)
		}
	}

	// Step 2: Wait for all replica queues to drain
	s.logger.Info("drain: waiting for replication to complete")
	for {
		allDone := true
		for _, v := range lvs {
			pending, err := v.HasPendingReplicas(ctx)
			if err != nil {
				return fmt.Errorf("drain: checking replica queue: %w", err)
			}
			if pending {
				allDone = false
				break
			}
		}
		if allDone {
			break
		}
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Step 3: Purge local copies
	s.logger.Info("drain: purging local copies")
	for _, v := range lvs {
		if err := v.PurgeAllFiles(ctx); err != nil {
			// Leave() is deliberately NOT called on failure — the caller (SIGQUIT handler)
			// decides what to do next; the node remains a cluster member so it can retry
			// or be manually shut down.
			return fmt.Errorf("drain: purge failed: %w", err)
		}
	}

	// Step 4: Leave cluster
	s.logger.Info("drain: leaving cluster")
	s.members.Leave()
	return nil
}

func (s *service) GetReplicaInfo(ctx context.Context, key string) ([]string, int, error) {
	for _, v := range s.members.LocalVolumes() {
		_, ok, err := v.HasFile(ctx, key)
		if err != nil || !ok {
			continue
		}
		vids, replica, err := v.FileVolumeIDs(ctx, key)
		if err == nil {
			return vids, replica, nil
		}
	}
	return nil, 0, errors.New("not found")
}

// getLocalVolumes returns all local volumes in preferred write order.
// Currently shuffles randomly so no single volume is always favoured;
// future implementations may sort by available space, least-used, or other
// heuristics.
func (s *service) getLocalVolumes(_ context.Context, _ string) []volume.Local {
	vls := s.members.LocalVolumes()
	shuffled := make([]volume.Local, len(vls))
	copy(shuffled, vls)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
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
	m = <-doneC
	if m.v == nil {
		// If it's done without a value, means
		// that the doneC has ben closed and that
		// no volume was found
		err = errors.New("not found")
	}
	// Cancel all the possible still running
	// request to the volumes
	cfn()

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
