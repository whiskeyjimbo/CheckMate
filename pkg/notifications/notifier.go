package notifications

import "context"

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
	Host     string
	Port     string
	Protocol string
}

type Notifier interface {
	SendNotification(ctx context.Context, notification Notification) error
	Type() NotificationType
	Initialize(ctx context.Context) error
	Close() error
}
