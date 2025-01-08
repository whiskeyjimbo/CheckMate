package metrics

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/whiskeyjimbo/CheckMate/internal/checkers"
	"github.com/whiskeyjimbo/CheckMate/internal/health"
	"go.uber.org/zap"
)

const (
	metricsPort = ":9100"
	namespace   = "checkmate"
)

type MetricLabels struct {
	Site     string
	Group    string
	Host     string
	Port     string
	Protocol string
}

type PrometheusMetrics struct {
	logger *zap.SugaredLogger

	// Check metrics
	checkStatus  *prometheus.GaugeVec
	checkLatency *prometheus.GaugeVec
	latencyHist  *prometheus.HistogramVec

	// Graph metrics
	nodeInfo   *prometheus.GaugeVec
	edgeInfo   *prometheus.GaugeVec
	hostsUp    *prometheus.GaugeVec
	hostsTotal *prometheus.GaugeVec

	// Certificate metrics
	certExpiryDays *prometheus.GaugeVec
	monitorSite    string
}

type GroupMetrics struct {
	Site         string
	Group        string
	Port         string
	Protocol     string
	Tags         []string
	Success      bool
	ResponseTime time.Duration
	HostsUp      int
	HostsTotal   int
}

func NewPrometheusMetrics(logger *zap.SugaredLogger, monitorSite string) *PrometheusMetrics {
	p := &PrometheusMetrics{
		logger:      logger,
		monitorSite: monitorSite,
	}
	p.initMetrics()
	return p
}

func (p *PrometheusMetrics) initMetrics() {
	p.checkStatus = createCheckStatusMetric()
	p.checkLatency = createCheckLatencyMetric()
	p.latencyHist = createLatencyHistogram()
	p.hostsUp, p.hostsTotal = createHostCountMetrics()
	p.nodeInfo = createNodeMetric()
	p.edgeInfo = createEdgeMetric()
	p.certExpiryDays = createCertExpiryMetric()
}

func StartMetricsServer(logger *zap.SugaredLogger) {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health/live", health.LivenessHandler)
	http.HandleFunc("/health/ready", health.ReadinessHandler)

	go func() {
		if err := http.ListenAndServe(metricsPort, nil); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start metrics server: %v", err)
		}
	}()
}

func (p *PrometheusMetrics) UpdateGroup(metrics GroupMetrics) {
	labels := MetricLabels{
		Site:     metrics.Site,
		Group:    metrics.Group,
		Port:     metrics.Port,
		Protocol: metrics.Protocol,
	}
	p.updateMetrics(labels, metrics.Tags, metrics.Success, metrics.ResponseTime)
	p.updateGroupCounts(metrics.Site, metrics.Group, metrics.Port, metrics.Protocol, metrics.HostsUp, metrics.HostsTotal)
}

func (p *PrometheusMetrics) updateMetrics(labels MetricLabels, tags []string, success bool, elapsed time.Duration) {
	tagString := normalizeTagString(tags)
	labelValues := []string{labels.Site, labels.Group, labels.Host, labels.Port, labels.Protocol, tagString}

	statusValue := 0.0
	if success {
		statusValue = 1.0
	}
	p.checkStatus.WithLabelValues(labelValues...).Set(statusValue)

	// Update latency metrics only for successful checks
	if success {
		latencyMs := float64(elapsed.Milliseconds())
		p.checkLatency.WithLabelValues(labelValues...).Set(latencyMs)
		p.latencyHist.WithLabelValues(labelValues...).Observe(latencyMs)
	}

	p.updateGraphMetrics(labels, tagString, success, elapsed)
}

