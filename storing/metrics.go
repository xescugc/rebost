package storing

import (
	"context"

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
		}
		return nil
	}, usedGauge, totalGauge, filesGauge)
	if err != nil {
		s.logger.Error("failed to register metrics callback", "err", err)
	}
}
