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
	"fmt"

	"github.com/whiskeyjimbo/CheckMate/internal/config"
	"github.com/whiskeyjimbo/CheckMate/internal/rules"
)

func BuildMessage(rule rules.Rule, result rules.RuleResult, mode config.RuleMode, successfulChecks, totalHosts int) string {
	if result.Error != nil {
		return fmt.Sprintf("Rule evaluation failed: %v", result.Error)
	}

	var modeInfo string
	switch mode {
	case config.RuleModeAny:
		modeInfo = fmt.Sprintf(" (%d/%d hosts up)", successfulChecks, totalHosts)
	case config.RuleModeAll:
		if successfulChecks == 0 {
			modeInfo = " (all hosts down)"
		} else {
			modeInfo = fmt.Sprintf(" (%d/%d hosts up)", successfulChecks, totalHosts)
		}
	}

	return fmt.Sprintf("Rule condition met: %s%s", rule.Name, modeInfo)
}

func GetLevel(result rules.RuleResult) NotificationLevel {
	if result.Error != nil {
		return ErrorLevel
	}
	return WarningLevel
}

func SendRuleNotifications(
	ctx context.Context,
	rule rules.Rule,
	notification Notification,
	notifierMap map[string]Notifier,
) error {
	if len(rule.Notifications) == 0 {
		for _, notifier := range notifierMap {
			if err := notifier.SendNotification(ctx, notification); err != nil {
				return fmt.Errorf("failed to send notification: %w", err)
			}
		}
		return nil
	}

	for _, notificationType := range rule.Notifications {
		if notifier, ok := notifierMap[notificationType]; ok {
			if err := notifier.SendNotification(ctx, notification); err != nil {
				return fmt.Errorf("failed to send notification %s: %w", notificationType, err)
			}
		}
	}
	return nil
}
