package checkers

import (
	"net"
	"time"
)

type TCPChecker struct {
	protocol Protocol
}

func (c *TCPChecker) Protocol() Protocol {
	return c.protocol
}

func (c *TCPChecker) Check(address string) CheckResult {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
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
