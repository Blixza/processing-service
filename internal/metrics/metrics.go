package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	JobsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "worker_jobs_processed_total",
		Help: "The total number of processed jobs",
	}, []string{"status", "type"})

	ProcessingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "worker_job_duration_seconds",
		Help:    "Time spent processing a job",
		Buckets: prometheus.DefBuckets,
	})
)
