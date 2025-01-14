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
	"fmt"
	"sync"
	"time"

	"github.com/whiskeyjimbo/CheckMate/internal/checkers"
	"github.com/whiskeyjimbo/CheckMate/internal/config"
	"github.com/whiskeyjimbo/CheckMate/internal/metrics"
	"github.com/whiskeyjimbo/CheckMate/internal/notifications"
	"github.com/whiskeyjimbo/CheckMate/internal/tags"
	"go.uber.org/zap"
)

func startMonitoring(ctx context.Context, wg *sync.WaitGroup, logger *zap.SugaredLogger, cfg *config.Config, metrics *metrics.PrometheusMetrics, notifierMap map[string]notifications.Notifier) {
	baseContext := BaseContext{
		Ctx:         ctx,
		Logger:      logger,
		Site:        cfg.MonitorSite,
		NotifierMap: notifierMap,
	}

	for _, site := range cfg.Sites {
		for _, group := range site.Groups {
			baseContext.Group = group
			baseContext.Tags = tags.MergeTags(site.Tags, group.Tags)

			for _, check := range group.Checks {
				wg.Add(1)
				go func(check config.CheckConfig) {
					defer wg.Done()
					MonitorGroup(MonitoringContext{
						Base:    baseContext,
						Check:   check,
						Rules:   cfg.Rules,
						Metrics: metrics,
					})
				}(check)
			}
		}
	}
}

func MonitorGroup(mc MonitoringContext) {
	checker, interval, err := initializeChecker(mc)
	if err != nil {
		mc.Base.Logger.Fatal(err)
	}

	var downtime time.Duration
	lastRuleEval := make(map[string]time.Time)
	ruleModeResolver := config.NewRuleModeResolver(mc.Base.Group)

	for {
		select {
		case <-mc.Base.Ctx.Done():
			return
		default:
			checkStart := time.Now()
			hostResults := performHostChecks(mc, checker)
			stats := calculateGroupStats(hostResults)

			mc.Metrics.UpdateGroup(metrics.GroupMetrics{
				Site:        mc.Base.Site,
				Group:       mc.Base.Group.Name,
				Port:        mc.Check.Port,
				Protocol:    string(mc.Check.Protocol),
				Tags:        mc.Base.Tags,
				HostResults: hostResults,
				HostsUp:     stats.SuccessfulChecks,
				HostsTotal:  stats.TotalHosts,
			})

			shouldUpdateDowntime := ruleModeResolver.ShouldTrigger(stats.AnyDown, stats.AllDown, mc.Check)
			downtime = updateDowntime(downtime, interval, !shouldUpdateDowntime)

			processRules(mc, stats, downtime, lastRuleEval, ruleModeResolver, hostResults)
			waitForNextCheckInterval(interval, time.Since(checkStart))
		}
	}
}

func initializeChecker(mc MonitoringContext) (checkers.Checker, time.Duration, error) {
	interval, err := time.ParseDuration(mc.Check.Interval)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid interval: %w", err)
	}

	protocol := checkers.Protocol(mc.Check.Protocol)
	if !protocol.IsValid() {
		supported := checkers.ListProtocols()
		return nil, 0, fmt.Errorf("unsupported protocol %q. Supported protocols: %v", protocol, supported)
	}

	checker, err := checkers.NewChecker(protocol)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create checker: %w", err)
	}

	// Set the timeout to the interval (maybe i should update to interval-1 second), which will be validated by the checker, and min/max will be enforced
	_ = checker.SetTimeout(interval)

	return checker, interval, nil
}
