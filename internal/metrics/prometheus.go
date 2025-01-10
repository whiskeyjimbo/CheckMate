// Copyright (C) 2025 Jeff Rose
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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

type HostResult struct {
	Success      bool
	ResponseTime time.Duration
	Error        error
}

type GroupMetrics struct {
	Site        string
	Group       string
	Port        string
	Protocol    string
	Tags        []string
	HostResults map[string]HostResult
	HostsUp     int
	HostsTotal  int
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
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health/live", health.LivenessHandler)
	mux.HandleFunc("/health/ready", health.ReadinessHandler)

	server := &http.Server{
		Addr:              metricsPort,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start metrics server: %v", err)
		}
	}()
}

func (p *PrometheusMetrics) UpdateGroup(metrics GroupMetrics) {
	for host, result := range metrics.HostResults {
		labels := MetricLabels{
			Site:     metrics.Site,
			Group:    metrics.Group,
			Host:     host,
			Port:     metrics.Port,
			Protocol: metrics.Protocol,
		}
		p.updateMetrics(labels, metrics.Tags, result.Success, result.ResponseTime)
	}
	p.updateGroupCounts(metrics.Site, metrics.Group, metrics.Port, metrics.Protocol, metrics.HostsUp, metrics.HostsTotal)
}

func (p *PrometheusMetrics) updateMetrics(labels MetricLabels, tags []string, success bool, elapsed time.Duration) {
	tagString := normalizeTagString(tags)

	fullLabels := []string{labels.Site, labels.Group, labels.Host, labels.Port, labels.Protocol, tagString}

	histLabels := []string{labels.Site, labels.Group, labels.Port, labels.Protocol, tagString}

	statusValue := 0.0
	if success {
		statusValue = 1.0
	}
	p.checkStatus.WithLabelValues(fullLabels...).Set(statusValue)

	// Update latency metrics only for successful checks
	if success {
		p.checkLatency.WithLabelValues(fullLabels...).Set(float64(elapsed.Milliseconds()))
		p.latencyHist.WithLabelValues(histLabels...).Observe(float64(elapsed.Seconds()))
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
			Name:      "host_check_status",
			Help:      "Status of the host check (1 for up, 0 for down)",
		},
		[]string{"site", "group", "host", "port", "protocol", "tags"},
	)
}

func createCheckLatencyMetric() *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "host_check_latency_milliseconds",
			Help:      "Latency of the host check in milliseconds",
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
