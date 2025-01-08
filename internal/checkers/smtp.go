package checkers

import (
	"context"
	"net/smtp"
	"time"
)

type SMTPChecker struct{}

func NewSMTPChecker() *SMTPChecker {
	return &SMTPChecker{}
}

func (c *SMTPChecker) Protocol() Protocol {
	return SMTP
}

func (c *SMTPChecker) Check(ctx context.Context, address string) CheckResult {
	start := time.Now()

	client, err := smtp.Dial(address)
	elapsed := time.Since(start)

	if err != nil {
		return newFailedResult(elapsed, err)
	}
	defer client.Close()

	return newSuccessResult(elapsed)
}
