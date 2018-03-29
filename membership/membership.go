package membership

import (
	"fmt"

	"github.com/hashicorp/memberlist"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/volume"
)

//go:generate mockgen -destination=../mock/membership.go -mock_names=Membership=Membership -package=mock github.com/xescugc/rebost/membership Membership

// Membership is the interface that hides the logic behind the
// cluseter members. In this "domain" (rebost), the members
// are considered volume.Volume.
type Membership interface {
	// Volumes return all the Volumes of the cluster
	Volumes() []volume.Volume

	Leave()
}

type membership struct {
	members *memberlist.Memberlist
	events  *memberlist.EventDelegate

	localVolumes  []volume.Volume
	remoteVolumes map[string]volume.Volume
}

// New returns an implementation of the Membership interface
func New(cfg *config.Config, lv []volume.Volume, remote string) (Membership, error) {
	m := &membership{
		localVolumes:  lv,
		remoteVolumes: make(map[string]volume.Volume),
	}

	list, err := memberlist.Create(m.buildConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("Failed to create memberlist: %s", err.Error())
	}

	m.members = list

	if remote != "" {
		_, err = list.Join([]string{remote})
		if err != nil {
			return nil, fmt.Errorf("Failed to join cluster: %s", err.Error())
		}
	}

	return m, nil
}

// Volumes return all the volumes/nodes of the cluester
// it'll return the "localVolumes" first and then
// the "removeVolumes" but all will fulfill the
// volume.Volume interface so it's transparent for
// for the user
func (m *membership) Volumes() (res []volume.Volume) {
	res = append(res, m.localVolumes...)
	for _, r := range m.remoteVolumes {
		res = append(res, r)
	}
	return
}

func (m *membership) Leave() {
	m.members.Leave(0)
}

func (m *membership) buildConfig(cfg *config.Config) *memberlist.Config {
	mcfg := memberlist.DefaultLocalConfig()
	if cfg.MemberlistBindPort != 0 {
		mcfg.BindPort = cfg.MemberlistBindPort
	}
	if cfg.MemberlistName != "" {
		mcfg.Name = cfg.MemberlistName
	}
	mcfg.Events = &eventDelegate{members: m}
	return mcfg
}
