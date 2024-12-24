package checkers

import (
	"net/smtp"
	"time"
	"context"
)

type SMTPChecker struct {
	protocol Protocol
}

func (c *SMTPChecker) Protocol() Protocol {
	return c.protocol
}

func (c *SMTPChecker) Check(ctx context.Context, address string) CheckResult {
	_ = ctx // TODO: figure out context in smtp
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
