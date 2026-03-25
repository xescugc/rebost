package membership

// Status represents the lifecycle state of a node.
type Status string

const (
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusDraining Status = "draining"
)
