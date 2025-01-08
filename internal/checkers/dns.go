package checkers

import (
	"context"
	"net"
	"time"
)

type DNSChecker struct{}

func NewDNSChecker() *DNSChecker {
	return &DNSChecker{}
}

func (c *DNSChecker) Protocol() Protocol {
	return DNS
}

func (c *DNSChecker) Check(ctx context.Context, hosts []string, port string) []HostCheckResult {
	results := make([]HostCheckResult, 0, len(hosts))

	for _, host := range hosts {
		start := time.Now()

		_, err := net.DefaultResolver.LookupHost(ctx, host)
		if err != nil {
			results = append(results, newHostResult(host, newFailedResult(time.Since(start), err)))
			continue
		}

		results = append(results, newHostResult(host, newSuccessResult(time.Since(start))))
	}

	return results
}
