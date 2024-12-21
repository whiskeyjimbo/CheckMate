package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

type PrometheusMetrics struct {
	logger                *zap.SugaredLogger
	checkStatusGauge      *prometheus.GaugeVec
	checkLatencyGauge     *prometheus.GaugeVec
	checkLatencyHistogram *prometheus.HistogramVec
}

func NewPrometheusMetrics(logger *zap.SugaredLogger) *PrometheusMetrics {
	// this probably wont be super accurate based off of polling, maybe i should switch to a counter
	statusGauge := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "check_success",
			Help: "Status of the check (1 for success, 0 for failure)",
		},
		[]string{"host", "port", "protocol"},
	)
	latencyGauge := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "check_latency_milliseconds",
			Help: "Gauge of the check duration in milliseconds",
		},
		[]string{"host", "port", "protocol"},
	)
	latencyHistogram := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "check_latency_milliseconds_histogram",
			Help:    "Histogram of the check duration in milliseconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"host", "port", "protocol"},
	)

	return &PrometheusMetrics{
		logger:                logger,
		checkStatusGauge:      statusGauge,
		checkLatencyGauge:     latencyGauge,
		checkLatencyHistogram: latencyHistogram,
	}
}

func (p *PrometheusMetrics) Update(host string, port string, protocol string, success bool, elapsed time.Duration) {
	statusValue := 0.0
	if success {
		statusValue = 1.0
	}
	p.checkStatusGauge.WithLabelValues(host, port, protocol).Set(statusValue)

	latencyMs := float64(elapsed.Milliseconds())
	p.checkLatencyGauge.WithLabelValues(host, port, protocol).Set(latencyMs)
	p.checkLatencyHistogram.WithLabelValues(host, port, protocol).Observe(latencyMs)
}
