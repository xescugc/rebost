package config

import (
	"strings"

	"github.com/spf13/viper"
	"github.com/xescugc/rebost/util"
)

const defaultMemberlistNameLen = 5

// Config represents the struct whith all the possible
// configuration options
type Config struct {
	// Port is the port to open to the public
	Port int `mapstructure:"port"`

	// Volumes is th list of the local volumes
	Volumes []string `mapstructure:"volumes"`

	// Remote is the URL of another Node
	Remote string `mapstructure:"remote"`

	// Replica is the default number of replicas
	// that each file will have if none specified
	// If set to -1 it'll not try to replicate any
	// of the created files and it'll not store any
	// replica from another Node
	Replica int `mapstructure:"replica"`

	MemberlistBindPort int    `mapstructure:"memberlist-bind-port`
	MemberlistName     string `mapstructure:"memberlist-name"`
}

// New returns a new Config from the viper.Viper, the ENV variables
// are readed by using the convertion of "_" and all caps
func New(v *viper.Viper) (*Config, error) {
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	mbp, err := util.FreePort()
	if err != nil {
		return nil, err
	}
	v.SetDefault("memberlist-bind-port", mbp)

	name := utils.RandomString(defaultMemberlistNameLen)
	v.SetDefault("memberlist-name", name)

	if v.GetString("config") != "" {
		v.SetConfigFile(v.GetString("config"))
		err = v.ReadInConfig()
		if err != nil {
			return nil, err
		}
	}

	var cfg Config
	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
