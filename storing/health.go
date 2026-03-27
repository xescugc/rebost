package storing

import (
	"context"
	"fmt"
)

func (s *service) Health(ctx context.Context) error {
	for _, v := range s.members.LocalVolumes() {
		if _, err := v.GetState(ctx); err != nil {
			return fmt.Errorf("volume %s unhealthy: %w", v.ID(), err)
		}
	}
	return nil
}
