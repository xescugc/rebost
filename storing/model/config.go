package model

import "github.com/xescugc/rebost/config"

// Config is the transport representation of the config.Config
type Config struct {
	Port    int      `json:"port"`
	Volumes []string `json:"volumes"`
	Remote  string   `json:"remote"`
	Replica int      `json:"int"`

	Name string `json:"name"`

	Memberlist ConfigMemberlist `json:"memberlist"`

	Dashboard ConfigDashboard `json:"dashboard"`
}

// ConfigMemberlist is the set  of configuration required for the memberlist,
type ConfigMemberlist struct {
	Port int `json:"port"`
}

// ConfigDashboard is the configuration required for the dashboard
type ConfigDashboard struct {
	Port    int  `json:"port"`
	Enabled bool `json:"enabled"`
}

// ToConfig converts a model.Config to a config.Config
func ToConfig(c Config) *config.Config {
	return &config.Config{
		Port:    c.Port,
		Volumes: c.Volumes,
		Remote:  c.Remote,
		Replica: c.Replica,
		Name:    c.Name,
		Memberlist: config.Memberlist{
			Port: c.Memberlist.Port,
		},
		Dashboard: config.Dashboard{
			Port:    c.Dashboard.Port,
			Enabled: c.Dashboard.Enabled,
		},
	}
}

// ConfigToModel converts a config.Config to a model.Config
func ConfigToModel(c *config.Config) Config {
	return Config{
		Port:    c.Port,
		Volumes: c.Volumes,
		Remote:  c.Remote,
		Replica: c.Replica,
		Name:    c.Name,
		Memberlist: ConfigMemberlist{
			Port: c.Memberlist.Port,
		},
		Dashboard: ConfigDashboard{
			Port:    c.Dashboard.Port,
			Enabled: c.Dashboard.Enabled,
		},
	}
}
