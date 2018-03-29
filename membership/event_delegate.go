package membership

import (
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
	c, err := client.New(n.Address())
	if err != nil {
		panic(err)
	}
	e.members.remoteVolumes[n.Address()] = c
}

func (e *eventDelegate) NotifyLeave(n *memberlist.Node) {
	delete(e.members.remoteVolumes, n.Address())
}

func (e *eventDelegate) NotifyUpdate(n *memberlist.Node) {
	c, err := client.New(n.Address())
	if err != nil {
		panic(err)
	}
	e.members.remoteVolumes[n.Address()] = c
}
