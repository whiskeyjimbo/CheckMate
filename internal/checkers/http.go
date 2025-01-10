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
	"net/http"
	"time"
)

const (
	defaultHTTPTimeout = 10 * time.Second
)

type HTTPChecker struct {
	BaseChecker
	client *http.Client
}

func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{
		BaseChecker: BaseChecker{
			timeout: defaultHTTPTimeout,
		},
		client: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}
}

func (c *HTTPChecker) Protocol() Protocol {
	return "HTTP"
}

func (c *HTTPChecker) Check(ctx context.Context, hosts []string, port string) []HostCheckResult {
	results := make([]HostCheckResult, 0, len(hosts))

	for _, host := range hosts {
		url := fmt.Sprintf("http://%s:%s", host, port)
		result := c.checkHost(ctx, host, func() error {
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			resp, err := c.client.Do(req)
			if err != nil {
				return fmt.Errorf("http request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				return fmt.Errorf("http status error: %d", resp.StatusCode)
			}
			return nil
		})
		results = append(results, result)
	}

	return results
}

func init() {
	RegisterChecker("HTTP", func() Checker { return NewHTTPChecker() })
}
