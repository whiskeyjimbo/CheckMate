package rules

import (
	"fmt"
	"time"
)

func evaluateCertRule(rule Rule, certExpiryTime time.Time) RuleResult {
	daysUntilExpiry := time.Until(certExpiryTime).Hours() / 24

	if daysUntilExpiry < float64(rule.MinDaysValidity) {
		return RuleResult{
			Satisfied: true,
			Message: fmt.Sprintf("Certificate expires in %.1f days (threshold: %d days)",
				daysUntilExpiry, rule.MinDaysValidity),
		}
	}

	return RuleResult{Satisfied: false}
}
