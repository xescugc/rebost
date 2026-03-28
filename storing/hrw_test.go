package storing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRankVolumeIDs(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		assert.Empty(t, rankVolumeIDs("key", nil))
	})
	t.Run("Single", func(t *testing.T) {
		assert.Equal(t, []string{"v1"}, rankVolumeIDs("key", []string{"v1"}))
	})
	t.Run("Deterministic", func(t *testing.T) {
		vids := []string{"v1", "v2", "v3"}
		r1 := rankVolumeIDs("testkey", vids)
		r2 := rankVolumeIDs("testkey", vids)
		assert.Equal(t, r1, r2)
	})
	t.Run("DifferentKeysCanProduceDifferentOrder", func(t *testing.T) {
		vids := []string{"va", "vb", "vc"}
		r1 := rankVolumeIDs("key1", vids)
		r2 := rankVolumeIDs("key2", vids)
		assert.ElementsMatch(t, vids, r1)
		assert.ElementsMatch(t, vids, r2)
	})
	t.Run("AllVidsPresent", func(t *testing.T) {
		vids := []string{"a", "b", "c", "d"}
		ranked := rankVolumeIDs("somekey", vids)
		assert.ElementsMatch(t, vids, ranked)
	})
}
