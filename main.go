package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/whiskeyjimbo/CheckMate/internal/config"
	"github.com/whiskeyjimbo/CheckMate/internal/health"
	"github.com/whiskeyjimbo/CheckMate/internal/metrics"
	"github.com/whiskeyjimbo/CheckMate/internal/monitor"
	"github.com/whiskeyjimbo/CheckMate/internal/notifications"
	"github.com/whiskeyjimbo/CheckMate/internal/tags"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"
)

func main() {
	logger := initLogger()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	health.SetReady(false)

	config, err := config.LoadConfiguration(os.Args)
	if err != nil {
		logger.Fatal(err)
	}

	notifierMap := initializeNotifiers(ctx, logger, config.Notifications)
	metrics.StartMetricsServer(logger)
	health.SetReady(true)

	var wg sync.WaitGroup
	startMonitoring(ctx, &wg, logger, config, metrics.NewPrometheusMetrics(logger, config.MonitorSite), notifierMap)
	waitForShutdown(logger, cancel, &wg)
}

func startMonitoring(
	ctx context.Context,
	wg *sync.WaitGroup,
	logger *zap.SugaredLogger,
	cfg *config.Config,
	metrics *metrics.PrometheusMetrics,
	notifierMap map[string]notifications.Notifier,
) {
	for _, site := range cfg.Sites {
		for _, group := range site.Groups {
			for _, checkConfig := range group.Checks {
				wg.Add(1)
				combinedTags := tags.Deduplicate(append(append(site.Tags, group.Tags...), checkConfig.Tags...))

				go func(site string, group config.GroupConfig, check config.CheckConfig, tags []string) {
					defer wg.Done()
					mc := monitor.MonitoringContext{
						Base: monitor.BaseContext{
							Ctx:         ctx,
							Logger:      logger,
							Site:        site,
							Group:       group,
							Tags:        tags,
							NotifierMap: notifierMap,
						},
						Check:   check,
						Metrics: metrics,
						Rules:   cfg.Rules,
					}
					monitor.MonitorGroup(mc)
				}(site.Name, group, checkConfig, combinedTags)
			}
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

func initializeNotifiers(ctx context.Context, logger *zap.SugaredLogger, configs []config.NotificationConfig) map[string]notifications.Notifier {
	notifierMap := make(map[string]notifications.Notifier)

	for _, n := range configs {
		notifier, err := notifications.NewNotifier(n.Type, logger)
		if err != nil {
			logger.Fatal(err)
		}
		if err := notifier.Initialize(ctx); err != nil {
			logger.Fatal(err)
		}
		notifierMap[n.Type] = notifier
	}

	return notifierMap
}

func waitForShutdown(logger *zap.SugaredLogger, cancel context.CancelFunc, wg *sync.WaitGroup) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	logger.Info("Received shutdown signal, exiting...")
	cancel()
	wg.Wait()
}
