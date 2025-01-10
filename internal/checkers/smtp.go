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

package checkers

import (
	"context"
	"fmt"
	"net/smtp"
	"time"
)

const (
	defaultSMTPTimeout = 10 * time.Second
)

type SMTPChecker struct {
	BaseChecker
}

func NewSMTPChecker() *SMTPChecker {
	return &SMTPChecker{
		BaseChecker: BaseChecker{
			timeout: defaultSMTPTimeout,
		},
	}
}

func (c *SMTPChecker) Protocol() Protocol {
	return "SMTP"
}

func (c *SMTPChecker) Check(ctx context.Context, hosts []string, port string) []HostCheckResult {
	results := make([]HostCheckResult, 0, len(hosts))

	for _, host := range hosts {
		address := fmt.Sprintf("%s:%s", host, port)
		result := c.checkHost(ctx, host, func() error {
			client, err := smtp.Dial(address)
			if err != nil {
				return fmt.Errorf("smtp connection failed: %w", err)
			}
			defer client.Close()

			if err := client.Hello("checkmate.monitor"); err != nil {
				return fmt.Errorf("smtp hello failed: %w", err)
			}
			return nil
		})
		results = append(results, result)
	}

	return results
}

func init() {
	RegisterChecker("SMTP", func() Checker { return NewSMTPChecker() })
}
