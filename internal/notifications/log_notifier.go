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

	"go.uber.org/zap"
)

type LogNotifier struct {
	logger *zap.SugaredLogger
}

func NewLogNotifier(logger *zap.SugaredLogger) *LogNotifier {
	return &LogNotifier{
		logger: logger,
	}
}

func (n *LogNotifier) SendNotification(_ context.Context, notification Notification) error {
	logger := n.logger.With(
		"level", notification.Level,
		"site", notification.Site,
		"group", notification.Group,
		"host", notification.Host,
		"port", notification.Port,
		"protocol", notification.Protocol,
		"tags", notification.Tags,
	)

	switch notification.Level {
	case ErrorLevel:
		logger.Error(notification.Message)
	case WarningLevel:
		logger.Warn(notification.Message)
	default:
		logger.Info(notification.Message)
	}

	return nil
}

func (n *LogNotifier) Type() NotificationType {
	return LogNotification
}

func (n *LogNotifier) Initialize(ctx context.Context) error {
	_ = ctx
	return nil
}

func (n *LogNotifier) Close() error {
	return nil
}
