package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/whiskeyjimbo/CheckMate/pkg/checkers"
	"github.com/whiskeyjimbo/CheckMate/pkg/config"
	"github.com/whiskeyjimbo/CheckMate/pkg/health"
	"github.com/whiskeyjimbo/CheckMate/pkg/metrics"
	"github.com/whiskeyjimbo/CheckMate/pkg/notifications"
	"github.com/whiskeyjimbo/CheckMate/pkg/rules"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"
)

type MonitoringContext struct {
	Ctx         context.Context
	Logger      *zap.SugaredLogger
	Site        string
	Group       config.GroupConfig
	Check       config.CheckConfig
	Metrics     *metrics.PrometheusMetrics
	Rules       []rules.Rule
	Tags        []string
	NotifierMap map[string]notifications.Notifier
}

type CheckContext struct {
	Logger      *zap.SugaredLogger
	Site        string
	Group       string
	Host        string
	CheckConfig config.CheckConfig
	Success     bool
	Error       error
	Elapsed     time.Duration
	Tags        []string
}

func main() {
	logger := initLogger()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	health.SetReady(false)

	config, err := config.LoadConfiguration(os.Args)
	if err != nil {
		logger.Fatal(err)
	}

	notifierMap := make(map[string]notifications.Notifier)

	for _, n := range config.Notifications {
		notifier, err := notifications.NewNotifier(n.Type, logger)
		if err != nil {
			logger.Fatal(err)
		}
		if err := notifier.Initialize(ctx); err != nil {
			logger.Fatal(err)
		}
		defer notifier.Close()
		notifierMap[n.Type] = notifier
	}

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
				combinedTags := deduplicateTags(append(append(site.Tags, group.Tags...), checkConfig.Tags...))

				go func(site string, group config.GroupConfig, check config.CheckConfig, tags []string) {
					defer wg.Done()
					mc := MonitoringContext{
						Ctx:         ctx,
						Logger:      logger,
						Site:        site,
						Group:       group,
						Check:       check,
						Metrics:     metrics,
						Rules:       cfg.Rules,
						Tags:        tags,
						NotifierMap: notifierMap,
					}
					monitorGroup(mc)
				}(site.Name, group, checkConfig, combinedTags)
			}
		}
	}
}

func deduplicateTags(tags []string) []string {
	seen := make(map[string]bool)
	deduped := make([]string, 0, len(tags))

	for _, tag := range tags {
		if !seen[tag] {
			seen[tag] = true
			deduped = append(deduped, tag)
		}
	}
	return deduped
}

