package otelwrap

import (
	"context"
	"time"

	"github.com/xescugc/rebost/replica"
)

type instrReplicaRepo struct {
	inner replica.Repository
	inst  *instruments
}

func (r *instrReplicaRepo) Create(ctx context.Context, rp *replica.Replica) (err error) {
	defer func(s time.Time) { r.inst.record(ctx, "replica", "create", s, err) }(time.Now())
	return r.inner.Create(ctx, rp)
}

func (r *instrReplicaRepo) First(ctx context.Context) (_ *replica.Replica, err error) {
	defer func(s time.Time) { r.inst.record(ctx, "replica", "first", s, err) }(time.Now())
	return r.inner.First(ctx)
}

func (r *instrReplicaRepo) Delete(ctx context.Context, rp *replica.Replica) (err error) {
	defer func(s time.Time) { r.inst.record(ctx, "replica", "delete", s, err) }(time.Now())
	return r.inner.Delete(ctx, rp)
}

func (r *instrReplicaRepo) DeleteAll(ctx context.Context) (err error) {
	defer func(s time.Time) { r.inst.record(ctx, "replica", "delete_all", s, err) }(time.Now())
	return r.inner.DeleteAll(ctx)
}

func (r *instrReplicaRepo) HasAny(ctx context.Context) (_ bool, err error) {
	defer func(s time.Time) { r.inst.record(ctx, "replica", "has_any", s, err) }(time.Now())
	return r.inner.HasAny(ctx)
}

func (r *instrReplicaRepo) HasKey(ctx context.Context, key string) (_ bool, err error) {
	defer func(s time.Time) { r.inst.record(ctx, "replica", "has_key", s, err) }(time.Now())
	return r.inner.HasKey(ctx, key)
}
