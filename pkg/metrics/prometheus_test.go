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
		want     float64 // expected status value
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

			// Verify status gauge
			statusGauge := metrics.checkStatusGauge.WithLabelValues(tt.host, tt.port, tt.protocol)
			assert.Equal(t, tt.want, testutil.ToFloat64(statusGauge))

			// Verify latency gauge
			latencyGauge := metrics.checkLatencyGauge.WithLabelValues(tt.host, tt.port, tt.protocol)
			assert.Equal(t, float64(tt.elapsed.Milliseconds()), testutil.ToFloat64(latencyGauge))

			// Verify histogram (we can only verify that it was recorded)
			histogram, err := metrics.checkLatencyHistogram.GetMetricWithLabelValues(tt.host, tt.port, tt.protocol)
			assert.NoError(t, err)
			assert.NotNil(t, histogram)
		})
	}
}

func TestStartMetricsServer(t *testing.T) {
	logger := zap.NewNop().Sugar()
	
	// Start the metrics server
	StartMetricsServer(logger)

	// Make a request to the metrics endpoint to ensure it's working
	resp, err := http.Get("http://localhost:9100/metrics")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
} 