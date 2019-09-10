package config_test

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/config"
)

func TestNew(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		cfg, err := config.New(viper.New())
		require.NoError(t, err)
		assert.NotEmpty(t, cfg.MemberlistName)
		assert.NotEmpty(t, cfg.MemberlistBindPort)
		assert.Equal(t, config.DefaultReplica, cfg.Replica)
	})
}
