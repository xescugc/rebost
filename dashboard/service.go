package dashboard

import (
	"context"
	"fmt"
	"sort"

	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/membership"
	"github.com/xescugc/rebost/state"
	"github.com/xescugc/rebost/storing"

	"log/slog"
)

// Service exposes the Dashboard service interface
type Service interface {
	// ListNodes returns the list of all the nodes configuration
	ListNodes(context.Context) ([]*Node, error)
}

type service struct {
	members storing.Membership
	cfg     *config.Config

	logger *slog.Logger
}

// Node defines the aggregation of other entities
// to represent a Node we need here
type Node struct {
	Config config.Config
	State  membership.State
}

// New returns an implementation of the Dashboard with
// the given parameters
func New(cfg *config.Config, m storing.Membership, logger *slog.Logger) Service {
	return &service{
		members: m,
		cfg:     cfg,

		logger: logger.With("src", "dashboard", "name", cfg.Name),
	}
}

func (s *service) ListNodes(ctx context.Context) ([]*Node, error) {
	// As the Nodes only have the other nodes not the current, we append the
	// current node as a first thing
	nodes := []*Node{
		&Node{
			Config: *s.cfg,
			State: membership.State{
				Volumes: make(map[string]state.State),
			},
		},
	}

	for _, lv := range s.members.LocalVolumes() {
		s, err := lv.GetState(ctx)
		if err != nil {
			return nil, err
		}
		nodes[0].State.Volumes[lv.ID()] = *s
	}

	for _, n := range s.members.Nodes() {
		cfg, err := n.Config(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get config: %w", err)
		}
		st, err := s.members.GetNodeState(cfg.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get metadata: %w", err)
		}
		nodes = append(nodes, &Node{
			Config: *cfg,
			State:  *st,
		})
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Config.Name > nodes[j].Config.Name
	})

	return nodes, nil
}
