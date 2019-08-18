package storing

import (
	"log"
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
					log.Println(err)
					continue
				}
				for _, n := range s.members.Nodes() {
					ok, err := n.HasFile(s.ctx, rp.Key)
					if err != nil {
						log.Println(err)
						continue
					}
					//If the volume already has this key we ignore it
					//It means the replica is already on that Node
					//Or that it has a file with that name
					if ok {
						continue
					}
					iorc, err := v.GetFile(s.ctx, rp.Key)
					if err != nil {
						log.Println(err)
						continue
					}
					vID, err := n.CreateReplica(s.ctx, rp.Key, iorc)
					if err != nil {
						log.Println(err)
						continue
					}

					rp.VolumeIDs = append(rp.VolumeIDs, vID)

					for _, vid := range rp.VolumeIDs {
						n, err := s.members.GetNodeWithVolumeByID(vid)
						if err != nil {
							log.Println(err)
							continue
						}
						err = n.UpdateFileReplica(s.ctx, rp.Key, append([]string{v.ID()}, rp.VolumeIDs...), rp.OriginalCount)
						if err != nil {
							log.Println(err)
							continue
						}
					}

					err = v.UpdateReplica(s.ctx, rp, vID)
					if err != nil {
						log.Println(err)
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
