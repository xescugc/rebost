package membership

import (
	"encoding/json"
	"net"
	"strconv"

	"github.com/hashicorp/memberlist"
	"github.com/xescugc/rebost/client"
)

type eventDelegate struct {
	members *membership
}

func (e *eventDelegate) NotifyJoin(n *memberlist.Node) {
	// When a memberlist.Config is created it sends the JOIN event
	// for itself, we do not want to store the current node as we already
	// have the 'localVolumes' we just ignore the first node if we already
	// have not been initialized.
	if e.members.members == nil {
		return
	}

	var meta metadata
	err := json.Unmarshal(n.Meta, &meta)
	if err != nil {
		panic(err)
	}

	c, err := client.New(net.JoinHostPort(n.Addr.String(), strconv.Itoa(meta.Port)))
	if err != nil {
		panic(err)
	}

	e.members.remoteVolumesLock.Lock()
	e.members.remoteVolumes[n.Address()] = c
	e.members.remoteVolumesLock.Unlock()
}

func (e *eventDelegate) NotifyLeave(n *memberlist.Node) {
	e.members.remoteVolumesLock.Lock()
	delete(e.members.remoteVolumes, n.Address())
	e.members.remoteVolumesLock.Unlock()
}

func (e *eventDelegate) NotifyUpdate(n *memberlist.Node) {
	// For now we do not have any use case
	// for update so it's basically the
	// same logic as Join.
	// The update it's only triggered when
	// the node.Meta has changed
	e.NotifyJoin(n)
}
