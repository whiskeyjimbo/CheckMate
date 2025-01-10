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

package notifications

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
)

type NotificationType string

const (
	LogNotification NotificationType = "log"
	// Future notification types:
	// SlackNotification NotificationType = "slack"
	// EmailNotification NotificationType = "email"
)

type NotificationLevel string

const (
	InfoLevel    NotificationLevel = "info"
	WarningLevel NotificationLevel = "warning"
	ErrorLevel   NotificationLevel = "error"
)

type Notification struct {
	Message  string
	Level    NotificationLevel
	Site     string
	Group    string
	Host     string
	Port     string
	Protocol string
	Tags     []string
}

type Notifier interface {
	SendNotification(ctx context.Context, notification Notification) error
	Type() NotificationType
	Initialize(ctx context.Context) error
	Close() error
}

func NewNotifier(notifierType string, opts ...interface{}) (Notifier, error) {
	switch NotificationType(notifierType) {
	case LogNotification:
		if len(opts) > 0 {
			if logger, ok := opts[0].(*zap.SugaredLogger); ok {
				return NewLogNotifier(logger), nil
			}
		}
		return nil, errors.New("log notifier requires a logger")
	default:
		return nil, fmt.Errorf("unsupported notification type: %s", notifierType)
	}
}
