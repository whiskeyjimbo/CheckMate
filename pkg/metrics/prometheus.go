package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type PrometheusMetrics struct {
	checkSuccess  *prometheus.GaugeVec
	checkDuration *prometheus.HistogramVec
}

func NewPrometheusMetrics(sugar *zap.SugaredLogger) *PrometheusMetrics {
	// this probably wont be super accurate based off of polling, maybe i should switch to a counter
	checkSuccess := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "port_check_success",
			Help: "Indicates if the port check was successful (1 for success, 0 for failure)",
		},
		[]string{"host", "port", "protocol"},
	)
	checkDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "port_check_duration_seconds",
			Help:    "Histogram of the port check duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"host", "port", "protocol"},
	)
	prometheus.MustRegister(checkSuccess, checkDuration)

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		sugar.Info("Starting Prometheus metrics server on :9100")
		err := http.ListenAndServe(":9100", nil)
		if err != nil {
			sugar.Fatalf("Error starting Prometheus metrics server: %v", err)
		}
	}()

	return &PrometheusMetrics{
		checkSuccess:  checkSuccess,
		checkDuration: checkDuration,
	}
}

func (p *PrometheusMetrics) Update(host, port, protocol string, success bool, elapsed int64) {
	if success {
		p.checkSuccess.WithLabelValues(host, port, protocol).Set(1)
	} else {
		p.checkSuccess.WithLabelValues(host, port, protocol).Set(0)
	}
	p.checkDuration.WithLabelValues(host, port, protocol).Observe(float64(elapsed) / 1e6)
}
