package storing

import (
	"time"

	"github.com/xescugc/rebost/volume"
)

// loopVolumes handles all per-volume background operations for every local volume.
// New operation types (e.g. garbage collection) can be added here as additional
// processNext* helpers without spawning new top-level goroutines.
func (s *service) loopVolumes() {
	for {
		var workDone bool
		select {
		case <-s.ctx.Done():
			return
		default:
			for _, v := range s.members.LocalVolumes() {
				if s.processNextReplica(v) {
					workDone = true
				}
				if s.processNextDeletion(v) {
					workDone = true
				}
			}
		}
		// If nothing was done on one full pass, sleep to avoid busy-looping.
		if !workDone {
			time.Sleep(time.Second)
		}
	}
}

// processNextReplica picks up and handles one pending replication job for v.
// Returns true if a replication was attempted.
func (s *service) processNextReplica(v volume.Local) bool {
	rp, err := v.NextReplica(s.ctx)
	if err != nil {
		if err.Error() != "not found" {
			s.logger.Log("msg", err.Error())
		}
		return false
	}
	s.logger.Log("msg", "trying to replicate file", "key", rp.Key, "count", rp.Count)
	for _, n := range s.members.NodesWithoutVolumeIDs(rp.VolumeIDs) {
		_, ok, err := n.HasFile(s.ctx, rp.Key)
		if err != nil {
			s.logger.Log("msg", err.Error())
			continue
		}
		// If the volume already has this key we ignore it
		// It means the replica is already on that Node
		// Or that it has a file with that name
		// TODO: https://github.com/xescugc/rebost/issues/55
		if ok {
			s.logger.Log("msg", "file already present on this volume")
			continue
		}
		iorc, err := v.GetFile(s.ctx, rp.Key)
		if err != nil {
			s.logger.Log("msg", err.Error())
			continue
		}
		vID, err := n.CreateReplica(s.ctx, rp.Key, iorc, rp.TTL, rp.CreatedAt)
		if err != nil {
			s.logger.Log("msg", err.Error())
			continue
		}
		s.logger.Log("msg", "replica created")

		rp.VolumeIDs = append(rp.VolumeIDs, vID)

		for _, vid := range rp.VolumeIDs {
			// If is this node we do not need to do it
			// as it's the owner master
			if vid == v.ID() {
				continue
			}
			n, err := s.members.GetNodeWithVolumeByID(vid)
			if err != nil {
				s.logger.Log("msg", err.Error())
				continue
			}
			err = n.UpdateFileReplica(s.ctx, rp.Key, rp.VolumeIDs, rp.OriginalCount)
			if err != nil {
				s.logger.Log("msg", err.Error())
				continue
			}
		}

		err = v.UpdateReplica(s.ctx, rp, vID)
		if err != nil {
			s.logger.Log("msg", err.Error())
			continue
		}

		// As it has replicated the rp from NextReplica to one of the Nodes
		// we can exit the loop
		return true
	}
	return false
}

// processNextDeletion picks up and handles one pending remote-deletion job for v.
// Returns true if a deletion was processed.
func (s *service) processNextDeletion(v volume.Local) bool {
	d, err := v.NextDeletion(s.ctx)
	if err != nil {
		if err.Error() != "not found" {
			s.logger.Log("msg", err.Error())
		}
		return false
	}
	s.logger.Log("msg", "propagating delete to replicas", "key", d.Key)
	for _, vid := range d.VolumeIDs {
		node, err := s.members.GetNodeWithVolumeByID(vid)
		if err != nil {
			// Node may be gone; skip it.
			s.logger.Log("msg", err.Error())
			continue
		}
		if err = node.DeleteFile(s.ctx, d.Key); err != nil {
			// "not found" is acceptable — replica was already removed.
			s.logger.Log("msg", err.Error())
		}
	}
	if err = v.DeleteDeletion(s.ctx, d); err != nil {
		s.logger.Log("msg", err.Error())
	}
	return true
}

// loopRemovedVolumeDIs checks if any volumeID
// was removed that it's interesting for the node
// meaning that it has to recover a lost replica
func (s *service) loopRemovedVolumeDIs() {
	for {
		select {
		case <-s.ctx.Done():
			goto end
		default:
			rvids := s.members.RemovedVolumeIDs()
			if len(rvids) == 0 {
				time.Sleep(time.Second)
				break
			}

			for _, vid := range rvids {
				for _, lv := range s.members.LocalVolumes() {
					err := lv.SynchronizeReplicas(s.ctx, vid)
					if err != nil {
						s.logger.Log("msg", err.Error())
						continue
					}
				}
			}
		}
	}
end:
	return
}
