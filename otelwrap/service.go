package otelwrap

import (
	"context"
	"io"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/storing"
	"github.com/xescugc/rebost/volume"
)

type tracedService struct {
	storing.Service
}

// ServiceWithTracing wraps inner with OpenTelemetry span instrumentation.
func ServiceWithTracing(inner storing.Service) storing.Service {
	return &tracedService{Service: inner}
}

func (s *tracedService) CreateFile(ctx context.Context, k string, r io.ReadCloser, rep int, ttl time.Duration, ca time.Time) (err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "storing.CreateFile",
		trace.WithAttributes(attribute.String("key", k), attribute.Int("replica", rep)))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return s.Service.CreateFile(ctx, k, r, rep, ttl, ca)
}

func (s *tracedService) GetFile(ctx context.Context, k string) (_ io.ReadCloser, _ int64, err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "storing.GetFile",
		trace.WithAttributes(attribute.String("key", k)))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return s.Service.GetFile(ctx, k)
}

func (s *tracedService) StatFile(ctx context.Context, k string) (_ *volume.FileStat, err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "storing.StatFile",
		trace.WithAttributes(attribute.String("key", k)))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return s.Service.StatFile(ctx, k)
}

func (s *tracedService) HasFile(ctx context.Context, k string) (_ string, _ bool, err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "storing.HasFile",
		trace.WithAttributes(attribute.String("key", k)))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return s.Service.HasFile(ctx, k)
}

func (s *tracedService) DeleteFile(ctx context.Context, k string) (err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "storing.DeleteFile",
		trace.WithAttributes(attribute.String("key", k)))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return s.Service.DeleteFile(ctx, k)
}

func (s *tracedService) UpdateFileReplica(ctx context.Context, k string, volumeIDs []string, replica int) (err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "storing.UpdateFileReplica",
		trace.WithAttributes(attribute.String("key", k)))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return s.Service.UpdateFileReplica(ctx, k, volumeIDs, replica)
}

func (s *tracedService) CreateReplica(ctx context.Context, k string, reader io.ReadCloser, ttl time.Duration, ca time.Time) (_ string, err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "storing.CreateReplica",
		trace.WithAttributes(attribute.String("key", k)))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return s.Service.CreateReplica(ctx, k, reader, ttl, ca)
}

func (s *tracedService) GetReplicaInfo(ctx context.Context, k string) (_ []string, _ int, err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "storing.GetReplicaInfo",
		trace.WithAttributes(attribute.String("key", k)))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return s.Service.GetReplicaInfo(ctx, k)
}

func (s *tracedService) Drain(ctx context.Context) (err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "storing.Drain")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return s.Service.Drain(ctx)
}

func (s *tracedService) Ready(ctx context.Context) (err error) {
	ctx, span := otel.Tracer("rebost").Start(ctx, "storing.Ready")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	return s.Service.Ready(ctx)
}

// Config delegates directly — metadata only, no span needed.
func (s *tracedService) Config(ctx context.Context) (*config.Config, error) {
	return s.Service.Config(ctx)
}
