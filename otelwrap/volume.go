package otelwrap

import (
	"context"
	"io"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/volume"
)

type tracedLocalVolume struct {
	volume.Local
}

// LocalVolumeWithTracing wraps inner with OpenTelemetry span instrumentation.
func LocalVolumeWithTracing(inner volume.Local) volume.Local {
	return &tracedLocalVolume{Local: inner}
}

func (t *tracedLocalVolume) CreateFile(ctx context.Context, key string, reader io.ReadCloser, replica int, ttl time.Duration, ca time.Time) (err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "volume.CreateFile",
		trace.WithAttributes(
			attribute.String("key", key),
			attribute.String("volume_id", t.Local.ID()),
		))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return t.Local.CreateFile(ctx, key, reader, replica, ttl, ca)
}

func (t *tracedLocalVolume) GetFile(ctx context.Context, key string) (_ io.ReadCloser, _ int64, err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "volume.GetFile",
		trace.WithAttributes(
			attribute.String("key", key),
			attribute.String("volume_id", t.Local.ID()),
		))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return t.Local.GetFile(ctx, key)
}

func (t *tracedLocalVolume) StatFile(ctx context.Context, key string) (_ *volume.FileStat, err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "volume.StatFile",
		trace.WithAttributes(
			attribute.String("key", key),
			attribute.String("volume_id", t.Local.ID()),
		))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return t.Local.StatFile(ctx, key)
}

func (t *tracedLocalVolume) HasFile(ctx context.Context, key string) (_ string, _ bool, err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "volume.HasFile",
		trace.WithAttributes(
			attribute.String("key", key),
			attribute.String("volume_id", t.Local.ID()),
		))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return t.Local.HasFile(ctx, key)
}

func (t *tracedLocalVolume) DeleteFile(ctx context.Context, key string) (err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "volume.DeleteFile",
		trace.WithAttributes(
			attribute.String("key", key),
			attribute.String("volume_id", t.Local.ID()),
		))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return t.Local.DeleteFile(ctx, key)
}

func (t *tracedLocalVolume) SynchronizeReplicas(ctx context.Context, vID string) (err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "volume.SynchronizeReplicas",
		trace.WithAttributes(
			attribute.String("volume_id", t.Local.ID()),
			attribute.String("removed_volume_id", vID),
		))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return t.Local.SynchronizeReplicas(ctx, vID)
}

func (t *tracedLocalVolume) NextReplica(ctx context.Context) (_ *replica.Replica, err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "volume.NextReplica",
		trace.WithAttributes(attribute.String("volume_id", t.Local.ID())))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return t.Local.NextReplica(ctx)
}

func (t *tracedLocalVolume) UpdateReplica(ctx context.Context, rp *replica.Replica, vID string) (err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "volume.UpdateReplica",
		trace.WithAttributes(attribute.String("volume_id", t.Local.ID())))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return t.Local.UpdateReplica(ctx, rp, vID)
}
