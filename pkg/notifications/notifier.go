package notifications

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type NotificationType string

const (
	LogNotification NotificationType = "log"
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
	Tags     []string
	Site     string
	Group    string
	Host     string
	Port     string
	Protocol string
	PortTags []string
	HostTags []string
	SiteTags []string
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
		return nil, fmt.Errorf("log notifier requires a logger")
	// case SlackNotification:
	//     return NewSlackNotifier(opts...), nil
	// case EmailNotification:
	//     return NewEmailNotifier(opts...), nil
	default:
		return nil, fmt.Errorf("unsupported notification type: %s", notifierType)
	}
}
