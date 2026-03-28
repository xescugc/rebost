package storing

import (
	"time"

	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/volume"
)

func (s *service) loopRebalance() {
	if s.cfg.Timing.RebalanceInterval <= 0 {
		return
	}
	ticker := time.NewTicker(s.cfg.Timing.RebalanceInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for _, v := range s.members.LocalVolumes() {
				s.rebalanceVolume(v)
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *service) rebalanceVolume(v volume.Local) {
	files, err := v.AllFiles(s.ctx)
	if err != nil {
		s.logger.Error("rebalance: failed to list files", "err", err)
		return
	}
	allVids := s.members.AllVolumeIDs()
	for _, f := range files {
		if len(f.VolumeIDs) == 0 || f.VolumeIDs[0] != v.ID() {
			continue
		}
		if len(f.Keys) == 0 {
			continue
		}
		ranked := rankVolumeIDs(f.Keys[0], allVids)
		if len(ranked) == 0 || ranked[0] == v.ID() {
			continue
		}
		s.transferOwnership(v, f, ranked[0])
	}
}

func (s *service) transferOwnership(v volume.Local, f *file.File, winnerVid string) {
	key := f.Keys[0]
	winner, err := s.members.GetNodeWithVolumeByID(winnerVid)
	if err != nil {
		s.logger.Error("rebalance: winner node not found", "vid", winnerVid, "err", err)
		return
	}

	reader, _, err := v.GetFile(s.ctx, key)
	if err != nil {
		s.logger.Error("rebalance: failed to read local file", "key", key, "err", err)
		return
	}

	if _, err = winner.CreateReplica(s.ctx, key, reader, f.TTL, f.CreatedAt); err != nil {
		s.logger.Error("rebalance: failed to push replica to winner", "key", key, "vid", winnerVid, "err", err)
		return
	}

	// Build updated VolumeIDs: winnerVid at index 0, remaining vids (excluding local ID and winnerVid).
	newVolumeIDs := make([]string, 0, len(f.VolumeIDs))
	newVolumeIDs = append(newVolumeIDs, winnerVid)
	for _, vid := range f.VolumeIDs {
		if vid != v.ID() && vid != winnerVid {
			newVolumeIDs = append(newVolumeIDs, vid)
		}
	}

	// Notify all replica holders of the new VolumeIDs list.
	for _, vid := range newVolumeIDs {
		node, err := s.members.GetNodeWithVolumeByID(vid)
		if err != nil {
			s.logger.Error("rebalance: could not notify node", "vid", vid, "err", err)
			continue
		}
		if err = node.UpdateFileReplica(s.ctx, key, newVolumeIDs, f.Replica); err != nil {
			s.logger.Error("rebalance: failed to update replica info", "key", key, "vid", vid, "err", err)
		}
	}

	// Remove local copy (local-only delete, no deletion queue for remote copies).
	if err := v.DeleteFile(s.ctx, key); err != nil {
		s.logger.Error("rebalance: failed to delete local copy", "key", key, "err", err)
	}
}
