package main

import (
	"fmt"
	"time"

	"github.com/whiskeyjimbo/CheckMate/pkg/checkers"
	"github.com/whiskeyjimbo/CheckMate/pkg/config"
	"github.com/whiskeyjimbo/CheckMate/pkg/metrics"
	"go.uber.org/zap"
)

func main() {
	zapL, _ := zap.NewProduction()
	defer zapL.Sync()
	logger := zapL.Sugar()

	configFile := config.GetEnv("CONFIG_FILE", "config.yaml")
	config, err := config.LoadConfig(configFile)
	if err != nil {
		logger.Warnf("Using default values: %v", err)
	}

	interval, err := time.ParseDuration(config.Interval)
	if err != nil {
		logger.Fatalf("Invalid interval %s: %v", config.Interval, err)
	}

	address := fmt.Sprintf("%s:%s", config.Host, config.Port)

	checker, err := checkers.NewChecker(config.Protocol)
	if err != nil {
		logger.Fatalf("Unsupported protocol %s", config.Protocol)
	}

	promMetricsEndpoint := metrics.NewPrometheusMetrics(logger)

	for {
		success, elapsed, err := checker.Check(address)
		l := logger.
			With("host", config.Host).
			With("port", config.Port).
			With("protocol", config.Protocol).
			With("responseTime_us", elapsed)
		if err != nil {
			l.With("success", false).Error("Check failed: %v", err)
		} else if success {
			l.With("success", true).Info("Check succeeded")
		} else {
			l.With("success", false).Error("Check failed: Unknown")
		}
		promMetricsEndpoint.Update(config.Host, config.Port, config.Protocol, success, elapsed)

		time.Sleep(interval)
	}
}
