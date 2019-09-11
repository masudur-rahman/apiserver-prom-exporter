package api

import "github.com/prometheus/client_golang/prometheus"

var (
	prom                           *prometheus.Registry
	promVersion                    prometheus.Gauge
	promHttpRequestTotal           *prometheus.CounterVec
	promHttpRequestDurationSeconds *prometheus.HistogramVec
)

func init() {
	prom = prometheus.NewRegistry()

	promVersion = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "apiserver_prom_exporter_version",
		Help: "Version of apiserver-prom-exporter",
		ConstLabels: map[string]string{
			"version": "v0.1.0",
		},
	})

	promHttpRequestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "apiserver_prom_exporter_http_request_total",
		Help: "Count of all http requests",
	}, []string{"url", "method", "code"})

	promHttpRequestDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "apiserver_prom_exporter_http_request_duration_seconds",
		Help:    "HTTP request duration",
		Buckets: []float64{4, 8, 12, 16, 20},
	}, []string{"url", "method"})

	prom.MustRegister(promVersion, promHttpRequestTotal, promHttpRequestDurationSeconds)
}
