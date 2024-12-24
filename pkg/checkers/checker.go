package checkers

import (
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
	Check(address string) CheckResult
	Protocol() Protocol
}

func NewChecker(protocol string) (Checker, error) {
	switch Protocol(protocol) {
	case ProtocolTCP:
		return &TCPChecker{protocol: ProtocolTCP}, nil
	case ProtocolHTTP:
		return &HTTPChecker{protocol: ProtocolHTTP}, nil
	case ProtocolSMTP:
		return &SMTPChecker{protocol: ProtocolSMTP}, nil
	case ProtocolDNS:
		return &DNSChecker{protocol: ProtocolDNS}, nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
