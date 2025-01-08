package checkers

import (
	"context"
	"fmt"
	"net"
	"time"
)

type TCPChecker struct {
	timeout time.Duration
}

func NewTCPChecker() *TCPChecker {
	return &TCPChecker{
		timeout: 10 * time.Second,
	}
}

func (c *TCPChecker) Protocol() Protocol {
	return TCP
}

func (c *TCPChecker) Check(ctx context.Context, address string) CheckResult {
	start := time.Now()

	dialer := net.Dialer{
		Timeout: c.timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	elapsed := time.Since(start)

	if err != nil {
		return newFailedResult(elapsed, fmt.Errorf("TCP connection failed: %w", err))
	}
	defer conn.Close()

	return newSuccessResult(elapsed)
}
