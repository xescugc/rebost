package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/util"
)

func TestFreePort(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		p1, err := util.FreePort()
		require.NoError(t, err)
		p2, err := util.FreePort()
		require.NoError(t, err)

		assert.NotEqual(t, p1, p2)
	})
}
