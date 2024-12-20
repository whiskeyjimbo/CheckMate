package checkers

import (
	"net/smtp"
	"time"
)

type SMTPChecker struct{}

func (c SMTPChecker) Check(address string) (success bool, responseTime int64, err error) {
	start := time.Now()
	smtp, err := smtp.Dial(address)
	elapsed := time.Since(start).Microseconds()
	if err != nil {
		return false, elapsed, err
	}
	defer smtp.Close()
	return true, elapsed, nil
}
