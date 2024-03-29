package membership

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/hashicorp/memberlist"
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/volume"
)

// Membership handles all the logic of the Node
// persistence and also the localVolumes
type Membership struct {
	members *memberlist.Memberlist
	events  *memberlist.EventDelegate

	localVolumes []volume.Local
	cfg          *config.Config

	// Somehow improve this to make it easy
	// to search for Nodes by Volume
	nodesLock sync.RWMutex
	nodes     map[string]node

	// removedVolumeIDs list of all the volumeIDs removed by
	// nodes leaving the cluster and the time in which
	// this was detected
	removedVolumeIDs     map[string]time.Time
	removedVolumeIDsLock sync.Mutex

	logger kitlog.Logger
}

// node represents a Node in the cluster, with the Metadata (meta)
// and the connection (conn) to it
type node struct {
	conn  *client.Client
	meta  Metadata
	state State
}

// New returns an implementation of the Membership interface
func New(cfg *config.Config, lv []volume.Local, remote string, logger kitlog.Logger) (*Membership, error) {
	m := &Membership{
		localVolumes:     lv,
		nodes:            make(map[string]node),
		cfg:              cfg,
		removedVolumeIDs: make(map[string]time.Time),
		logger:           kitlog.With(logger, "src", "membership", "name", cfg.Name),
	}

	list, err := memberlist.Create(m.buildConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("Failed to create memberlist: %s", err.Error())
	}

	m.members = list

	if remote != "" {
		u, err := url.Parse(remote)
		if err != nil {
			return nil, err
		}

		if u.Scheme == "" {
			u.Scheme = "http"
		}

		c, err := client.New(u.String())
		if err != nil {
			return nil, err
		}

		cfg, err := c.Config(context.TODO())
		if err != nil {
			return nil, err
		}

		host, _, err := net.SplitHostPort(u.Host)
		if err != nil {
			return nil, err
		}

		hostPort := net.JoinHostPort(host, strconv.Itoa(cfg.Memberlist.Port))
		_, err = list.Join([]string{hostPort})
		if err != nil {
			return nil, fmt.Errorf("Failed to join cluster: %s", err.Error())
		}
		m.logger.Log("msg", fmt.Sprintf("Joined remote cluster %q", hostPort))
	}

	return m, nil
}

// LocalVolumes returns all the local volumes
func (m *Membership) LocalVolumes() []volume.Local {
	return m.localVolumes
}

// GetNodeWithVolumeByID returns the Node/client.Client that has
// the gicen vid
func (m *Membership) GetNodeWithVolumeByID(vid string) (*client.Client, error) {
	m.nodesLock.RLock()
	defer m.nodesLock.RUnlock()
	for _, n := range m.nodes {
		if _, ok := n.meta.Volumes[vid]; ok {
			return n.conn, nil
		}
	}

	return nil, errors.New("not found")
}

// GetNodeState returns the volume State
func (m *Membership) GetNodeState(nn string) (*State, error) {
	m.nodesLock.RLock()
	defer m.nodesLock.RUnlock()
	for kn, n := range m.nodes {
		if kn == nn {
			return &n.state, nil
		}
	}

	return nil, errors.New("not found")
}

func (m *Membership) updateNodeState(s State) error {
	m.nodesLock.RLock()
	defer m.nodesLock.RUnlock()
	if n, ok := m.nodes[s.Node]; ok {
		n.state = s
		m.nodes[s.Node] = n
		return nil
	}

	return errors.New("not found")
}

// Nodes return all the nodes of the Cluster
func (m *Membership) Nodes() (res []*client.Client) {
	m.nodesLock.RLock()
	for _, r := range m.nodes {
		res = append(res, r.conn)
	}
	m.nodesLock.RUnlock()

	return
}

// NodesWithoutVolumeIDs return all the nodes of the Cluster
func (m *Membership) NodesWithoutVolumeIDs(vids []string) (res []*client.Client) {
	m.nodesLock.RLock()
	for _, r := range m.nodes {
		var found bool
		for _, vid := range vids {
			if _, ok := r.meta.Volumes[vid]; ok {
				found = true
			}
		}
		if !found {
			res = append(res, r.conn)
		}
	}
	m.nodesLock.RUnlock()

	return
}

// RemovedVolumeIDs returns the list of removed VolumeIDs from
// the cluster.
// WARNING: Each call to it empties the list so the list
// of nodes have to be stored/used once called
func (m *Membership) RemovedVolumeIDs() []string {
	m.removedVolumeIDsLock.Lock()

	rvids := make([]string, 0, len(m.removedVolumeIDs))
	for vid, t := range m.removedVolumeIDs {
		if t.Add(m.cfg.VolumeDowntime).Before(time.Now()) {
			rvids = append(rvids, vid)
			delete(m.removedVolumeIDs, vid)
		}
	}

	m.removedVolumeIDsLock.Unlock()
	return rvids
}

// Leave makes the node leave the cluster
func (m *Membership) Leave() {
	m.members.Leave(0)
}

func (m *Membership) buildConfig(cfg *config.Config) *memberlist.Config {
	mcfg := memberlist.DefaultLocalConfig()
	if cfg.Memberlist.Port != 0 {
		mcfg.BindPort = cfg.Memberlist.Port
	}
	mcfg.Name = cfg.Name
	mcfg.Events = &eventDelegate{members: m}
	mcfg.Delegate = &delegate{members: m}
	return mcfg
}
