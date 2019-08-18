package model

type UpdateFileReplica struct {
	VolumeIDs []string `json:"volume_ids"`
	Replica   int      `json:"replica"`
}
