package checkers

import (
	"context"
	"fmt"
	"time"
)

type Protocol string

const (
	TCP   Protocol = "TCP"
	HTTP  Protocol = "HTTP"
	HTTPS Protocol = "HTTPS"
	SMTP  Protocol = "SMTP"
	DNS   Protocol = "DNS"
)

type CheckResult struct {
	Error        error
	Metadata     map[string]interface{}
	ResponseTime time.Duration
	Success      bool
}

type HostCheckResult struct {
	Host string
	CheckResult
}

type Checker interface {
	Check(ctx context.Context, hosts []string, port string) []HostCheckResult
	Protocol() Protocol
}

func NewChecker(protocol Protocol) (Checker, error) {
	switch protocol {
	case TCP:
		return NewTCPChecker(), nil
	case HTTP:
		return NewHTTPChecker(), nil
	case HTTPS:
		return NewHTTPSChecker(), nil
	case SMTP:
		return NewSMTPChecker(), nil
	case DNS:
		return NewDNSChecker(), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

func newFailedResult(duration time.Duration, err error) CheckResult {
	return CheckResult{
		Success:      false,
		ResponseTime: duration,
		Error:        err,
	}
}

func newSuccessResult(duration time.Duration) CheckResult {
	return CheckResult{
		Success:      true,
		ResponseTime: duration,
		Error:        nil,
	}
}

func newHostResult(host string, result CheckResult) HostCheckResult {
	return HostCheckResult{
		Host:        host,
		CheckResult: result,
	}
}
