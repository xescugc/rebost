package transport

import (
	"github.com/xescugc/rebost/dashboard"
)

// HomeResponse defines the response of the Home page
type HomeResponse struct {
	Nodes []*dashboard.Node
	Err   error
}
