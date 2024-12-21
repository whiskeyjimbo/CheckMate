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
		for _, checkConfig := range hostConfig.Checks {
			go monitorHost(logger, hostConfig.Host, checkConfig, promMetricsEndpoint)
		}
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

func monitorHost(logger *zap.SugaredLogger, host string, checkConfig config.CheckConfig, promMetricsEndpoint *metrics.PrometheusMetrics) {
	interval, err := time.ParseDuration(checkConfig.Interval)
	if err != nil {
		logger.Fatalf("Invalid interval %s: %v", checkConfig.Interval, err)
	}

	address := fmt.Sprintf("%s:%s", host, checkConfig.Port)

	checker, err := checkers.NewChecker(checkConfig.Protocol)
	if err != nil {
		logger.Fatalf("Unsupported protocol %s", checkConfig.Protocol)
	}

	for {
		success, elapsed, err := checker.Check(address)
		l := logger.
			With("host", host).
			With("port", checkConfig.Port).
			With("protocol", checkConfig.Protocol).
			With("responseTime_us", elapsed)
		if err != nil {
			l.With("success", false).Warnf("Check failed: %v", err)
		} else if success {
			l.With("success", true).Infof("Check succeeded")
		} else {
			l.With("success", false).Error("Check failed: Unknown")
		}
		promMetricsEndpoint.Update(host, checkConfig.Port, checkConfig.Protocol, success, time.Duration(elapsed)*time.Microsecond)

		// TODO: i wonder if i should remove the elapsed time from the sleep interval?
		time.Sleep(interval)
	}
}
