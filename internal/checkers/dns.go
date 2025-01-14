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
	dnsMinTimeout     = 500 * time.Millisecond
	dnsMaxTimeout     = 5 * time.Second
	dnsDefaultTimeout = 2 * time.Second
)

type DNSChecker struct {
	BaseChecker
	resolver *net.Resolver
	mu       sync.RWMutex
}

func NewDNSChecker() *DNSChecker {
	return &DNSChecker{
		BaseChecker: NewBaseChecker(TimeoutBounds{
			Min:     dnsMinTimeout,
			Max:     dnsMaxTimeout,
			Default: dnsDefaultTimeout,
		}),
		resolver: net.DefaultResolver,
	}
}

func (c *DNSChecker) Protocol() Protocol {
	return "DNS"
}

func (c *DNSChecker) Check(ctx context.Context, hosts []string, port string) []HostCheckResult {
	return c.BaseChecker.Check(ctx, hosts, port, c.checkDNS)
}

func (c *DNSChecker) checkDNS(ctx context.Context, host string, port string) (map[string]interface{}, error) {
	lookupCtx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()

	ips, err := c.resolver.LookupIP(lookupCtx, "ip4", host)
	if err != nil {
		return nil, fmt.Errorf("dns lookup failed: %w", err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP addresses found for host")
	}

	return map[string]interface{}{
		"ips": ips,
	}, nil
}

func init() {
	RegisterChecker("DNS", func() Checker { return NewDNSChecker() })
}
