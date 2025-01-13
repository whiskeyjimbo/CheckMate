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
	"sync"
	"time"
)

const (
	tcpMinTimeout     = 1 * time.Second
	tcpMaxTimeout     = 10 * time.Second
	tcpDefaultTimeout = 5 * time.Second
)

type TCPChecker struct {
	BaseChecker
	mu sync.RWMutex
}

func NewTCPChecker() *TCPChecker {
	return &TCPChecker{
		BaseChecker: NewBaseChecker(TimeoutBounds{
			Min:     tcpMinTimeout,
			Max:     tcpMaxTimeout,
			Default: tcpDefaultTimeout,
		}),
	}
}

func (c *TCPChecker) Protocol() Protocol {
	return "TCP"
}

func (c *TCPChecker) Check(ctx context.Context, hosts []string, port string) []HostCheckResult {
	return c.BaseChecker.Check(ctx, hosts, port, c.checkTCP)
}

func (c *TCPChecker) checkTCP(ctx context.Context, host string, port string) (map[string]interface{}, error) {
	address := fmt.Sprintf("%s:%s", host, port)
	conn, err := net.DialTimeout("tcp", address, c.GetTimeout())
	if err != nil {
		return nil, fmt.Errorf("tcp connection failed: %w", err)
	}
	defer conn.Close()
	return nil, nil
}

func init() {
	RegisterChecker("TCP", func() Checker { return NewTCPChecker() })
}
