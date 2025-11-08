package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	MatchesTotal  = promauto.NewCounter(prometheus.CounterOpts{Namespace: "ride_matching", Name: "matches_total", Help: "Total number of matches"})
	MatchLatency  = promauto.NewHistogram(prometheus.HistogramOpts{Namespace: "ride_matching", Name: "match_latency_seconds", Help: "Match latency seconds"})
	DriversOnline = promauto.NewGauge(prometheus.GaugeOpts{Namespace: "ride_matching", Name: "drivers_online", Help: "Number of online drivers"})

	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{Namespace: "ride_matching", Name: "http_requests_total", Help: "Total HTTP requests handled"},
		[]string{"method", "path", "status"},
	)
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ride_matching",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency distribution",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
)
