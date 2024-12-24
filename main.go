package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/whiskeyjimbo/CheckMate/pkg/checkers"
	"github.com/whiskeyjimbo/CheckMate/pkg/config"
	"github.com/whiskeyjimbo/CheckMate/pkg/metrics"
	"github.com/whiskeyjimbo/CheckMate/pkg/rules"
	"go.uber.org/zap"
)

func main() {
	logger := initLogger()

	config, err := config.LoadConfiguration(os.Args)
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	metrics.StartMetricsServer(logger)

	confRules := config.RawRules

	promMetricsEndpoint := metrics.NewPrometheusMetrics(logger)

	for _, hostConfig := range config.Hosts {
		for _, checkConfig := range hostConfig.Checks {
			go monitorHost(logger, hostConfig.Host, checkConfig, promMetricsEndpoint, confRules)
		}
	}

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

		logCheckResult(logger, host, checkConfig, success, err, time.Duration(elapsed)*time.Microsecond)

		promMetricsEndpoint.Update(host, checkConfig.Port, checkConfig.Protocol, success, time.Duration(elapsed)*time.Microsecond)

		downtime = updateDowntime(downtime, interval, success)

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

func initLogger() *zap.SugaredLogger {
    zapL, err := zap.NewProduction()
    if err != nil {
        fmt.Printf("Failed to initialize logger: %v\n", err)
        os.Exit(1)
    }
    defer zapL.Sync()
    return zapL.Sugar()
}

func logCheckResult(logger *zap.SugaredLogger, host string, checkConfig config.CheckConfig, success bool, err error, elapsed time.Duration) {
    l := logger.With(
        "host", host,
        "port", checkConfig.Port,
        "protocol", checkConfig.Protocol,
        "responseTime_us", elapsed,
    )

    if err != nil {
        l.With("success", false).Warnf("Check failed: %v", err)
    } else if success {
        l.With("success", true).Info("Check succeeded")
    } else {
        l.With("success", false).Error("Check failed: Unknown")
    }
}

func sleepUntilNextCheck(interval, elapsed time.Duration) {
    sleepDuration := interval - elapsed
    if sleepDuration > 0 {
        time.Sleep(sleepDuration)
    }
}

func updateDowntime(currentDowntime, interval time.Duration, success bool) time.Duration {
    if !success {
        return currentDowntime + interval
    }
    return 0
}

func waitForShutdown(logger *zap.SugaredLogger) {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    <-c
    logger.Info("Received shutdown signal, exiting...")
}

