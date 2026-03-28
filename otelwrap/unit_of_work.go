// Package otelwrap wraps the UnitOfWork and its repositories with OpenTelemetry
// instrumentation. It follows the same decorator pattern as fs.UOWWithFs.
package otelwrap

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"

	"github.com/xescugc/rebost/deletion"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/uow"
)

// instruments holds the shared OTEL instruments created once and reused
// across all transactions.
type instruments struct {
	opDuration otelmetric.Float64Histogram
}

func newInstruments(meter otelmetric.Meter) (*instruments, error) {
	h, err := meter.Float64Histogram(
		"rebost.db.operation.duration",
		otelmetric.WithDescription("Duration of BoltDB repository operations"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}
	return &instruments{opDuration: h}, nil
}

func (inst *instruments) record(ctx context.Context, repo, method string, start time.Time, err error) {
	status := "ok"
	if err != nil {
		status = "error"
	}
	inst.opDuration.Record(ctx, time.Since(start).Seconds(),
		otelmetric.WithAttributes(
			attribute.String("repo", repo),
			attribute.String("method", method),
			attribute.String("status", status),
		),
	)
}

// wrappedUoW embeds an inner UnitOfWork and overrides Files, Replicas, and
// Deletions to return instrumented wrappers. All other methods (IDXKeys,
// IDXTTLs, IDXVolumes, Fs, Scrubs, State) delegate automatically via embedding.
type wrappedUoW struct {
	uow.UnitOfWork
	inst *instruments
}

func (w *wrappedUoW) Files() file.Repository {
	return &instrFileRepo{inner: w.UnitOfWork.Files(), inst: w.inst}
}

func (w *wrappedUoW) Replicas() replica.Repository {
	return &instrReplicaRepo{inner: w.UnitOfWork.Replicas(), inst: w.inst}
}

func (w *wrappedUoW) Deletions() deletion.Repository {
	return &instrDeletionRepo{inner: w.UnitOfWork.Deletions(), inst: w.inst}
}

// UOWWithOTEL wraps inner with OTEL instrumentation using the global
// MeterProvider. The global MeterProvider must be configured before calling
// this function.
func UOWWithOTEL(inner uow.StartUnitOfWork) (uow.StartUnitOfWork, error) {
	inst, err := newInstruments(otel.Meter("rebost"))
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, t uow.Type, fn uow.UnitOfWorkFn) error {
		return inner(ctx, t, func(ctx context.Context, uw uow.UnitOfWork) error {
			return fn(ctx, &wrappedUoW{UnitOfWork: uw, inst: inst})
		})
	}, nil
}
