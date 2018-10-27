package volume

import (
	"log"
	"time"

	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/uow"
)

func (l *local) loopFromReplicaPendent() {
	for {
		select {
		case <-l.ctx.Done():
			break
		default:
			l.popFromReplicaPendent()
		}
	}
}

func (l *local) popFromReplicaPendent() {
	var (
		err error
		rp  *replica.Pendent
	)
	err = l.startUnitOfWork(l.ctx, uow.Write, func(uw uow.UnitOfWork) error {
		// The convination logic of First && Delete is
		// equal to a Pop in a list
		rp, err = uw.ReplicaPendent().First(l.ctx)
		if err != nil {
			return err
		}
		// There is nothing on the DB
		// so we just return
		if rp == nil {
			return nil
		}

		err = uw.ReplicaPendent().Delete(l.ctx, rp)
		if err != nil {
			return err
		}

		rp.Replica -= 1
		if rp.Replica > 0 {
			err = uw.ReplicaPendent().Create(l.ctx, rp)
			if err != nil {
				return err
			}
		}
		return nil
	}, l.replicaPendent)

	if err != nil && err.Error() != "not found" {
		// TODO: What to do?
		log.Println(err)
	} else if rp == nil {
		time.Sleep(time.Second)
	} else {
		l.replicaPendentChan <- *rp
	}
}
