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
	"sync"
	"time"
)

const (
	smtpMinTimeout     = 5 * time.Second
	smtpMaxTimeout     = 15 * time.Second
	smtpDefaultTimeout = 10 * time.Second
)

type SMTPChecker struct {
	BaseChecker
	mu sync.RWMutex
}

func NewSMTPChecker() *SMTPChecker {
	return &SMTPChecker{
		BaseChecker: NewBaseChecker(TimeoutBounds{
			Min:     smtpMinTimeout,
			Max:     smtpMaxTimeout,
			Default: smtpDefaultTimeout,
		}),
	}
}

func (c *SMTPChecker) Protocol() Protocol {
	return "SMTP"
}

func (c *SMTPChecker) Check(ctx context.Context, hosts []string, port string) []HostCheckResult {
	return c.BaseChecker.Check(ctx, hosts, port, c.checkSMTP)
}

func (c *SMTPChecker) checkSMTP(ctx context.Context, host string, port string) (map[string]interface{}, error) {
	address := fmt.Sprintf("%s:%s", host, port)
	client, err := smtp.Dial(address)
	if err != nil {
		return nil, fmt.Errorf("smtp connection failed: %w", err)
	}
	defer client.Close()

	if err := client.Hello("checkmate.monitor"); err != nil {
		return nil, fmt.Errorf("smtp hello failed: %w", err)
	}
	return nil, nil
}

func init() {
	RegisterChecker("SMTP", func() Checker { return NewSMTPChecker() })
}
