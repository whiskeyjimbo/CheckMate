package monitor

import (
	"context"
	"time"

	"github.com/whiskeyjimbo/CheckMate/internal/config"
	"github.com/whiskeyjimbo/CheckMate/internal/metrics"
	"github.com/whiskeyjimbo/CheckMate/internal/notifications"
	"github.com/whiskeyjimbo/CheckMate/internal/rules"
	"go.uber.org/zap"
)

type MonitoringContext struct {
	Base    BaseContext
	Metrics *metrics.PrometheusMetrics
	Check   config.CheckConfig
	Rules   []rules.Rule
}

type BaseContext struct {
	Ctx         context.Context
	Logger      *zap.SugaredLogger
	NotifierMap map[string]notifications.Notifier
	Group       config.GroupConfig
	Site        string
	Tags        []string
}

type CheckContext struct {
	Error       error
	Logger      *zap.SugaredLogger
	Site        string
	Group       string
	Host        string
	CheckConfig config.CheckConfig
	Tags        []string
	Elapsed     time.Duration
	Success     bool
}

type GroupStats struct {
	AnyDown          bool
	AllDown          bool
	SuccessfulChecks int
	TotalHosts       int
	AvgResponseTime  time.Duration
}

type HostResult struct {
	Error        error
	ResponseTime time.Duration
	Success      bool
}
