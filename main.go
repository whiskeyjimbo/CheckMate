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
	"github.com/whiskeyjimbo/CheckMate/pkg/health"
	"github.com/whiskeyjimbo/CheckMate/pkg/metrics"
	"github.com/whiskeyjimbo/CheckMate/pkg/notifications"
	"github.com/whiskeyjimbo/CheckMate/pkg/rules"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"
)

type hostResult struct {
	success      bool
	responseTime time.Duration
	err          error
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
	startMonitoring(ctx, &wg, logger, config, metrics.NewPrometheusMetrics(logger), notifierMap)

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
					monitorGroup(ctx, logger, site, group, check, metrics, cfg.Rules, tags, notifierMap)
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

func performHostChecks(
	ctx context.Context,
	checker checkers.Checker,
	hosts []config.HostConfig,
	checkConfig config.CheckConfig,
) map[string]hostResult {
	hostResults := make(map[string]hostResult)
	for _, host := range hosts {
		address := fmt.Sprintf("%s:%s", host.Host, checkConfig.Port)
		checkCtx, checkCancel := context.WithTimeout(ctx, 10*time.Second)
		result := checker.Check(checkCtx, address)
		checkCancel()

		hostResults[host.Host] = hostResult{
			success:      result.Success,
			responseTime: result.ResponseTime,
			err:          result.Error,
		}
	}
	return hostResults
}

func processResults(hostResults map[string]hostResult) (time.Duration, bool, bool, int) {
	var totalResponseTime time.Duration
	allDown := true
	anyDown := false
	successfulChecks := 0

	for _, result := range hostResults {
		if result.success {
			allDown = false
			totalResponseTime += result.responseTime
			successfulChecks++
		} else {
			anyDown = true
		}
	}

	return totalResponseTime, allDown, anyDown, successfulChecks
}

func evaluateRules(
	ctx context.Context,
	confRules []rules.Rule,
	groupTags []string,
	lastRuleEval map[string]time.Time,
	downtime time.Duration,
	avgResponseTime time.Duration,
	ruleMode config.RuleMode,
	successfulChecks, totalHosts int,
	site string,
	group string,
	checkConfig config.CheckConfig,
	notifierMap map[string]notifications.Notifier,
) {
	for _, rule := range confRules {
		if !hasMatchingTags(groupTags, rule.Tags) {
			continue
		}

		if time.Since(lastRuleEval[rule.Name]) < time.Minute {
			continue
		}

		ruleResult := rules.EvaluateRule(rule, downtime, avgResponseTime)
		if ruleResult.Error != nil || ruleResult.Satisfied {
			notification := notifications.Notification{
				Message:  buildNotificationMessage(rule, ruleResult, ruleMode, successfulChecks, totalHosts),
				Level:    getNotificationLevel(ruleResult),
				Tags:     groupTags,
				Site:     site,
				Group:    group,
				Port:     checkConfig.Port,
				Protocol: string(checkConfig.Protocol),
			}
			sendRuleNotifications(ctx, rule, notification, notifierMap)
		}
		lastRuleEval[rule.Name] = time.Now()
	}
}

func processGroupCheck(
	ctx context.Context,
	logger *zap.SugaredLogger,
	site string,
	group config.GroupConfig,
	checkConfig config.CheckConfig,
	checker checkers.Checker,
	groupTags []string,
) (time.Duration, bool, bool, int, int) {
	hostResults := performHostChecks(ctx, checker, group.Hosts, checkConfig)

	totalResponseTime, allDown, anyDown, successfulChecks := processResults(hostResults)

	for host, result := range hostResults {
		logCheckResult(logger, site, group.Name, host, checkConfig, result.success, result.err, result.responseTime, groupTags)
	}

	return totalResponseTime, allDown, anyDown, successfulChecks, len(group.Hosts)
}

func updateMetricsAndDowntime(
	totalResponseTime time.Duration,
	allDown bool,
	anyDown bool,
	successfulChecks int,
	interval time.Duration,
	group config.GroupConfig,
	site string,
	checkConfig config.CheckConfig,
	groupTags []string,
	metrics *metrics.PrometheusMetrics,
) (time.Duration, time.Duration) {
	var avgResponseTime time.Duration
	if successfulChecks > 0 {
		avgResponseTime = totalResponseTime / time.Duration(successfulChecks)
	}

	metrics.UpdateGroup(site, group.Name, checkConfig.Port, string(checkConfig.Protocol), groupTags, !allDown, avgResponseTime)

	shouldUpdateDowntime := group.RuleMode == config.RuleModeAny && anyDown || group.RuleMode != config.RuleModeAny && allDown
	return avgResponseTime, updateDowntime(0, interval, !shouldUpdateDowntime)
}

func monitorGroup(
	ctx context.Context,
	logger *zap.SugaredLogger,
	site string,
	group config.GroupConfig,
	checkConfig config.CheckConfig,
	promMetricsEndpoint *metrics.PrometheusMetrics,
	confRules []rules.Rule,
	groupTags []string,
	notifierMap map[string]notifications.Notifier,
) {
	interval, err := time.ParseDuration(checkConfig.Interval)
	if err != nil {
		logger.Fatal(err)
	}

	checker, err := checkers.NewChecker(checkConfig.Protocol)
	if err != nil {
		logger.Fatal(err)
	}

	lastRuleEval := make(map[string]time.Time)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			checkStart := time.Now()

			totalResponseTime, allDown, anyDown, successfulChecks, totalHosts := processGroupCheck(
				ctx, logger, site, group, checkConfig, checker, groupTags,
			)

			avgResponseTime, downtime := updateMetricsAndDowntime(
				totalResponseTime, allDown, anyDown, successfulChecks,
				interval, group, site, checkConfig, groupTags,
				promMetricsEndpoint,
			)

			evaluateRules(
				ctx, confRules, groupTags, lastRuleEval,
				downtime, avgResponseTime, group.RuleMode,
				successfulChecks, totalHosts, site,
				group.Name, checkConfig, notifierMap,
			)

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

func logCheckResult(logger *zap.SugaredLogger, site string, group string, host string, checkConfig config.CheckConfig, success bool, err error, elapsed time.Duration, tags []string) {
	l := logger.With(
		"site", site,
		"group", group,
		"host", host,
		"port", checkConfig.Port,
		"protocol", checkConfig.Protocol,
		"responseTime_us", elapsed,
		"success", success,
		"tags", tags,
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
