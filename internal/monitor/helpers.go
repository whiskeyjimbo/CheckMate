// Copyright (C) 2025 Jeff Rose
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package monitor

import (
	"context"
	"strings"
	"time"

	"github.com/whiskeyjimbo/CheckMate/internal/checkers"
	"github.com/whiskeyjimbo/CheckMate/internal/config"
	"github.com/whiskeyjimbo/CheckMate/internal/metrics"
	"github.com/whiskeyjimbo/CheckMate/internal/notifications"
	"github.com/whiskeyjimbo/CheckMate/internal/rules"
	"github.com/whiskeyjimbo/CheckMate/internal/tags"
)

func performHostChecks(mc MonitoringContext, checker checkers.Checker) map[string]metrics.HostResult {
	hostResults := make(map[string]metrics.HostResult)
	hosts := make([]string, 0, len(mc.Base.Group.Hosts))

	for _, host := range mc.Base.Group.Hosts {
		hosts = append(hosts, host.Host)
	}

	checkCtx, checkCancel := context.WithTimeout(mc.Base.Ctx, 10*time.Second)
	results := checker.Check(checkCtx, hosts, mc.Check.Port)
	checkCancel()

	for _, result := range results {
		hostResults[result.Host] = metrics.HostResult{
			Success:      result.Success,
			ResponseTime: result.ResponseTime,
			Error:        result.Error,
		}

		logCheckResult(CheckContext{
			Logger:      mc.Base.Logger,
			Site:        mc.Base.Site,
			Group:       mc.Base.Group.Name,
			Host:        result.Host,
			CheckConfig: mc.Check,
			Success:     result.Success,
			Error:       result.Error,
			Elapsed:     result.ResponseTime,
			Tags:        mc.Base.Tags,
		})
	}

	return hostResults
}

func calculateGroupStats(results map[string]metrics.HostResult) GroupStats {
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
	hostResults map[string]metrics.HostResult,
) {
	failingHosts := collectFailingHosts(hostResults)

	for _, rule := range mc.Rules {
		if !tags.HasMatching(mc.Base.Tags, rule.Tags) {
			continue
		}

		evaluateAndProcessRule(mc, rule, stats, downtime, ruleModeResolver, failingHosts)
		lastRuleEval[rule.Name] = time.Now()
	}
}

func collectFailingHosts(hostResults map[string]metrics.HostResult) []string {
	var failingHosts []string
	for host, result := range hostResults {
		if !result.Success {
			failingHosts = append(failingHosts, host)
		}
	}
	return failingHosts
}

func evaluateAndProcessRule(
	mc MonitoringContext,
	rule rules.Rule,
	stats GroupStats,
	downtime time.Duration,
	ruleModeResolver *config.RuleModeResolver,
	failingHosts []string,
) {
	params := rules.EvaluationParams{
		Downtime:     downtime,
		ResponseTime: stats.AvgResponseTime,
	}
	ruleResult := rules.EvaluateRule(rule, params)
	if !shouldSendNotification(ruleResult) {
		return
	}

	effectiveMode := ruleModeResolver.GetEffectiveRuleMode(mc.Check)
	sendNotifications(mc, rule, ruleResult, effectiveMode, stats, failingHosts)
}

func shouldSendNotification(result rules.RuleResult) bool {
	return result.Error != nil || result.Satisfied
}

func sendNotifications(
	mc MonitoringContext,
	rule rules.Rule,
	ruleResult rules.RuleResult,
	effectiveMode config.RuleMode,
	stats GroupStats,
	failingHosts []string,
) {
	if effectiveMode == config.RuleModeAny {
		sendIndividualNotifications(mc, rule, ruleResult, effectiveMode, stats, failingHosts)
	} else {
		sendGroupNotification(mc, rule, ruleResult, effectiveMode, stats, failingHosts)
	}
}

func sendIndividualNotifications(
	mc MonitoringContext,
	rule rules.Rule,
	ruleResult rules.RuleResult,
	effectiveMode config.RuleMode,
	stats GroupStats,
	failingHosts []string,
) {
	for _, failingHost := range failingHosts {
		notification := createNotification(mc, rule, ruleResult, effectiveMode, stats, failingHost)
		notifications.SendRuleNotifications(mc.Base.Ctx, rule, notification, mc.Base.NotifierMap)
	}
}

func sendGroupNotification(
	mc MonitoringContext,
	rule rules.Rule,
	ruleResult rules.RuleResult,
	effectiveMode config.RuleMode,
	stats GroupStats,
	failingHosts []string,
) {
	notification := createNotification(mc, rule, ruleResult, effectiveMode, stats, strings.Join(failingHosts, ","))
	notifications.SendRuleNotifications(mc.Base.Ctx, rule, notification, mc.Base.NotifierMap)
}

func createNotification(
	mc MonitoringContext,
	rule rules.Rule,
	ruleResult rules.RuleResult,
	effectiveMode config.RuleMode,
	stats GroupStats,
	host string,
) notifications.Notification {
	return notifications.Notification{
		Message:  notifications.BuildMessage(rule, ruleResult, effectiveMode, stats.SuccessfulChecks, stats.TotalHosts),
		Level:    notifications.GetLevel(ruleResult),
		Tags:     mc.Base.Tags,
		Site:     mc.Base.Site,
		Group:    mc.Base.Group.Name,
		Port:     mc.Check.Port,
		Protocol: string(mc.Check.Protocol),
		Host:     host,
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

func waitForNextCheckInterval(interval, elapsed time.Duration) {
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
