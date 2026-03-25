package model

// ReplicaInfo carries the volume IDs and replica count for a file,
// returned by the GET /replicas/{key} endpoint.
type ReplicaInfo struct {
	VolumeIDs []string `json:"volume_ids"`
	Replica   int      `json:"replica"`
}
