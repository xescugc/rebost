package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/xescugc/rebost/util"
	"github.com/xescugc/rebost/volume"
	"github.com/xyproto/randomstring"
)

const (
	// DefaultPort is the default value the port has
	DefaultPort = 3805

	// DefaultReplica is the default number of replicas use
	// if none is defined
	DefaultReplica = 3

	// DefaultCacheSize is the default size of the cache
	DefaultCacheSize = 200

	// DefaultVolumeDowntime is the default time
	// a Volume can be down before start replicating
	DefaultVolumeDowntime = 2 * time.Minute

	// defaultNameLen is the default length of the
	// auto generated Node name if none defined
	defaultNameLen = 7
)

// Config represents the struct with all the possible
// configuration options
type Config struct {
	// Port is the port to open to the public
	Port int `mapstructure:"port"`

	// Volumes is the list of the local volumes
	Volumes []string `mapstructure:"volumes"`

	// Remote is the URL of another Node
	Remote string `mapstructure:"remote"`

	// Replica is the default number of replicas
	// that each file will have if none specified
	// If set to -1 it'll not try to replicate any
	// of the created files and it'll not store any
	// replica from another Node
	Replica int `mapstructure:"replica"`

	// VolumeDowntime is the maximum time a volume can be down
	// before the rest of the cluster try to rebalance
	// the lost of data, is the time we'll wait for it
	// to go back up again
	VolumeDowntime time.Duration `mapstructure:"volume-downtime"`

	// Name is the name the Node will have inside of the Memberlist
	Name string `mapstructure:"name"`

	Cache Cache

	Memberlist Memberlist

	Dashboard Dashboard
}

// Memberlist is the set  of configuration required for the memberlist,
// the name has  moved to the main part of the config as it's more clear there
type Memberlist struct {
	Port int `mapstructure:"port"`
}

// Dashboard is the configuration required for the dashboard
type Dashboard struct {
	Port    int  `mapstructure:"port"`
	Enabled bool `mapstructure:"enabled"`
}

// Cache is the configuration required for the cache
type Cache struct {
	Size int `mapstructure:"size"`
}

// New returns a new Config from the viper.Viper, the ENV variables
// are reade by using the convertion of "_" and all caps
func New(v *viper.Viper) (*Config, error) {
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.AutomaticEnv()

	mbp, err := util.FreePort()
	if err != nil {
		return nil, err
	}
	v.SetDefault("memberlist.port", mbp)

	v.SetDefault("port", DefaultPort)
	v.SetDefault("replica", DefaultReplica)
	v.SetDefault("volume-downtime", DefaultVolumeDowntime)
	v.SetDefault("cache.size", DefaultCacheSize)

	name := randomstring.HumanFriendlyEnglishString(defaultNameLen)
	v.SetDefault("name", name)

	if v.GetString("config") != "" {
		v.SetConfigFile(v.GetString("config"))
		err = v.ReadInConfig()
		if err != nil {
			return nil, err
		}
	}

	var cfg Config
	err = v.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}

	if cfg.VolumeDowntime < volume.TickerDuration {
		return nil, fmt.Errorf("the volume-downtime cannot be lower than %s", volume.TickerDuration)
	}

	return &cfg, nil
}