// TODO: getting pretty large, need to break up
func monitorGroup(mc MonitoringContext) {
	interval, err := time.ParseDuration(mc.Check.Interval)
	if err != nil {
		mc.Logger.Fatal(err)
	}

	checker, err := checkers.NewChecker(mc.Check.Protocol)
	if err != nil {
		mc.Logger.Fatal(err)
	}

	var downtime time.Duration
	lastRuleEval := make(map[string]time.Time)

	ruleModeResolver := config.NewRuleModeResolver(mc.Group)

	for {
		select {
		case <-mc.Ctx.Done():
			return
		default:
			checkStart := time.Now()

			var totalResponseTime time.Duration
			allDown := true
			anyDown := false
			successfulChecks := 0
			totalHosts := len(mc.Group.Hosts)

			type hostResult struct {
				success      bool
				responseTime time.Duration
				err          error
			}
			hostResults := make(map[string]hostResult)

			for _, host := range mc.Group.Hosts {
				address := fmt.Sprintf("%s:%s", host.Host, mc.Check.Port)
				checkCtx, checkCancel := context.WithTimeout(mc.Ctx, 10*time.Second)
				result := checker.Check(checkCtx, address)
				checkCancel()

				hostResults[host.Host] = hostResult{
					success:      result.Success,
					responseTime: result.ResponseTime,
					err:          result.Error,
				}

				if result.Success {
					allDown = false
					totalResponseTime += result.ResponseTime
					successfulChecks++
				} else {
					anyDown = true
				}

				logCheckResult(CheckContext{
					Logger:      mc.Logger,
					Site:        mc.Site,
					Group:       mc.Group.Name,
					Host:        host.Host,
					CheckConfig: mc.Check,
					Success:     result.Success,
					Error:       result.Error,
					Elapsed:     result.ResponseTime,
					Tags:        mc.Tags,
				})
			}

			var avgResponseTime time.Duration
			if successfulChecks > 0 {
				avgResponseTime = totalResponseTime / time.Duration(successfulChecks)
			}

			mc.Metrics.UpdateGroup(
				mc.Site,
				mc.Group.Name,
				mc.Check.Port,
				string(mc.Check.Protocol),
				mc.Tags,
				!allDown,
				avgResponseTime,
				successfulChecks,
				totalHosts,
			)

			shouldUpdateDowntime := ruleModeResolver.ShouldTrigger(anyDown, allDown, mc.Check)
			downtime = updateDowntime(downtime, interval, !shouldUpdateDowntime)

			var failingHosts []string
			for host, result := range hostResults {
				if !result.success {
					failingHosts = append(failingHosts, host)
				}
			}

			for _, rule := range mc.Rules {
				if !hasMatchingTags(mc.Tags, rule.Tags) {
					continue
				}

				if time.Since(lastRuleEval[rule.Name]) < time.Minute {
					continue
				}

				ruleResult := rules.EvaluateRule(rule, downtime, avgResponseTime)
				if ruleResult.Error != nil || ruleResult.Satisfied {
					effectiveMode := ruleModeResolver.GetEffectiveRuleMode(mc.Check)

					if effectiveMode == config.RuleModeAny {
						// Send individual notifications for each failing host if rule mode is any
						for _, failingHost := range failingHosts {
							notification := notifications.Notification{
								Message:  buildNotificationMessage(rule, ruleResult, effectiveMode, successfulChecks, totalHosts),
								Level:    getNotificationLevel(ruleResult),
								Tags:     mc.Tags,
								Site:     mc.Site,
								Group:    mc.Group.Name,
								Port:     mc.Check.Port,
								Protocol: string(mc.Check.Protocol),
								Host:     failingHost,
							}
							sendRuleNotifications(mc.Ctx, rule, notification, mc.NotifierMap)
						}
					} else {
						// Send single group-level notification if rule mode is all
						notification := notifications.Notification{
							Message:  buildNotificationMessage(rule, ruleResult, effectiveMode, successfulChecks, totalHosts),
							Level:    getNotificationLevel(ruleResult),
							Tags:     mc.Tags,
							Site:     mc.Site,
							Group:    mc.Group.Name,
							Port:     mc.Check.Port,
							Protocol: string(mc.Check.Protocol),
							Host:     strings.Join(failingHosts, ","),
						}
						sendRuleNotifications(mc.Ctx, rule, notification, mc.NotifierMap)
					}
				}
				lastRuleEval[rule.Name] = time.Now()
			}

			sleepUntilNextCheck(interval, time.Since(checkStart))
		}
	}
}

func buildNotificationMessage(rule rules.Rule, result rules.RuleResult, mode config.RuleMode, successfulChecks, totalHosts int) string {
	if result.Error != nil {
		return fmt.Sprintf("Rule evaluation failed: %v", result.Error)
	}

	var modeInfo string
	switch mode {
	case config.RuleModeAny:
		modeInfo = fmt.Sprintf(" (%d/%d hosts up)", successfulChecks, totalHosts)
	case config.RuleModeAll:
		if successfulChecks == 0 {
			modeInfo = " (all hosts down)"
		} else {
			modeInfo = fmt.Sprintf(" (%d/%d hosts up)", successfulChecks, totalHosts)
		}
	}

	return fmt.Sprintf("Rule condition met: %s%s", rule.Name, modeInfo)
}

func getNotificationLevel(result rules.RuleResult) notifications.NotificationLevel {
	if result.Error != nil {
		return notifications.ErrorLevel
	}
	return notifications.WarningLevel
}

func sendRuleNotifications(
	ctx context.Context,
	rule rules.Rule,
	notification notifications.Notification,
	notifierMap map[string]notifications.Notifier,
) {
	if len(rule.Notifications) == 0 {
		for _, notifier := range notifierMap {
			notifier.SendNotification(ctx, notification)
		}
		return
	}

	for _, notificationType := range rule.Notifications {
		if notifier, ok := notifierMap[notificationType]; ok {
			notifier.SendNotification(ctx, notification)
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

func logCheckResult(ctx CheckContext) {
	l := ctx.Logger.With(
		"site", ctx.Site,
		"group", ctx.Group,
		"host", ctx.Host,
		"port", ctx.CheckConfig.Port,
		"protocol", ctx.CheckConfig.Protocol,
		"latency_ms", ctx.Elapsed.Milliseconds(),
		"success", ctx.Success,
		"tags", ctx.Tags,
	)

	switch {
	case ctx.Error != nil:
		l.Warn(ctx.Error)
	case ctx.Success:
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

func hasMatchingTags(allTags, ruleTags []string) bool {
	if len(ruleTags) == 0 {
		return true
	}

	tagMap := make(map[string]bool)
	for _, tag := range allTags {
		tagMap[tag] = true
	}

	for _, ruleTag := range ruleTags {
		if tagMap[ruleTag] {
			return true
		}
	}
	return false
}
