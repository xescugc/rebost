package otelwrap

import (
	"context"
	"time"

	"github.com/xescugc/rebost/file"
)

type instrFileRepo struct {
	inner file.Repository
	inst  *instruments
}

func (r *instrFileRepo) CreateOrReplace(ctx context.Context, f *file.File) (err error) {
	defer func(s time.Time) { r.inst.record(ctx, "files", "create_or_replace", s, err) }(time.Now())
	return r.inner.CreateOrReplace(ctx, f)
}

func (r *instrFileRepo) FindBySignature(ctx context.Context, sig string) (_ *file.File, err error) {
	defer func(s time.Time) { r.inst.record(ctx, "files", "find_by_signature", s, err) }(time.Now())
	return r.inner.FindBySignature(ctx, sig)
}

func (r *instrFileRepo) DeleteBySignature(ctx context.Context, sig string) (err error) {
	defer func(s time.Time) { r.inst.record(ctx, "files", "delete_by_signature", s, err) }(time.Now())
	return r.inner.DeleteBySignature(ctx, sig)
}

func (r *instrFileRepo) DeleteAll(ctx context.Context) (err error) {
	defer func(s time.Time) { r.inst.record(ctx, "files", "delete_all", s, err) }(time.Now())
	return r.inner.DeleteAll(ctx)
}

func (r *instrFileRepo) All(ctx context.Context) (_ []*file.File, err error) {
	defer func(s time.Time) { r.inst.record(ctx, "files", "all", s, err) }(time.Now())
	return r.inner.All(ctx)
}
