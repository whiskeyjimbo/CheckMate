package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const (
	metricsPort = ":9100"
	namespace   = "checkmate"
)

type MetricLabels struct {
	Host     string
	Port     string
	Protocol string
}

type PrometheusMetrics struct {
	logger                *zap.SugaredLogger
	checkStatusGauge      *prometheus.GaugeVec
	checkLatencyGauge     *prometheus.GaugeVec
	checkLatencyHistogram *prometheus.HistogramVec
}

func NewPrometheusMetrics(logger *zap.SugaredLogger) *PrometheusMetrics {
	return &PrometheusMetrics{
		logger:                logger,
		checkStatusGauge:      createStatusGauge(),
		checkLatencyGauge:     createLatencyGauge(),
		checkLatencyHistogram: createLatencyHistogram(),
	}
}

func StartMetricsServer(logger *zap.SugaredLogger) {
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		if err := http.ListenAndServe(metricsPort, nil); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start Prometheus metrics server: %v", err)
		}
	}()
}

func (p *PrometheusMetrics) Update(host, port, protocol string, success bool, elapsed time.Duration) {
	labels := MetricLabels{
		Host:     host,
		Port:     port,
		Protocol: protocol,
	}
	p.updateMetrics(labels, success, elapsed)
}

func (p *PrometheusMetrics) updateMetrics(labels MetricLabels, success bool, elapsed time.Duration) {
	labelValues := []string{labels.Host, labels.Port, labels.Protocol}

	statusValue := 0.0
	if success {
		statusValue = 1.0
	}

	latencyMs := float64(elapsed.Milliseconds())

	p.checkStatusGauge.WithLabelValues(labelValues...).Set(statusValue)
	p.checkLatencyGauge.WithLabelValues(labelValues...).Set(latencyMs)
	p.checkLatencyHistogram.WithLabelValues(labelValues...).Observe(latencyMs)
}

func createStatusGauge() *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "check_success",
			Help:      "Status of the check (1 for success, 0 for failure)",
		},
		[]string{"host", "port", "protocol"},
	)
}

func createLatencyGauge() *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "check_latency_milliseconds",
			Help:      "Gauge of the check duration in milliseconds",
		},
		[]string{"host", "port", "protocol"},
	)
}

func createLatencyHistogram() *prometheus.HistogramVec {
	return promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "check_latency_milliseconds_histogram",
			Help:      "Histogram of the check duration in milliseconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"host", "port", "protocol"},
	)
}
