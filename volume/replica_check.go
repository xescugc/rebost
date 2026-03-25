package volume

import (
	"context"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/uow"
)

// loopReplicaCheck ticks every replicaCheckInterval and checks whether all
// locally-owned files have the required number of replicas, enqueuing a new
// replica job for any that are under-replicated and have no job queued yet.
func (l *local) loopReplicaCheck() {
	if l.replicaCheckInterval <= 0 {
		return
	}
	tk := time.NewTicker(l.replicaCheckInterval)
	defer tk.Stop()
	for {
		select {
		case <-l.ctx.Done():
			return
		case <-tk.C:
			l.runReplicaCheck()
		}
	}
}

func (l *local) runReplicaCheck() {
	// Phase 1: snapshot all file records in a single read transaction.
	type fileSnapshot struct {
		key       string
		signature string
		volumeIDs []string
		replica   int
	}
	var candidates []fileSnapshot

	if err := l.startUnitOfWork(l.ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		files, err := uw.Files().All(ctx)
		if err != nil {
			return err
		}
		for _, f := range files {
			if len(f.VolumeIDs) >= f.Replica {
				continue
			}
			if len(f.VolumeIDs) == 0 || f.VolumeIDs[0] != l.id {
				continue
			}
			candidates = append(candidates, fileSnapshot{
				key:       f.Keys[0],
				signature: f.Signature,
				volumeIDs: append([]string(nil), f.VolumeIDs...),
				replica:   f.Replica,
			})
		}
		return nil
	}); err != nil {
		l.logger.Error("replica-check: failed to list files", "err", err)
		return
	}

	// Phase 2: for each candidate, create a replica job if one isn't queued.
	for _, c := range candidates {
		if err := l.startUnitOfWork(l.ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
			exists, err := uw.Replicas().HasKey(ctx, c.key)
			if err != nil {
				return err
			}
			if exists {
				return nil
			}
			rp := &replica.Replica{
				ID:            uuid.NewV4().String(),
				Key:           c.key,
				Count:         c.replica - len(c.volumeIDs),
				OriginalCount: c.replica,
				Signature:     c.signature,
				VolumeID:      l.id,
				VolumeIDs:     c.volumeIDs,
			}
			return uw.Replicas().Create(ctx, rp)
		}); err != nil {
			l.logger.Error("replica-check: failed to enqueue replica job", "key", c.key, "err", err)
		}
	}
}
