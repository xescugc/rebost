package state

import "context"

//go:generate mockgen -destination=../mock/state_repository.go -mock_names=Repository=StateRepository -package=mock github.com/xescugc/rebost/state Repository

// Repository are the actions that can be done to modify the volume State
type Repository interface {
	// Find returns the State, if there is no State it'll return an empty State
	Find(ctx context.Context, vid string) (*State, error)

	// Update updates the State
	Update(ctx context.Context, vid string, s *State) error
}
