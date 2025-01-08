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
	Check   config.CheckConfig
	Rules   []rules.Rule
	Metrics *metrics.PrometheusMetrics
}

type BaseContext struct {
	Ctx         context.Context
	Logger      *zap.SugaredLogger
	Site        string
	Group       config.GroupConfig
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

type GroupStats struct {
	AnyDown          bool
	AllDown          bool
	SuccessfulChecks int
	TotalHosts       int
	AvgResponseTime  time.Duration
}

type HostResult struct {
	Success      bool
	ResponseTime time.Duration
	Error        error
}
