package membership

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"

	"github.com/hashicorp/memberlist"
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/storing"
	"github.com/xescugc/rebost/volume"
)

type Membership struct {
	members *memberlist.Memberlist
	events  *memberlist.EventDelegate

	localVolumes []volume.Local
	cfg          *config.Config

	nodesLock sync.RWMutex
	nodes     map[string]storing.Service
}

// New returns an implementation of the Membership interface
func New(cfg *config.Config, lv []volume.Local, remote string) (*Membership, error) {
	m := &Membership{
		localVolumes: lv,
		nodes:        make(map[string]storing.Service),
		cfg:          cfg,
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

		_, err = list.Join([]string{net.JoinHostPort(host, strconv.Itoa(cfg.MemberlistBindPort))})
		if err != nil {
			return nil, fmt.Errorf("Failed to join cluster: %s", err.Error())
		}
	}

	return m, nil
}

// LocalVolumes returns all the local volumes
func (m *Membership) LocalVolumes() []volume.Local {
	return m.localVolumes
}

// Services return all the nodes of the Cluster
func (m *Membership) Nodes() (res []storing.Service) {
	m.nodesLock.RLock()
	for _, r := range m.nodes {
		res = append(res, r)
	}
	m.nodesLock.RUnlock()

	return
}

func (m *Membership) Leave() {
	m.members.Leave(0)
}

func (m *Membership) buildConfig(cfg *config.Config) *memberlist.Config {
	mcfg := memberlist.DefaultLocalConfig()
	if cfg.MemberlistBindPort != 0 {
		mcfg.BindPort = cfg.MemberlistBindPort
	}
	mcfg.Name = cfg.MemberlistName
	mcfg.Events = &eventDelegate{members: m}
	mcfg.Delegate = &delegate{members: m}
	return mcfg
}
