package membership

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"

	kitlog "github.com/go-kit/kit/log"
	"github.com/hashicorp/memberlist"
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/storing"
	"github.com/xescugc/rebost/volume"
)

// Membership handles all the logic of the Node
// persistentce and also the localVolumes
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
	// nodes leaving the cluster
	removedVolumeIDs     []string
	removedVolumeIDsLock sync.Mutex

	logger kitlog.Logger
}

// node represents a Node in the cluseter, with the metadata (meta)
// and the connection (conn) to it
type node struct {
	conn storing.Service
	meta metadata
}

// New returns an implementation of the Membership interface
func New(cfg *config.Config, lv []volume.Local, remote string, logger kitlog.Logger) (*Membership, error) {
	m := &Membership{
		localVolumes:     lv,
		nodes:            make(map[string]node),
		cfg:              cfg,
		removedVolumeIDs: make([]string, 0),
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

// GetNodeWithVolumeByID returns the Node/storing.Service that has
// the gicen vid
func (m *Membership) GetNodeWithVolumeByID(vid string) (storing.Service, error) {
	m.nodesLock.RLock()
	defer m.nodesLock.RUnlock()
	for _, n := range m.nodes {
		for _, nvid := range n.meta.VolumeIDs {
			if nvid == vid {
				return n.conn, nil
			}
		}
	}

	return nil, errors.New("not found")
}

// Nodes return all the nodes of the Cluster
func (m *Membership) Nodes() (res []storing.Service) {
	m.nodesLock.RLock()
	for _, r := range m.nodes {
		res = append(res, r.conn)
	}
	m.nodesLock.RUnlock()

	return
}

// RemovedVolumeIDs returns the list of removed VolumeIDs from
// the cluser.
// WARNING: Each call to it empties the list so the list
// of nodes have to be stored/used once called
func (m *Membership) RemovedVolumeIDs() []string {
	m.removedVolumeIDsLock.Lock()

	rvids := make([]string, 0, len(m.removedVolumeIDs))
	for _, vid := range m.removedVolumeIDs {
		rvids = append(rvids, vid)
	}
	m.removedVolumeIDs = make([]string, 0)

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
