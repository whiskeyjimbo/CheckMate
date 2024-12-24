package checkers

import (
	"net"
	"time"
)

type DNSChecker struct {
	protocol Protocol
}

func (c *DNSChecker) Protocol() Protocol {
	return c.protocol
}

func (c *DNSChecker) Check(address string) CheckResult {
	start := time.Now()
	_, err := net.LookupHost(address)
	elapsed := time.Since(start)

	if err != nil {
		return CheckResult{
			Success:      false,
			ResponseTime: elapsed,
			Error:        err,
		}
	}

	return CheckResult{
		Success:      true,
		ResponseTime: elapsed,
		Error:        nil,
	}
}
