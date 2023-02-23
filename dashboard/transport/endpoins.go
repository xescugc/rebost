package transport

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/dashboard"
)

// Endpoints is the list of all the endpoints of the Dashboard
type Endpoints struct {
	Home endpoint.Endpoint
}

// MakeServerEndpoints initializes the Endpoints of the Dashboard
func MakeServerEndpoints(s dashboard.Service) Endpoints {
	return Endpoints{
		Home: MakeHomeEndpoint(s),
	}
}

// HomeRequest defines the request for the Home page
type HomeRequest struct {
}

// HomeResponse defines the response of the Home page
type HomeResponse struct {
	Nodes []*config.Config
	Err   error
}

// MakeHomeEndpoint has the logic to get the needed information for the Home Page
func MakeHomeEndpoint(s dashboard.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		_ = request.(HomeRequest)
		ns, err := s.ListNodes(ctx)
		return HomeResponse{Nodes: ns, Err: err}, nil
	}
}
