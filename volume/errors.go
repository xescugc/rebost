package volume

import "errors"

// ErrNoSpace is returned by CreateFile when the volume has no capacity
// for the incoming file.
var ErrNoSpace = errors.New("no space left on volume")

// ErrClusterFull is returned by storing.Service.CreateFile when no volume
// in the cluster has enough capacity for the incoming file. It is defined
// here (rather than in package storing) to avoid a test-only import cycle:
// storing/replica_test.go (package storing) imports storing/transport/http,
// which would cycle back if transport imported storing.
var ErrClusterFull = errors.New("cluster has no available storage capacity")
