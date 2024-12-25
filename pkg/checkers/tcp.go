package checkers

import (
	"context"
	"net"
	"time"
)

type TCPChecker struct {
	protocol Protocol
}

func (c *TCPChecker) Protocol() Protocol {
	return c.protocol
}

func (c *TCPChecker) Check(ctx context.Context, address string) CheckResult {
	start := time.Now()

	dialer := net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	elapsed := time.Since(start)

	if err != nil {
		return CheckResult{
			Success:      false,
			ResponseTime: elapsed,
			Error:        err,
		}
	}
	defer conn.Close()

	return CheckResult{
		Success:      true,
		ResponseTime: elapsed,
		Error:        nil,
	}
}
