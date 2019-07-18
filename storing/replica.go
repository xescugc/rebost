package storing

import (
	"fmt"

	"github.com/xescugc/rebost/replica"
)

// loopVolumesReplicaPendent checks if any of the local
// volumes has any replicas pendent, if they do then
// it checks if any of the nodes wants to replicate
// it after that pushes it to the Retry
func (s *service) loopVolumesReplicaPendent() {
	for {
		for _, v := range s.members.LocalVolumes() {
			select {
			case rp, closed := <-v.ReplicasPendent():
				if !closed {
					for _, n := range s.members.Nodes() {
						err := n.CreateReplicaPendent(s.ctx, rp)
						if err != nil {
							// TODO: logs
							fmt.Println(err)
						}

						ncfg, err := n.Config(s.ctx)
						if err != nil {
							// TODO: logs
							fmt.Println(err)
						}

						err = v.CreateReplicaRetry(s.ctx, replica.NewRetryFromPendent(rp, ncfg.MemberlistName))
						if err != nil {
							// TODO: logs
							fmt.Println(err)
							continue
						}
						// Once one of the nodes accepts the
						// ReplicaPendent we can exit
						break
					}
				}
			default:
				continue
			}
		}
	}
}
