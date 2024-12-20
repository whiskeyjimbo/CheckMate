package notifications

// stub out notifications interface, will probably need more params
type Notifier interface {
	SendNotification(message string) error
}
