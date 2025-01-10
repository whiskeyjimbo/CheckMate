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
