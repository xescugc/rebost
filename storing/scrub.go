package storing

import (
	"github.com/xescugc/rebost/volume"
)

// processNextScrub picks up one pending scrub-repair job for v and attempts to
// restore the corrupt file from a known-good remote replica.
// Returns true if a job was dequeued (regardless of repair outcome).
func (s *service) processNextScrub(v volume.Local) bool {
	sc, err := v.NextScrub(s.ctx)
	if err != nil {
		if err.Error() != "not found" {
			s.logger.Error(err.Error())
		}
		return false
	}

	if len(sc.VolumeIDs) == 0 {
		s.logger.Error("scrub: corrupt file has no remote replicas, cannot repair", "key", sc.Key)
		if err := v.DeleteScrub(s.ctx, sc); err != nil {
			s.logger.Error(err.Error())
		}
		return true
	}

	repaired := false
	for _, vid := range sc.VolumeIDs {
		node, err := s.members.GetNodeWithVolumeByID(vid)
		if err != nil {
			s.logger.Error(err.Error())
			continue
		}

		_, ok, err := node.HasFile(s.ctx, sc.Key)
		if err != nil || !ok {
			continue
		}

		reader, _, err := node.GetFile(s.ctx, sc.Key)
		if err != nil {
			s.logger.Error(err.Error())
			continue
		}

		if err := v.OverwriteFileContent(s.ctx, sc.Signature, reader); err != nil {
			s.logger.Error("scrub: repair write failed", "key", sc.Key, "err", err)
			reader.Close()
			continue
		}
		reader.Close()
		s.logger.Info("scrub: repaired corrupt file", "key", sc.Key)
		repaired = true
		break
	}

	if !repaired {
		s.logger.Error("scrub: could not repair corrupt file from any replica", "key", sc.Key)
	}

	if err := v.DeleteScrub(s.ctx, sc); err != nil {
		s.logger.Error(err.Error())
	}
	return true
}
