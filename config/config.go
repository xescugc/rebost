package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config represents the struct whith all the possible
// configuration options
type Config struct {
	Port    string
	Volumes []string
	Remote  string

	MemberlistBindPort int
	MemberlistName     string
}

// New returns a new Config from the viper.Viper, the ENV variables
// are readed by using the convertion of "_" and all caps
func New(v *viper.Viper) (*Config, error) {
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	if v.GetString("config") != "" {
		v.SetConfigFile(v.GetString("config"))
		err := v.ReadInConfig()
		if err != nil {
			return nil, err
		}
	}

	return &Config{
		Port:    v.GetString("port"),
		Volumes: v.GetStringSlice("volumes"),
		Remote:  v.GetString("remote"),

		MemberlistBindPort: v.GetInt("memberlist-bind-port"),
		MemberlistName:     v.GetString("memberlist-name"),
	}, nil
}
