package membership

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/state"
)

func TestNodesWithoutVolumeIDsSortedByFreeSpace(t *testing.T) {
	cA, err := client.New("localhost:1")
	require.NoError(t, err)
	cB, err := client.New("localhost:2")
	require.NoError(t, err)
	cC, err := client.New("localhost:3")
	require.NoError(t, err)

	m := &Membership{
		nodes: map[string]node{
			"sort-a": {
				conn: cA,
				meta: Metadata{Status: StatusRunning, Volumes: map[string]struct{}{}},
				state: State{Volumes: map[string]state.State{
					"sort-vol-a": {VolumeTotalSize: 1000, VolumeUsedSize: 900}, // 100 free
				}},
			},
			"sort-b": {
				conn: cB,
				meta: Metadata{Status: StatusRunning, Volumes: map[string]struct{}{}},
				state: State{Volumes: map[string]state.State{
					"sort-vol-b": {VolumeTotalSize: 1000, VolumeUsedSize: 200}, // 800 free
				}},
			},
			"sort-c": {
				conn: cC,
				meta: Metadata{Status: StatusRunning, Volumes: map[string]struct{}{}},
				state: State{Volumes: map[string]state.State{
					"sort-vol-c": {VolumeTotalSize: 1000, VolumeUsedSize: 600}, // 400 free
				}},
			},
		},
	}

	ns := m.NodesWithoutVolumeIDs([]string{})
	require.Len(t, ns, 3)
	assert.Equal(t, cB, ns[0], "first: sort-b (800 free)")
	assert.Equal(t, cC, ns[1], "second: sort-c (400 free)")
	assert.Equal(t, cA, ns[2], "third: sort-a (100 free)")
}
