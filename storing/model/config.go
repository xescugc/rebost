package model

// Config is the transport representation of the config.Config
type Config struct {
	Port              int      `json:"port"`
	Volumes           []string `json:"volumes"`
	Remote            string   `json:"remote"`
	Replica           int      `json:"int"`
	MaxReplicaPendent int      `json:"max_replica_pendent"`

	MemberlistBindPort int    `json:"memberlist_bind_port"`
	MemberlistName     string `json:"memberlist_name"`
}
