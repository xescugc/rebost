package storing

import (
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
)

// loopVolumesReplicas checks if any of the local
// volumes has any replicas, if they do then
// it checks if any of the nodes wants to replicate
func (s *service) loopVolumesReplicas() {
	for {
		var noReplica bool
		select {
		case <-s.ctx.Done():
			goto end
		default:
			for _, v := range s.members.LocalVolumes() {
				rp, err := v.NextReplica(s.ctx)
				if err != nil {
					noReplica = true
					log.Println(s.cfg.MemberlistName, err)
					continue
				}
				spew.Dump(rp)
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
					vID, err := n.CreateReplica(s.ctx, rp.Key, iorc, v.ID(), rp.OriginalCount)
					if err != nil {
						log.Println(err)
						continue
					}

					err = v.UpdateReplica(s.ctx, rp, vID)
					if err != nil {
						log.Println(err)
						continue
					}
				}
			}
		}
		// If nothing was replicated on one run sleep
		// to give a delay
		if noReplica {
			time.Sleep(time.Second)
		}
	}
end:
	return
}
