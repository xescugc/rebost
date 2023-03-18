package membership

import (
	"context"
	"encoding/json"

	"github.com/xescugc/rebost/state"
)

type delegate struct {
	members *Membership
}

func (d *delegate) NodeMeta(limit int) []byte {
	m := Metadata{
		Port:    d.members.cfg.Port,
		Volumes: make(map[string]struct{}),
	}
	for _, v := range d.members.localVolumes {
		m.Volumes[v.ID()] = struct{}{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return b
}

func (d *delegate) NotifyMsg([]byte) {}

func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return nil
}

func (d *delegate) LocalState(join bool) []byte {
	s := State{
		Node:    d.members.cfg.Name,
		Volumes: make(map[string]state.State),
	}
	for _, v := range d.members.localVolumes {
		vs, err := v.GetState(context.Background())
		if err != nil {
			vs = &state.State{}
		}
		s.Volumes[v.ID()] = *vs
	}
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return b
}

func (d *delegate) MergeRemoteState(buf []byte, join bool) {
	var s State
	_ = json.Unmarshal(buf, &s)
	_ = d.members.updateNodeState(s)
}
