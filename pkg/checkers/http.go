package checkers

import (
	"fmt"
	"net/http"
	"time"
)

type HTTPChecker struct{}

func (c HTTPChecker) Check(address string) (success bool, responseTime int64, err error) {
	start := time.Now()
	resp, err := http.Get(fmt.Sprintf("http://%s", address))
	elapsed := time.Since(start).Microseconds()
	if err != nil {
		return false, elapsed, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, elapsed, nil
}
