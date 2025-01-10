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
	"net"
	"time"
)

const defaultTCPTimeout = 10 * time.Second

type TCPChecker struct {
	BaseChecker
}

func NewTCPChecker() *TCPChecker {
	return &TCPChecker{
		BaseChecker: BaseChecker{
			timeout: defaultTCPTimeout,
		},
	}
}

func (c *TCPChecker) Protocol() Protocol {
	return "TCP"
}

func (c *TCPChecker) Check(ctx context.Context, hosts []string, port string) []HostCheckResult {
	results := make([]HostCheckResult, 0, len(hosts))

	for _, host := range hosts {
		address := fmt.Sprintf("%s:%s", host, port)
		result := c.checkHost(ctx, host, func() error {
			conn, err := net.DialTimeout("tcp", address, c.timeout)
			if err != nil {
				return fmt.Errorf("tcp connection failed: %w", err)
			}
			defer conn.Close()
			return nil
		})
		results = append(results, result)
	}

	return results
}

func init() {
	RegisterChecker("TCP", func() Checker { return NewTCPChecker() })
}
