package config

import (
	"net"
	"strconv"
	"strings"

	"github.com/spf13/viper"
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
	Replica int

	// MaxReplicaPendent is the max number of
	// replications that it can be holding/accept
	// at a given time
	MaxReplicaPendent int

	MemberlistBindPort int
	MemberlistName     string
}

// New returns a new Config from the viper.Viper, the ENV variables
// are readed by using the convertion of "_" and all caps
func New(v *viper.Viper) (*Config, error) {
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	// Opens a TCP connection to a free port on the host
	// and closes the connection but getting the port from it
	// so the 'memberlist-bind-port' can be setted to a free
	// random port each time if no one is specified
	l, err := net.Listen("tcp", "")
	if err != nil {
		return nil, err
	}
	l.Close()
	sl := strings.Split(l.Addr().String(), ":")
	mbp, err := strconv.Atoi(sl[len(sl)-1])
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
		Port:              v.GetInt("port"),
		Volumes:           v.GetStringSlice("volumes"),
		Remote:            v.GetString("remote"),
		Replica:           v.GetInt("replica"),
		MaxReplicaPendent: v.GetInt("max-replica-pendent"),

		MemberlistBindPort: v.GetInt("memberlist-bind-port"),
		MemberlistName:     v.GetString("memberlist-name"),
	}, nil
}
