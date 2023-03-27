package state

import "context"

//go:generate mockgen -destination=../mock/state_repository.go -mock_names=Repository=StateRepository -package=mock github.com/xescugc/rebost/state Repository

// Repository are the actions that can be done to modify the volume State
type Repository interface {
	// Find returns the State, if there is no State it'll return an empty State
	Find(ctx context.Context) (*State, error)

	// Update updates the State
	Update(ctx context.Context, s *State) error

	// DeleteAll deletes all the state data
	DeleteAll(ctx context.Context) error
}
