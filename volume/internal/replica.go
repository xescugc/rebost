package internal

import (
	"context"

	uuid "github.com/satori/go.uuid"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/uow"
)

type Replica struct {
	replicaPendent replica.PendentRepository
	replicaRetry   replica.RetryRepository

	startUnitOfWork uow.StartUnitOfWork

	replicasPendentChan chan replica.Pendent
	replicasRetryChan   chan replica.Retry
}

func NewReplica(rp replica.PendentRepository, rr replica.RetryRepository, suow uow.StartUnitOfWork) *Replica {
	return &Replica{
		startUnitOfWork: suow,

		replicasPendentChan: make(chan replica.Pendent),
		replicaPendent:      rp,

		replicasRetryChan: make(chan replica.Retry),
		replicaRetry:      rr,
	}
}

func (r *Replica) CreateReplicaPendent(ctx context.Context, vID, key, sig string, rep int) error {
	rp := &replica.Pendent{
		ID:  uuid.NewV4().String(),
		Key: key,
		// TODO: For now we are ignoring the fact
		// that if the file exists the replicas may
		// chage and be more or lesss
		Replica:   rep,
		Signature: sig,
		VolumeID:  vID,
	}

	err := r.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		err := uw.ReplicaPendent().Create(ctx, rp)
		if err != nil {
			return err
		}

		return nil
	}, r.replicaPendent)

	if err != nil {
		return err
	}

	return nil
}

func (r *Replica) CreateReplicaRetry(ctx context.Context, rr *replica.Retry) error {
	err := r.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		err := uw.ReplicaRetry().Create(ctx, rr)
		if err != nil {
			return err
		}

		return nil
	}, r.replicaRetry)

	if err != nil {
		return err
	}

	return nil
}

func (r *Replica) PopFromReplicaPendent(ctx context.Context) error {
	var (
		err error
		rp  *replica.Pendent
	)
	err = r.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		// The convination logic of First && Delete is
		// equal to a Pop in a list
		rp, err = uw.ReplicaPendent().First(ctx)
		if err != nil {
			return err
		}
		// There is nothing on the DB
		// so we just return
		if rp == nil {
			return nil
		}

		err = uw.ReplicaPendent().Delete(ctx, rp)
		if err != nil {
			return err
		}

		rp.Replica -= 1
		if rp.Replica > 0 {
			err = r.CreateReplicaPendent(ctx, rp.VolumeID, rp.Key, rp.Signature, rp.Replica)
			if err != nil {
				return err
			}
		}
		return nil
	}, r.replicaPendent)

	if err != nil {
		if err.Error() == "not found" {
			return nil
		} else {
			return err
		}
	} else {
		r.replicasPendentChan <- *rp
	}

	return nil
}

func (r *Replica) PopFromReplicaRetry(ctx context.Context) error {
	var (
		err error
		rr  *replica.Retry
	)
	err = r.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		// The convination logic of First && Delete is
		// equal to a Pop in a list
		rr, err = uw.ReplicaRetry().First(ctx)
		if err != nil {
			return err
		}
		// There is nothing on the DB
		// so we just return
		if rr == nil {
			return nil
		}

		err = uw.ReplicaRetry().Delete(ctx, rr)
		if err != nil {
			return err
		}

		return nil
	}, r.replicaRetry)

	if err != nil {
		if err.Error() == "not found" {
			return nil
		} else {
			return err
		}
	} else {
		r.replicasRetryChan <- *rr
	}

	return nil
}

func (r *Replica) ReplicasPendent() <-chan replica.Pendent {
	return r.replicasPendentChan
}

func (r *Replica) ReplicasRetry() <-chan replica.Retry {
	return r.replicasRetryChan
}
