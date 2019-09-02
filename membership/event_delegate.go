package membership

import (
	"encoding/json"
	"net"
	"strconv"

	"github.com/hashicorp/memberlist"
	"github.com/xescugc/rebost/client"
)

type eventDelegate struct {
	members *Membership
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

	nn := node{
		conn: c,
		meta: meta,
	}

	e.members.nodesLock.Lock()
	e.members.nodes[n.Name] = nn
	e.members.nodesLock.Unlock()
}

func (e *eventDelegate) NotifyLeave(n *memberlist.Node) {
	e.members.nodesLock.Lock()
	e.members.removedVolumeIDsLock.Lock()

	nn := e.members.nodes[n.Name]
	e.members.removedVolumeIDs = append(e.members.removedVolumeIDs, nn.meta.VolumeIDs...)
	delete(e.members.nodes, n.Name)

	e.members.nodesLock.Unlock()
	e.members.removedVolumeIDsLock.Unlock()
}

func (e *eventDelegate) NotifyUpdate(n *memberlist.Node) {
	// For now we do not have any use case
	// for update so it's basically the
	// same logic as Join.
	// The update it's only triggered when
	// the node.Meta has changed

	// TODO: Check if it has been any VolumeID deleted from the
	// Node Updated, if it has it should be added to the removedVolumeIDs
	e.NotifyJoin(n)
}
