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
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	configFile := config.GetEnv("CONFIG_FILE", "config.yaml")
	config, err := config.LoadConfig(configFile)
	if err != nil {
		sugar.Infof("Using default values: %v", err)
	}

	interval, err := time.ParseDuration(config.Interval)
	if err != nil {
		sugar.Fatalf("Invalid interval %s: %v", config.Interval, err)
	}

	address := fmt.Sprintf("%s:%s", config.Host, config.Port)

	checker, err := checkers.NewChecker(config.Protocol)
	if err != nil {
		sugar.Fatalf("Unsupported protocol %s", config.Protocol)
	}

	promMetricsEndpoint := metrics.NewPrometheusMetrics(sugar)

	for {
		success, elapsed, err := checker.Check(address)
		if err != nil {
			sugar.With("status", "failure").With("responseTime_us", elapsed).Errorf("Check failed: %v", err)
		} else if success {
			sugar.With("status", "success").With("responseTime_us", elapsed).Info("Check succeeded")
		} else {
			sugar.With("status", "failure").With("responseTime_us", elapsed).Error("Check failed: Unknown")
		}
		promMetricsEndpoint.Update(config.Host, config.Port, config.Protocol, success, elapsed)

		time.Sleep(interval)
	}
}
