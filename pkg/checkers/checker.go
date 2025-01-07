package checkers

import (
	"context"
	"fmt"
	"time"
)

type Protocol string

const (
	ProtocolTCP  Protocol = "TCP"
	ProtocolHTTP Protocol = "HTTP"
	ProtocolSMTP Protocol = "SMTP"
	ProtocolDNS  Protocol = "DNS"
)

type CheckResult struct {
	Success      bool
	ResponseTime time.Duration
	Error        error
}

type Checker interface {
	Check(ctx context.Context, address string) CheckResult
	Protocol() Protocol
}

func NewChecker(protocol string) (Checker, error) {
	switch Protocol(protocol) {
	case ProtocolTCP:
		return NewTCPChecker(), nil
	case ProtocolHTTP:
		return NewHTTPChecker(), nil
	case ProtocolSMTP:
		return NewSMTPChecker(), nil
	case ProtocolDNS:
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
