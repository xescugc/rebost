package dashboard

import (
	"context"
	"fmt"

	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/storing"

	kitlog "github.com/go-kit/kit/log"
)

// Service exposes the Dashboard service interface
type Service interface {
	// ListNodes returns the list of all the nodes configuration
	ListNodes(context.Context) ([]*config.Config, error)
}

type service struct {
	members storing.Membership
	cfg     *config.Config

	logger kitlog.Logger
}

// New returns an implementation of the Dashboard with
// the given parameters
func New(cfg *config.Config, m storing.Membership, logger kitlog.Logger) Service {
	return &service{
		members: m,
		cfg:     cfg,

		logger: kitlog.With(logger, "src", "dashboard", "name", cfg.Name),
	}
}

func (s *service) ListNodes(ctx context.Context) ([]*config.Config, error) {
	// As the Nodes only have the other nodes not the current, we append the
	// current node as a first thing
	cfgs := []*config.Config{s.cfg}
	for _, n := range s.members.Nodes() {
		cfg, err := n.Config(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get config: %w", err)
		}
		cfgs = append(cfgs, cfg)
	}
	return cfgs, nil
}
