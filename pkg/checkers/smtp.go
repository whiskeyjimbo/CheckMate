package checkers

import (
	"net/smtp"
	"time"
)

type SMTPChecker struct {
	protocol Protocol
}

func (c *SMTPChecker) Protocol() Protocol {
	return c.protocol
}

func (c *SMTPChecker) Check(address string) CheckResult {
	start := time.Now()
	client, err := smtp.Dial(address)
	elapsed := time.Since(start)

	if err != nil {
		return CheckResult{
			Success:      false,
			ResponseTime: elapsed,
			Error:        err,
		}
	}
	defer client.Close()

	return CheckResult{
		Success:      true,
		ResponseTime: elapsed,
		Error:        nil,
	}
}
