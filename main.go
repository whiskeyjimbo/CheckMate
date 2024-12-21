package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/whiskeyjimbo/CheckMate/pkg/checkers"
	"github.com/whiskeyjimbo/CheckMate/pkg/config"
	"github.com/whiskeyjimbo/CheckMate/pkg/metrics"
	"go.uber.org/zap"
)

func main() {
	zapL, _ := zap.NewProduction()
	defer zapL.Sync()
	logger := zapL.Sugar()

	configFile := "config.yaml"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	config, err := config.LoadConfig(configFile)
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	promMetricsEndpoint := metrics.NewPrometheusMetrics(logger)

	for _, hostConfig := range config.Hosts {
		go monitorHost(logger, hostConfig, promMetricsEndpoint)
	}

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(":9100", nil); err != nil {
			logger.Fatalf("Failed to start Prometheus metrics server: %v", err)
		}
	}()

	chanLength := len(config.Hosts) - 1
	c := make(chan os.Signal, chanLength)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	logger.Info("Received shutdown signal, exiting...")
}

func monitorHost(logger *zap.SugaredLogger, hostConfig config.HostConfig, promMetricsEndpoint *metrics.PrometheusMetrics) {
	interval, err := time.ParseDuration(hostConfig.Interval)
	if err != nil {
		logger.Fatalf("Invalid interval %s: %v", hostConfig.Interval, err)
	}

	address := fmt.Sprintf("%s:%s", hostConfig.Host, hostConfig.Port)

	checker, err := checkers.NewChecker(hostConfig.Protocol)
	if err != nil {
		logger.Fatalf("Unsupported protocol %s", hostConfig.Protocol)
	}

	for {
		success, elapsed, err := checker.Check(address)
		l := logger.
			With("host", hostConfig.Host).
			With("port", hostConfig.Port).
			With("protocol", hostConfig.Protocol).
			With("responseTime_us", elapsed)
		if err != nil {
			l.With("success", false).Warnf("Check failed: %v", err)
		} else if success {
			l.With("success", true).Infof("Check succeeded")
		} else {
			l.With("success", false).Errorf("Check failed: Unknown")
		}
		promMetricsEndpoint.Update(hostConfig.Host, hostConfig.Port, hostConfig.Protocol, success, time.Duration(elapsed)*time.Microsecond)

		// TODO: i wonder if i should remove the elapsed time from the sleep interval?
		time.Sleep(interval)
	}
}
