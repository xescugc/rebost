package membership

import "encoding/json"

type delegate struct {
	members *Membership
}

func (d *delegate) NodeMeta(limit int) []byte {
	m := metadata{
		Port:      d.members.cfg.Port,
		VolumeIDs: make([]string, 0, len(d.members.localVolumes)),
	}
	for _, v := range d.members.localVolumes {
		m.VolumeIDs = append(m.VolumeIDs, v.ID())
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
	return nil
}

func (d *delegate) MergeRemoteState(buf []byte, join bool) {}
