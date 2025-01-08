package rules

import (
	"fmt"
	"time"
)

type CertRule struct {
	Name            string   `yaml:"name"`
	MinDaysValidity int      `yaml:"minDaysValidity"`
	Tags            []string `yaml:"tags"`
	Notifications   []string `yaml:"notifications"`
}

func EvaluateCertRule(rule CertRule, certExpiryTime time.Time) RuleResult {
	daysUntilExpiry := time.Until(certExpiryTime).Hours() / 24

	if daysUntilExpiry < float64(rule.MinDaysValidity) {
		return RuleResult{
			Satisfied: true,
			Message:   fmt.Sprintf("Certificate expires in %.1f days (threshold: %d days)", daysUntilExpiry, rule.MinDaysValidity),
		}
	}

	return RuleResult{Satisfied: false}
}
