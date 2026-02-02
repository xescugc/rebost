package storing

import (
	"time"
)

// loopVolumesReplicas checks if any of the local
// volumes has any replicas, if they do then
// it checks if any of the nodes wants to replicate
func (s *service) loopVolumesReplicas() {
	for {
		var replicated bool
		select {
		case <-s.ctx.Done():
			goto end
		default:
			for _, v := range s.members.LocalVolumes() {
				rp, err := v.NextReplica(s.ctx)
				if err != nil {
					if err.Error() != "not found" {
						s.logger.Log("msg", err.Error())
					}
					continue
				}
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
					replicated = true
					break
				}
			}
		}
		// If nothing was replicated on one run sleep
		// to give a delay and not be constantly
		// asking for items to the volumes
		if !replicated {
			time.Sleep(time.Second)
		}
	}
end:
	return
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
