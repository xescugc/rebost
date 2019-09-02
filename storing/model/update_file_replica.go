package model

// UpdateFileReplica it's the body of the UpdateFileReplica
type UpdateFileReplica struct {
	VolumeIDs []string `json:"volume_ids"`
	Replica   int      `json:"replica"`
}
