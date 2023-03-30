package config_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/volume"
)

func TestNew(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		cfg, err := config.New(viper.New())
		require.NoError(t, err)
		assert.NotEmpty(t, cfg.Name)
		assert.NotEmpty(t, cfg.Memberlist.Port)
		assert.Equal(t, config.DefaultReplica, cfg.Replica)
		assert.Equal(t, config.DefaultCacheSize, cfg.Cache.Size)
		assert.Equal(t, config.DefaultVolumeDowntime, cfg.VolumeDowntime)
	})
	t.Run("InvalidVolumeDowntime", func(t *testing.T) {
		v := viper.New()
		v.Set("volume-downtime", 20*time.Second)
		_, err := config.New(v)
		assert.EqualError(t, err, fmt.Sprintf("the volume-downtime cannot be lower than %s", volume.TickerDuration))
	})
}
