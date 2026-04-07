package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Registry struct {
	JobsProcessed      *prometheus.CounterVec
	ProcessingDuration prometheus.Histogram
}

func NewRegistry() *Registry {
	return &Registry{
		JobsProcessed: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "worker_jobs_processed_total",
			Help: "The total number of processed jobs",
		}, []string{"status", "type"}),

		ProcessingDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "processing_duration_seconds",
			Help: "Time spent processing an image",
		}),
	}
}
