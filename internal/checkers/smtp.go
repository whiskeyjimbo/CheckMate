package checkers

import (
	"context"
	"fmt"
	"net"
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

func (c *SMTPChecker) Check(ctx context.Context, hosts []string, port string) []HostCheckResult {
	results := make([]HostCheckResult, 0, len(hosts))

	for _, host := range hosts {
		address := fmt.Sprintf("%s:%s", host, port)
		start := time.Now()

		d := net.Dialer{Timeout: 10 * time.Second}
		conn, err := d.DialContext(ctx, "tcp", address)
		if err != nil {
			results = append(results, newHostResult(host, newFailedResult(time.Since(start), err)))
			continue
		}

		client, err := smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			results = append(results, newHostResult(host, newFailedResult(time.Since(start), err)))
			continue
		}
		client.Close()

		results = append(results, newHostResult(host, newSuccessResult(time.Since(start))))
	}

	return results
}
