package storing

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

func (s *service) registerMetrics() {
	meter := otel.Meter("rebost")

	usedGauge, err := meter.Int64ObservableGauge(
		"rebost.volume.storage.used_bytes",
		otelmetric.WithDescription("Bytes used on each local volume"),
		otelmetric.WithUnit("By"),
	)
	if err != nil {
		s.logger.Error("failed to create rebost.volume.storage.used_bytes gauge", "err", err)
		return
	}

	totalGauge, err := meter.Int64ObservableGauge(
		"rebost.volume.storage.total_bytes",
		otelmetric.WithDescription("Total capacity of each local volume"),
		otelmetric.WithUnit("By"),
	)
	if err != nil {
		s.logger.Error("failed to create rebost.volume.storage.total_bytes gauge", "err", err)
		return
	}

	filesGauge, err := meter.Int64ObservableGauge(
		"rebost.volume.files",
		otelmetric.WithDescription("Number of file records stored on each local volume"),
	)
	if err != nil {
		s.logger.Error("failed to create rebost.volume.files gauge", "err", err)
		return
	}

	replicaQueueGauge, err := meter.Int64ObservableGauge(
		"rebost.volume.replica_queue_depth",
		otelmetric.WithDescription("Number of pending replica jobs on each local volume"),
	)
	if err != nil {
		s.logger.Error("failed to create rebost.volume.replica_queue_depth gauge", "err", err)
		return
	}

	deletionQueueGauge, err := meter.Int64ObservableGauge(
		"rebost.volume.deletion_queue_depth",
		otelmetric.WithDescription("Number of pending deletion jobs on each local volume"),
	)
	if err != nil {
		s.logger.Error("failed to create rebost.volume.deletion_queue_depth gauge", "err", err)
		return
	}

	replicationLagGauge, err := meter.Float64ObservableGauge(
		"rebost.volume.replication_lag_seconds",
		otelmetric.WithDescription("Age of the oldest pending replica job in seconds; 0 if queue is empty"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		s.logger.Error("failed to create rebost.volume.replication_lag_seconds gauge", "err", err)
		return
	}

	_, err = meter.RegisterCallback(func(ctx context.Context, o otelmetric.Observer) error {
		for _, v := range s.members.LocalVolumes() {
			attrs := otelmetric.WithAttributes(attribute.String("volume_id", v.ID()))
			if st, err := v.GetState(ctx); err == nil {
				o.ObserveInt64(usedGauge, int64(st.UsedSize()), attrs)
				o.ObserveInt64(totalGauge, int64(st.TotalSize()), attrs)
			}
			if files, err := v.AllFiles(ctx); err == nil {
				o.ObserveInt64(filesGauge, int64(len(files)), attrs)
			}
			if n, err := v.CountReplicas(ctx); err == nil {
				o.ObserveInt64(replicaQueueGauge, n, attrs)
			}
			if n, err := v.CountDeletions(ctx); err == nil {
				o.ObserveInt64(deletionQueueGauge, n, attrs)
			}
			if r, ok, err := v.OldestReplica(ctx); err == nil {
				lag := float64(0)
				if ok {
					lag = time.Since(r.EnqueuedAt).Seconds()
				}
				o.ObserveFloat64(replicationLagGauge, lag, attrs)
			}
		}
		return nil
	}, usedGauge, totalGauge, filesGauge, replicaQueueGauge, deletionQueueGauge, replicationLagGauge)
	if err != nil {
		s.logger.Error("failed to register metrics callback", "err", err)
	}
}
