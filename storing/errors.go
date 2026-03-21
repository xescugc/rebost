package storing

import "github.com/xescugc/rebost/volume"

// ErrClusterFull is returned by CreateFile when no volume in the cluster
// has enough capacity to store the incoming file.
var ErrClusterFull = volume.ErrClusterFull
