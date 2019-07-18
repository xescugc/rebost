package membership

import "encoding/json"

type delegate struct {
	members *Membership
}

func (d *delegate) NodeMeta(limit int) []byte {
	m := metadata{Port: d.members.cfg.Port}
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
	return nil
}

func (d *delegate) MergeRemoteState(buf []byte, join bool) {}
