package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/whiskeyjimbo/CheckMate/internal/checkers"
	"github.com/whiskeyjimbo/CheckMate/internal/config"
	"github.com/whiskeyjimbo/CheckMate/internal/notifications"
	"github.com/whiskeyjimbo/CheckMate/internal/rules"
	"github.com/whiskeyjimbo/CheckMate/internal/tags"
)

func performHostChecks(mc MonitoringContext, checker checkers.Checker) map[string]HostResult {
	hostResults := make(map[string]HostResult)

	for _, host := range mc.Group.Hosts {
		address := fmt.Sprintf("%s:%s", host.Host, mc.Check.Port)
		checkCtx, checkCancel := context.WithTimeout(mc.Ctx, 10*time.Second)
		result := checker.Check(checkCtx, address)
		checkCancel()

		hostResults[host.Host] = HostResult{
			Success:      result.Success,
			ResponseTime: result.ResponseTime,
			Error:        result.Error,
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

	return hostResults
}

func calculateGroupStats(results map[string]HostResult) GroupStats {
	stats := GroupStats{
		AllDown:    true,
		TotalHosts: len(results),
	}

	var totalResponseTime time.Duration
	for _, result := range results {
		if result.Success {
			stats.AllDown = false
			stats.SuccessfulChecks++
			totalResponseTime += result.ResponseTime
		} else {
			stats.AnyDown = true
		}
	}

	if stats.SuccessfulChecks > 0 {
		stats.AvgResponseTime = totalResponseTime / time.Duration(stats.SuccessfulChecks)
	}

	return stats
}

func processRules(
	mc MonitoringContext,
	stats GroupStats,
	downtime time.Duration,
	lastRuleEval map[string]time.Time,
	ruleModeResolver *config.RuleModeResolver,
	hostResults map[string]HostResult,
) {
	var failingHosts []string
	for host, result := range hostResults {
		if !result.Success {
			failingHosts = append(failingHosts, host)
		}
	}

	for _, rule := range mc.Rules {
		if !tags.HasMatching(mc.Tags, rule.Tags) {
			continue
		}

		ruleResult := rules.EvaluateRule(rule, downtime, stats.AvgResponseTime)
		if ruleResult.Error != nil || ruleResult.Satisfied {
			effectiveMode := ruleModeResolver.GetEffectiveRuleMode(mc.Check)

			if effectiveMode == config.RuleModeAny {
				// Send individual notifications for each failing host
				for _, failingHost := range failingHosts {
					notification := notifications.Notification{
						Message:  notifications.BuildMessage(rule, ruleResult, effectiveMode, stats.SuccessfulChecks, stats.TotalHosts),
						Level:    notifications.GetLevel(ruleResult),
						Tags:     mc.Tags,
						Site:     mc.Site,
						Group:    mc.Group.Name,
						Port:     mc.Check.Port,
						Protocol: string(mc.Check.Protocol),
						Host:     failingHost,
					}
					notifications.SendRuleNotifications(mc.Ctx, rule, notification, mc.NotifierMap)
				}
			} else {
				// Send single group-level notification
				notification := notifications.Notification{
					Message:  notifications.BuildMessage(rule, ruleResult, effectiveMode, stats.SuccessfulChecks, stats.TotalHosts),
					Level:    notifications.GetLevel(ruleResult),
					Tags:     mc.Tags,
					Site:     mc.Site,
					Group:    mc.Group.Name,
					Port:     mc.Check.Port,
					Protocol: string(mc.Check.Protocol),
					Host:     strings.Join(failingHosts, ","),
				}
				notifications.SendRuleNotifications(mc.Ctx, rule, notification, mc.NotifierMap)
			}
		}
		lastRuleEval[rule.Name] = time.Now()
	}
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
	case !ctx.Success:
		l.Error("Check failed")
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
