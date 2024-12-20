package checkers

import (
	"net"
	"time"
)

type TCPChecker struct{}

func (c TCPChecker) Check(address string) (success bool, responseTime int64, err error) {
	start := time.Now()
	conn, err := net.Dial("tcp", address)
	elapsed := time.Since(start).Microseconds()
	if err != nil {
		return false, elapsed, err
	}
	defer conn.Close()
	return true, elapsed, nil
}
