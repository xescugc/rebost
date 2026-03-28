package otelwrap

import (
	"context"
	"time"

	"github.com/xescugc/rebost/deletion"
)

type instrDeletionRepo struct {
	inner deletion.Repository
	inst  *instruments
}

func (r *instrDeletionRepo) Create(ctx context.Context, d *deletion.Deletion) (err error) {
	defer func(s time.Time) { r.inst.record(ctx, "deletion", "create", s, err) }(time.Now())
	return r.inner.Create(ctx, d)
}

func (r *instrDeletionRepo) First(ctx context.Context) (_ *deletion.Deletion, err error) {
	defer func(s time.Time) { r.inst.record(ctx, "deletion", "first", s, err) }(time.Now())
	return r.inner.First(ctx)
}

func (r *instrDeletionRepo) Delete(ctx context.Context, d *deletion.Deletion) (err error) {
	defer func(s time.Time) { r.inst.record(ctx, "deletion", "delete", s, err) }(time.Now())
	return r.inner.Delete(ctx, d)
}
