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

type HTTPChecker struct {
	client *http.Client
}

func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *HTTPChecker) Protocol() Protocol {
	return HTTP
}

func (c *HTTPChecker) Check(ctx context.Context, hosts []string, port string) []HostCheckResult {
	results := make([]HostCheckResult, 0, len(hosts))

	for _, host := range hosts {
		address := fmt.Sprintf("%s:%s", host, port)
		url := fmt.Sprintf("http://%s", address)
		start := time.Now()

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			results = append(results, newHostResult(host, newFailedResult(time.Since(start), err)))
			continue
		}

		resp, err := c.client.Do(req)
		if err != nil {
			results = append(results, newHostResult(host, newFailedResult(time.Since(start), err)))
			continue
		}
		resp.Body.Close()

		results = append(results, newHostResult(host, newSuccessResult(time.Since(start))))
	}

	return results
}
