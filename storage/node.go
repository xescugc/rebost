package storage

const (
	StatusLeader = iota
	StatusFollower
	StatusCandidate
	StatusDraining
)

type Node interface {
	// Start the node concensus
	//Start()
	Heartbeat() Heartbeat

	Replicate(fileKey string, nodeID uint32)
}

//IO status
//Status
//Pending Replications
//The answer to the request of the heartbeat (if it brings information)
type Heartbeat struct{}
