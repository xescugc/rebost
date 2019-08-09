package config

import (
	"strings"

	"github.com/spf13/viper"
	"github.com/xescugc/rebost/util"
)

// Config represents the struct whith all the possible
// configuration options
type Config struct {
	// Port is the port to open to the public
	Port int

	// Volumes is th list of the local volumes
	Volumes []string

	// Remote is the URL of another Node
	Remote string

	// Replica is the default number of replicas
	// that each file will have if none specified
	// If set to -1 it'll not try to replicate any
	// of the created files and it'll not store any
	// replica from another Node
	Replica int

	MemberlistBindPort int
	MemberlistName     string
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

	if v.GetString("config") != "" {
		v.SetConfigFile(v.GetString("config"))
		err := v.ReadInConfig()
		if err != nil {
			return nil, err
		}
	}

	return &Config{
		Port:    v.GetInt("port"),
		Volumes: v.GetStringSlice("volumes"),
		Remote:  v.GetString("remote"),
		Replica: v.GetInt("replica"),

		MemberlistBindPort: v.GetInt("memberlist-bind-port"),
		MemberlistName:     v.GetString("memberlist-name"),
	}, nil
}
