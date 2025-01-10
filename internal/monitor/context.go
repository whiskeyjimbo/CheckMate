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
