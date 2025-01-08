package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type SlackNotifier struct {
	webhookURL string
	client     *http.Client
}

type slackMessage struct {
	Text        string                   `json:"text"`
	Attachments []slackMessageAttachment `json:"attachments"`
}

type slackMessageAttachment struct {
	Color  string       `json:"color"`
	Fields []slackField `json:"fields"`
}

type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func NewSlackNotifier(webhookURL string) (*SlackNotifier, error) {
	if !strings.HasPrefix(webhookURL, "https://hooks.slack.com/services/") {
		return nil, fmt.Errorf("invalid Slack webhook URL format")
	}

	return &SlackNotifier{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (s *SlackNotifier) Notify(ctx context.Context, notification Notification) error {
	color := getColorForLevel(notification.Level)

	msg := slackMessage{
		Text: notification.Message,
		Attachments: []slackMessageAttachment{{
			Color: color,
			Fields: []slackField{
				{Title: "Site", Value: notification.Site, Short: true},
				{Title: "Group", Value: notification.Group, Short: true},
				{Title: "Host", Value: notification.Host, Short: true},
				{Title: "Protocol", Value: notification.Protocol, Short: true},
				{Title: "Port", Value: notification.Port, Short: true},
				{Title: "Tags", Value: fmt.Sprintf("%v", notification.Tags), Short: false},
			},
		}},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CheckMate-Monitor/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return fmt.Errorf("rate limited by Slack API")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack notification failed with status: %d", resp.StatusCode)
	}

	return nil
}

func getColorForLevel(level Level) string {
	switch level {
	case ErrorLevel:
		return "#FF0000" // Red
	case WarningLevel:
		return "#FFA500" // Orange
	default:
		return "#36a64f" // Green
	}
}

func (s *SlackNotifier) Name() string {
	return "slack"
}
