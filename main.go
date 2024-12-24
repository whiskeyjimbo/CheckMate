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
	"github.com/whiskeyjimbo/CheckMate/pkg/rules"
	"go.uber.org/zap"
)

func main() {
	zapL, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
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

	confRules := config.RawRules

	promMetricsEndpoint := metrics.NewPrometheusMetrics(logger)

	for _, hostConfig := range config.Hosts {
		for _, checkConfig := range hostConfig.Checks {
			go monitorHost(logger, hostConfig.Host, checkConfig, promMetricsEndpoint, confRules)
		}
	}

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(":9100", nil); err != nil {
			logger.Fatalf("Failed to start Prometheus metrics server: %v", err)
		}
	}()

	waitForShutdown(logger)
}

func monitorHost(
	logger *zap.SugaredLogger,
	host string,
	checkConfig config.CheckConfig,
	promMetricsEndpoint *metrics.PrometheusMetrics,
	confRules []rules.Rule,
) {
	interval, err := time.ParseDuration(checkConfig.Interval)
	if err != nil {
		logger.Fatalf("Invalid interval %s: %v", checkConfig.Interval, err)
	}

	address := fmt.Sprintf("%s:%s", host, checkConfig.Port)

	checker, err := checkers.NewChecker(checkConfig.Protocol)
	if err != nil {
		logger.Fatalf("Unsupported protocol %s", checkConfig.Protocol)
	}

	var downtime time.Duration
	lastRuleEval := make(map[string]time.Time)

	for {
		checkStart := time.Now()
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

		if err != nil || !success {
			downtime += interval
		} else {
			downtime = 0
		}

		for _, rule := range confRules {
			if time.Since(lastRuleEval[rule.Name]) < time.Minute {
				continue
			}

			triggered, err := rules.EvaluateRule(rule, downtime, time.Duration(elapsed)*time.Microsecond)
			if err != nil {
				logger.Errorf("Failed to evaluate rule %s: %v", rule.Name, err)
				continue
			}

			if triggered {
				lastRuleEval[rule.Name] = time.Now()
				logger.Warnf("Rule triggered: host: %s, port: %s, protocol: %s, rule: %s", 
					host, checkConfig.Port, checkConfig.Protocol, rule.Name)
			}
		}

		sleepUntilNextCheck(interval, time.Since(checkStart))
	}
}

func sleepUntilNextCheck(interval, elapsed time.Duration) {
    sleepDuration := interval - elapsed
    if sleepDuration > 0 {
        time.Sleep(sleepDuration)
    }
}

func waitForShutdown(logger *zap.SugaredLogger) {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    <-c
    logger.Info("Received shutdown signal, exiting...")
}

