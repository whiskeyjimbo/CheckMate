package metrics

import (
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewPrometheusMetrics(t *testing.T) {
	logger := zap.NewNop().Sugar()
	metrics := NewPrometheusMetrics(logger)

	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.checkStatusGauge)
	assert.NotNil(t, metrics.checkLatencyGauge)
	assert.NotNil(t, metrics.checkLatencyHistogram)
}

func TestPrometheusMetrics_Update(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     string
		protocol string
		success  bool
		elapsed  time.Duration
		want     float64
	}{
		{
			name:     "successful check",
			host:     "example.com",
			port:     "80",
			protocol: "tcp",
			success:  true,
			elapsed:  100 * time.Millisecond,
			want:     1.0,
		},
		{
			name:     "failed check",
			host:     "example.com",
			port:     "443",
			protocol: "tcp",
			success:  false,
			elapsed:  50 * time.Millisecond,
			want:     0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prometheus.DefaultRegisterer = prometheus.NewRegistry()

			logger := zap.NewNop().Sugar()
			metrics := NewPrometheusMetrics(logger)

			metrics.Update(tt.host, tt.port, tt.protocol, tt.success, tt.elapsed)

			statusGauge := metrics.checkStatusGauge.WithLabelValues(tt.host, tt.port, tt.protocol)
			assert.Equal(t, tt.want, testutil.ToFloat64(statusGauge))

			latencyGauge := metrics.checkLatencyGauge.WithLabelValues(tt.host, tt.port, tt.protocol)
			assert.Equal(t, float64(tt.elapsed.Milliseconds()), testutil.ToFloat64(latencyGauge))

			histogram, err := metrics.checkLatencyHistogram.GetMetricWithLabelValues(tt.host, tt.port, tt.protocol)
			assert.NoError(t, err)
			assert.NotNil(t, histogram)
		})
	}
}

func TestStartMetricsServer(t *testing.T) {
	logger := zap.NewNop().Sugar()

	StartMetricsServer(logger)

	resp, err := http.Get("http://localhost:9100/metrics")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}
