package membership

import "github.com/xescugc/rebost/state"

// State holds the node state which will be notified to the other Nodes
type State struct {
	// Node is the name of the Node
	Node string `json:"node"`

	// Volumes is the list of volumes of the Node with the State
	// each one have
	Volumes map[string]state.State `json:"volume_ids"`
}
