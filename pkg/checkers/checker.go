package checkers

import "fmt"

type Checker interface {
	Check(address string) (success bool, responseTime int64, err error)
}

func NewChecker(protocol string) (Checker, error) {
	switch protocol {
	case "TCP":
		return &TCPChecker{}, nil
	case "HTTP":
		return &HTTPChecker{}, nil
	case "SMTP":
		return &SMTPChecker{}, nil
	case "DNS":
		return &DNSChecker{}, nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
