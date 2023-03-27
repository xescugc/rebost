package membership

import (
	"encoding/json"
	"net"
	"strconv"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/state"
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

	var meta Metadata
	err := json.Unmarshal(n.Meta, &meta)
	if err != nil {
		panic(err)
	}

	url := net.JoinHostPort(n.Addr.String(), strconv.Itoa(meta.Port))
	c, err := client.New(url)
	if err != nil {
		panic(err)
	}

	nn := node{
		conn: c,
		meta: meta,
		state: State{
			Volumes: make(map[string]state.State),
		},
	}

	e.members.nodesLock.Lock()
	e.members.nodes[n.Name] = nn
	e.members.logger.Log("action", "join", "name", n.Name, "url", url)
	e.members.nodesLock.Unlock()

	// We remove any vid that was marked as to be deleted
	e.members.removedVolumeIDsLock.Lock()
	for vid := range meta.Volumes {
		delete(e.members.removedVolumeIDs, vid)
	}
	e.members.removedVolumeIDsLock.Unlock()
}

func (e *eventDelegate) NotifyLeave(n *memberlist.Node) {
	e.members.nodesLock.Lock()
	e.members.removedVolumeIDsLock.Lock()

	nn := e.members.nodes[n.Name]
	for vid := range nn.meta.Volumes {
		e.members.removedVolumeIDs[vid] = time.Now()
	}
	delete(e.members.nodes, n.Name)
	e.members.logger.Log("action", "leave", "name", n.Name)

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
