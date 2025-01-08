package monitor

import (
	"time"

	"github.com/whiskeyjimbo/CheckMate/internal/checkers"
	"github.com/whiskeyjimbo/CheckMate/internal/config"
	"github.com/whiskeyjimbo/CheckMate/internal/metrics"
)

func MonitorGroup(mc MonitoringContext) {
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
			hostResults := performHostChecks(mc, checker)
			stats := calculateGroupStats(hostResults)

			mc.Metrics.UpdateGroup(metrics.GroupMetrics{
				Site:         mc.Site,
				Group:        mc.Group.Name,
				Port:         mc.Check.Port,
				Protocol:     string(mc.Check.Protocol),
				Tags:         mc.Tags,
				Success:      !stats.AllDown,
				ResponseTime: stats.AvgResponseTime,
				HostsUp:      stats.SuccessfulChecks,
				HostsTotal:   stats.TotalHosts,
			})

			shouldUpdateDowntime := ruleModeResolver.ShouldTrigger(stats.AnyDown, stats.AllDown, mc.Check)
			downtime = updateDowntime(downtime, interval, !shouldUpdateDowntime)

			processRules(mc, stats, downtime, lastRuleEval, ruleModeResolver, hostResults)
			sleepUntilNextCheck(interval, time.Since(checkStart))
		}
	}
}
