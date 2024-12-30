package notifications

import (
	"context"

	"go.uber.org/zap"
)

// Stub out log notifier interface
type LogNotifier struct {
	logger *zap.SugaredLogger
}

func NewLogNotifier(logger *zap.SugaredLogger) *LogNotifier {
	return &LogNotifier{
		logger: logger,
	}
}

func (n *LogNotifier) SendNotification(ctx context.Context, notification Notification) error {
	n.logger.With(
		"level", notification.Level,
		"site", notification.Site,
		"group", notification.Group,
		"host", notification.Host,
		"port", notification.Port,
		"protocol", notification.Protocol,
		"tags", notification.Tags,
	).Info(notification.Message)

	return nil
}

func (n *LogNotifier) Type() NotificationType {
	return LogNotification
}

func (n *LogNotifier) Initialize(ctx context.Context) error {
	return nil
}

func (n *LogNotifier) Close() error {
	return nil
}
