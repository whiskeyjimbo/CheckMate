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

func (c *TCPChecker) Check(ctx context.Context, hosts []string, port string) []HostCheckResult {
	results := make([]HostCheckResult, 0, len(hosts))

	for _, host := range hosts {
		address := fmt.Sprintf("%s:%s", host, port)
		start := time.Now()

		d := net.Dialer{}
		conn, err := d.DialContext(ctx, "tcp", address)
		if err != nil {
			results = append(results, newHostResult(host, newFailedResult(time.Since(start), err)))
			continue
		}
		conn.Close()

		results = append(results, newHostResult(host, newSuccessResult(time.Since(start))))
	}

	return results
}
