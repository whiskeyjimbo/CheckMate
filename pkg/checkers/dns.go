package checkers

import (
	"net"
	"time"
)

type DNSChecker struct{}

func (c DNSChecker) Check(address string) (success bool, responseTime int64, err error) {
	start := time.Now()
	_, err = net.LookupHost(address)
	elapsed := time.Since(start).Microseconds()
	if err != nil {
		return false, elapsed, err
	}
	return true, elapsed, nil
}
