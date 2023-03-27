package state

import (
	"time"
)

// State is the current state in which the volume is
type State struct {
	// The Mountpoint of the volume, so we can make
	// sure it's that if it's also used by
	// another volume we do not repeat Stats
	Mountpoint string

	SystemTotalSize int

	SystemUsedSize int

	// TotalSize is the total size of the volume
	// if not specified then this value will be -1
	VolumeTotalSize int

	// UsedSize is the total used size of the volume objects
	VolumeUsedSize int

	// UpdatedAt is useful to be able to know on restart
	// how long has it been since the last check, it's like
	// a heartbeat
	UpdatedAt time.Time
}

// CanStore will check if the b bytes fit into the defined sizes
// to prevent over sizing
func (s *State) CanStore(b int) bool {
	if s.UsedSize()+b > s.TotalSize() {
		return false
	}
	return true
}

// TotalSize returns the total size depending if the VolumeTotalSize
// is set or not
func (s *State) TotalSize() int {
	if s.VolumeTotalSize == -1 {
		return s.SystemTotalSize
	}
	return s.VolumeTotalSize
}

// UsedSize returns the used size depending if the VolumeTotalSize
// is set or not
func (s *State) UsedSize() int {
	if s.VolumeTotalSize == -1 {
		return s.SystemUsedSize
	}
	return s.VolumeUsedSize
}

// Use will try to set b if it fits
func (s *State) Use(b int) bool {
	if s.CanStore(b) {
		s.SystemUsedSize += b
		s.VolumeUsedSize += b
		return true
	}
	return false
}

// IsInDowntimeRange will check if the s.UpdatedAt plus the duration
// is older than the current date, meaning it's not on range
func (s *State) IsInDowntimeRange(d time.Duration) bool {
	return s.UpdatedAt.Add(d).Before(time.Now())
}
