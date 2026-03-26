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

	// DefaultScrubInterval is the default interval between scrub passes.
	DefaultScrubInterval = 24 * time.Hour
	// DefaultReplicaCheckInterval is the default interval between background
	// replica-check passes that detect and re-queue under-replicated files.
	DefaultReplicaCheckInterval = 1 * time.Hour
	// DefaultReplicaConsistencyInterval is the default interval for replica
	// consistency checks that detect and purge stale non-owner replicas.
	DefaultReplicaConsistencyInterval = 1 * time.Hour

	// defaultNameLen is the default length of the
	// auto generated Node name if none defined
	defaultNameLen = 7
)

// Timing holds all time-based configuration.
type Timing struct {
	// VolumeDowntime is the maximum time a volume can be down
	// before the rest of the cluster try to rebalance
	// the lost of data, is the time we'll wait for it
	// to go back up again
	VolumeDowntime time.Duration `mapstructure:"volume-downtime"`

	// ScrubInterval is how often each volume checksums all its files and
	// enqueues repair jobs for any that fail. Defaults to 24h.
	ScrubInterval time.Duration `mapstructure:"scrub-interval"`

	// ReplicaCheckInterval is how often each volume checks whether all
	// locally-owned files have the required number of replicas and re-queues
	// replication jobs for any that are under-replicated. Defaults to 1h.
	ReplicaCheckInterval time.Duration `mapstructure:"replica-check-interval"`

	// ReplicaConsistencyInterval is how often non-owner replicas verify that
	// they are still expected by the owner. Defaults to 1h.
	ReplicaConsistencyInterval time.Duration `mapstructure:"replica-consistency-interval"`
}

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

	// Name is the name the Node will have inside of the Memberlist
	Name string `mapstructure:"name"`

	Timing Timing

	Cache Cache

	Memberlist Memberlist

	Dashboard Dashboard

	S3 S3

	// Tags is a map of arbitrary key=value labels for this node.
	// Set at startup via --tag flags; propagated via gossip.
	Tags map[string]string `mapstructure:"tags"`
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

// S3 is the configuration required for S3-compatible API support
type S3 struct {
	// AccessKey is the AWS access key ID for Signature V4 authentication.
	// If empty, authentication is disabled and all requests are accepted.
	AccessKey string `mapstructure:"access_key"`

	// SecretKey is the AWS secret access key for Signature V4 authentication.
	SecretKey string `mapstructure:"secret_key"`

	// AuthMode controls which operations require authentication.
	// Valid values: "all" (default) — auth required for all requests;
	//               "write" — auth required only for write operations (PUT, DELETE, PATCH, POST).
	// Has no effect when AccessKey is empty.
	AuthMode string `mapstructure:"auth_mode"`
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
	v.SetDefault("timing.volume-downtime", DefaultVolumeDowntime)
	v.SetDefault("timing.scrub-interval", DefaultScrubInterval)
	v.SetDefault("timing.replica-check-interval", DefaultReplicaCheckInterval)
	v.SetDefault("timing.replica-consistency-interval", DefaultReplicaConsistencyInterval)
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

	if cfg.Timing.VolumeDowntime < volume.TickerDuration {
		return nil, fmt.Errorf("the volume-downtime cannot be lower than %s", volume.TickerDuration)
	}

	return &cfg, nil
}