func (p *PrometheusMetrics) updateGraphMetrics(labels MetricLabels, tagString string, success bool, responseTime time.Duration) {
	latencyMs := float64(responseTime.Milliseconds())

	// Monitor site node
	p.nodeInfo.WithLabelValues(p.monitorSite, "site", p.monitorSite, "monitor,internal", "9100", "monitor").Set(1)
	p.hostsUp.WithLabelValues(p.monitorSite, "site", "9100", "monitor").Set(1)
	p.hostsTotal.WithLabelValues(p.monitorSite, "site", "9100", "monitor").Set(1)

	// Site metrics
	if labels.Site != "" && labels.Site != p.monitorSite {
		p.nodeInfo.WithLabelValues(labels.Site, "site", labels.Site, tagString, labels.Port, labels.Protocol).Set(1)
		p.edgeInfo.WithLabelValues(p.monitorSite, labels.Site, "monitors", "latency", labels.Port, labels.Protocol).Set(latencyMs)
	}

	// Group metrics
	if labels.Group != "" {
		nodeID := fmt.Sprintf("%s/%s", labels.Site, labels.Group)
		p.nodeInfo.WithLabelValues(nodeID, "group", labels.Group, tagString, labels.Port, labels.Protocol).Set(1)
		p.edgeInfo.WithLabelValues(labels.Site, nodeID, "contains", "latency", labels.Port, labels.Protocol).Set(latencyMs)
	}

	// Host metrics
	if labels.Host != "" {
		nodeID := fmt.Sprintf("%s/%s/%s", labels.Site, labels.Group, labels.Host)
		p.nodeInfo.WithLabelValues(nodeID, "host", labels.Host, tagString, labels.Port, labels.Protocol).Set(float64(boolToInt(success)))
		groupID := fmt.Sprintf("%s/%s", labels.Site, labels.Group)
		p.edgeInfo.WithLabelValues(groupID, nodeID, "contains", "latency", labels.Port, labels.Protocol).Set(latencyMs)
	}
}

func (p *PrometheusMetrics) updateGroupCounts(site, group, port, protocol string, hostsUp, hostsTotal int) {
	nodeID := fmt.Sprintf("%s/%s", site, group)
	p.hostsUp.WithLabelValues(nodeID, "group", port, protocol).Set(float64(hostsUp))
	p.hostsTotal.WithLabelValues(nodeID, "group", port, protocol).Set(float64(hostsTotal))
}

func createGauge(name, help string) *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      name,
			Help:      help,
		},
		[]string{"site", "group", "host", "port", "protocol", "tags"},
	)
}

func createHistogram(name, help string) *prometheus.HistogramVec {
	return promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      name,
			Help:      help,
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"site", "group", "host", "port", "protocol", "tags"},
	)
}

func createNodeMetric() *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "node_info",
			Help:      "Node information for graph visualization",
		},
		[]string{"id", "type", "name", "tags", "port", "protocol"},
	)
}

func createEdgeMetric() *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "edge_info",
			Help:      "Edge information with latency for graph visualization",
		},
		[]string{"source", "target", "type", "metric", "port", "protocol"},
	)
}

func createHostCountMetrics() (*prometheus.GaugeVec, *prometheus.GaugeVec) {
	hostsUp := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "hosts_up",
			Help:      "Number of hosts up in a group or site",
		},
		[]string{"id", "type", "port", "protocol"},
	)

	hostsTotal := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "hosts_total",
			Help:      "Total number of hosts in a group or site",
		},
		[]string{"id", "type", "port", "protocol"},
	)

	return hostsUp, hostsTotal
}

func normalizeTagString(tags []string) string {
	if len(tags) == 0 {
		return "none"
	}
	return strings.Join(tags, ",")
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (p *PrometheusMetrics) UpdateCertificate(site, group, host, port string, certInfo *checkers.CertInfo) {
	if certInfo == nil {
		return
	}

	daysUntilExpiry := time.Until(certInfo.ExpiresAt).Hours() / 24
	p.certExpiryDays.With(prometheus.Labels{
		"site":   site,
		"group":  group,
		"host":   host,
		"port":   port,
		"issuer": certInfo.IssuedBy,
	}).Set(daysUntilExpiry)
}

func createCheckStatusMetric() *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "check_status",
			Help:      "Status of the check (1 for up, 0 for down)",
		},
		[]string{"site", "group", "host", "port", "protocol", "tags"},
	)
}

func createCheckLatencyMetric() *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "check_latency_seconds",
			Help:      "Latency of the check in seconds",
		},
		[]string{"site", "group", "host", "port", "protocol", "tags"},
	)
}

func createLatencyHistogram() *prometheus.HistogramVec {
	return promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "check_latency_histogram_seconds",
			Help:      "Histogram of check latencies",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"site", "group", "port", "protocol", "tags"},
	)
}

func createCertExpiryMetric() *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "cert_expiry_days",
			Help:      "Days until certificate expiration",
		},
		[]string{"site", "group", "host", "port", "issuer"},
	)
}
