package state_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xescugc/rebost/state"
)

func TestCanStore(t *testing.T) {
	tests := []struct {
		name   string
		state  state.State
		param  int
		result bool
	}{
		{
			name: "SuccessBySystem",
			state: state.State{
				SystemTotalSize: 10,
				SystemUsedSize:  5,
				VolumeTotalSize: -1,
			},
			param:  5,
			result: true,
		},
		{
			name: "SuccessByVolume",
			state: state.State{
				SystemTotalSize: 20,
				SystemUsedSize:  5,
				VolumeTotalSize: 10,
				VolumeUsedSize:  2,
			},
			param:  5,
			result: true,
		},
		{
			name: "FailBySystem",
			state: state.State{
				SystemTotalSize: 10,
				SystemUsedSize:  5,
				VolumeTotalSize: -1,
			},
			param:  10,
			result: false,
		},
		{
			name: "FailByVolume",
			state: state.State{
				SystemTotalSize: 20,
				SystemUsedSize:  5,
				VolumeTotalSize: 10,
				VolumeUsedSize:  2,
			},
			param:  10,
			result: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok := tt.state.CanStore(tt.param)
			assert.Equal(t, tt.result, ok)
		})
	}
}

func TestTotalSize(t *testing.T) {
	s := state.State{SystemTotalSize: 10, VolumeTotalSize: -1}
	assert.Equal(t, s.SystemTotalSize, s.TotalSize())

	s = state.State{SystemTotalSize: 10, VolumeTotalSize: 5}
	assert.Equal(t, s.VolumeTotalSize, s.TotalSize())
}

func TestUsedSize(t *testing.T) {
	s := state.State{SystemUsedSize: 10, VolumeTotalSize: -1}
	assert.Equal(t, s.SystemUsedSize, s.UsedSize())

	s = state.State{SystemUsedSize: 10, VolumeTotalSize: 5, VolumeUsedSize: 3}
	assert.Equal(t, s.VolumeUsedSize, s.UsedSize())
}

func TestUse(t *testing.T) {
	tests := []struct {
		name   string
		state  state.State
		estate state.State
		param  int
		result bool
	}{
		{
			name: "SuccessBySystem",
			state: state.State{
				SystemTotalSize: 10,
				SystemUsedSize:  5,
				VolumeTotalSize: -1,
			},
			estate: state.State{
				SystemTotalSize: 10,
				SystemUsedSize:  10,
				VolumeTotalSize: -1,
				VolumeUsedSize:  5,
			},
			param:  5,
			result: true,
		},
		{
			name: "SuccessByVolume",
			state: state.State{
				SystemTotalSize: 20,
				SystemUsedSize:  5,
				VolumeTotalSize: 10,
				VolumeUsedSize:  2,
			},
			estate: state.State{
				SystemTotalSize: 20,
				SystemUsedSize:  10,
				VolumeTotalSize: 10,
				VolumeUsedSize:  7,
			},
			param:  5,
			result: true,
		},
		{
			name: "FailBySystem",
			state: state.State{
				SystemTotalSize: 10,
				SystemUsedSize:  5,
				VolumeTotalSize: -1,
			},
			estate: state.State{
				SystemTotalSize: 10,
				SystemUsedSize:  5,
				VolumeTotalSize: -1,
			},
			param:  10,
			result: false,
		},
		{
			name: "FailByVolume",
			state: state.State{
				SystemTotalSize: 20,
				SystemUsedSize:  5,
				VolumeTotalSize: 10,
				VolumeUsedSize:  2,
			},
			estate: state.State{
				SystemTotalSize: 20,
				SystemUsedSize:  5,
				VolumeTotalSize: 10,
				VolumeUsedSize:  2,
			},
			param:  10,
			result: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok := tt.state.Use(tt.param)
			assert.Equal(t, tt.result, ok)
			assert.Equal(t, tt.estate, tt.state)
		})
	}
}
