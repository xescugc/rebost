package storing

import (
	"time"

	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/logevent"
	"github.com/xescugc/rebost/volume"
)

func (s *service) loopConsistencyCheck() {
	if s.cfg.Timing.ReplicaConsistencyInterval <= 0 {
		return
	}
	tk := time.NewTicker(s.cfg.Timing.ReplicaConsistencyInterval)
	defer tk.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-tk.C:
			s.runConsistencyCheck()
		}
	}
}

func (s *service) runConsistencyCheck() {
	for _, v := range s.members.LocalVolumes() {
		files, err := v.AllFiles(s.ctx)
		if err != nil {
			s.logger.Error("consistency-check: failed to list files", "err", err)
			continue
		}
		for _, f := range files {
			// Owners manage themselves; only non-owners need this check.
			if len(f.VolumeIDs) == 0 || f.VolumeIDs[0] == v.ID() {
				continue
			}
			s.checkFileConsistency(v, f)
		}
	}
}

func (s *service) checkFileConsistency(v volume.Local, f *file.File) {
	key := f.Keys[0]
	// Try VolumeIDs in order (owner first); stop at first successful response.
	for _, vid := range f.VolumeIDs {
		if vid == v.ID() {
			continue // don't ask ourselves
		}
		n, err := s.members.GetNodeWithVolumeByID(vid)
		if err != nil {
			continue // node not in cluster, try next
		}
		vids, _, err := n.CheckReplica(s.ctx, key)
		if err != nil {
			continue // unreachable, try next
		}
		// Got a response — check if this volume is still listed.
		for _, id := range vids {
			if id == v.ID() {
				return // still a valid holder, nothing to do
			}
		}
		// This volume is no longer listed; purge the stale local copy.
		s.logger.Info("consistency-check: purging stale replica", "event", logevent.ReplicaConsistencyPurged, "key", key, "volume", v.ID())
		if err := v.PurgeFile(s.ctx, key); err != nil {
			s.logger.Error("consistency-check: purge failed", "key", key, "err", err)
		}
		return
	}
	// All holders unreachable — hold the file until a leave event arrives.
}
