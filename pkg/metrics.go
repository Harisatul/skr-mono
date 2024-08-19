package pkg

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	RequestCounter prometheus.Counter
	ResponseTime   prometheus.Histogram
}

func NewMetrics(namespace string) *Metrics {
	return &Metrics{
		RequestCounter: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "requests_total",
			Help:      "The total number of requests",
		}),
		ResponseTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "response_time_seconds",
			Help:      "Response time in seconds",
			Buckets:   prometheus.DefBuckets,
		}),
	}
}
