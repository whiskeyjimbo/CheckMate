package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/whiskeyjimbo/CheckMate/pkg/checkers"
	"github.com/whiskeyjimbo/CheckMate/pkg/config"
	"github.com/whiskeyjimbo/CheckMate/pkg/metrics"
	"github.com/whiskeyjimbo/CheckMate/pkg/rules"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"
)

func main() {
	logger := initLogger()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := config.LoadConfiguration(os.Args)
	if err != nil {
		logger.Fatal(err)
	}

	metrics.StartMetricsServer(logger)

	promMetricsEndpoint := metrics.NewPrometheusMetrics(logger)

	var wg sync.WaitGroup
	startMonitoring(ctx, &wg, logger, config, promMetricsEndpoint)

	waitForShutdown(logger, cancel, &wg)
}

func startMonitoring(ctx context.Context, wg *sync.WaitGroup, logger *zap.SugaredLogger, cfg *config.Config, metrics *metrics.PrometheusMetrics) {
	for _, hostConfig := range cfg.Hosts {
		for _, checkConfig := range hostConfig.Checks {
			wg.Add(1)
			go func(host string, check config.CheckConfig) {
				defer wg.Done()
				monitorHost(ctx, logger, host, check, metrics, cfg.RawRules)
			}(hostConfig.Host, checkConfig)
		}
	}
}

func monitorHost(
	ctx context.Context,
	logger *zap.SugaredLogger,
	host string,
	checkConfig config.CheckConfig,
	promMetricsEndpoint *metrics.PrometheusMetrics,
	confRules []rules.Rule,
) {
	interval, err := time.ParseDuration(checkConfig.Interval)
	if err != nil {
		logger.Fatal(err)
	}

	address := fmt.Sprintf("%s:%s", host, checkConfig.Port)
	checker, err := checkers.NewChecker(checkConfig.Protocol)
	if err != nil {
		logger.Fatal(err)
	}

	var downtime time.Duration
	lastRuleEval := make(map[string]time.Time)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			checkStart := time.Now()
			checkCtx, checkCancel := context.WithTimeout(ctx, 10*time.Second)
			result := checker.Check(checkCtx, address)
			checkCancel()

			logCheckResult(logger, host, checkConfig, result.Success, result.Error, result.ResponseTime)

			promMetricsEndpoint.Update(host, checkConfig.Port, checkConfig.Protocol, result.Success, result.ResponseTime)

			downtime = updateDowntime(downtime, interval, result.Success)

			for _, rule := range confRules {
				if time.Since(lastRuleEval[rule.Name]) < time.Minute {
					continue
				}

				triggered, err := rules.EvaluateRule(rule, downtime, result.ResponseTime)
				if err != nil {
					logger.Error(err)
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
		"success", success,
	)

	switch {
	case err != nil:
		l.Warn(err)
	case success:
		l.Info("Check succeeded")
	default:
		l.Error("Unknown failure")
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

func waitForShutdown(logger *zap.SugaredLogger, cancel context.CancelFunc, wg *sync.WaitGroup) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	logger.Info("Received shutdown signal, exiting...")
	cancel()
	wg.Wait()
}
