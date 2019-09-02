package idxvolume_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xescugc/rebost/idxvolume"
)

func TestAddSignature(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		idxv := idxvolume.New("key", []string{"value"})

		idxv.AddSignature("value2")

		assert.Equal(t, []string{"value", "value2"}, idxv.Signatures)

		idxv.AddSignature("value")
		idxv.AddSignature("value2")

		assert.Equal(t, []string{"value", "value2"}, idxv.Signatures)
	})
}
