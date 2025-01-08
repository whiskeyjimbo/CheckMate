package checkers

import (
	"context"
	"fmt"
	"net"
	"time"
)

type DNSChecker struct{}

func NewDNSChecker() *DNSChecker {
	return &DNSChecker{}
}

func (c *DNSChecker) Protocol() Protocol {
	return ProtocolDNS
}

func (c *DNSChecker) Check(ctx context.Context, address string) CheckResult {
	start := time.Now()

	resolver := net.Resolver{}
	_, err := resolver.LookupHost(ctx, address)
	elapsed := time.Since(start)

	if err != nil {
		return newFailedResult(elapsed, fmt.Errorf("DNS lookup failed: %w", err))
	}

	return newSuccessResult(elapsed)
}
