package checkers

import (
	"fmt"
	"net/http"
	"time"
)

type HTTPChecker struct {
	protocol Protocol
}

func (c *HTTPChecker) Protocol() Protocol {
	return c.protocol
}

func (c *HTTPChecker) Check(address string) CheckResult {
	start := time.Now()
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Get(fmt.Sprintf("http://%s", address))
	elapsed := time.Since(start)

	if err != nil {
		return CheckResult{
			Success:      false,
			ResponseTime: elapsed,
			Error:        err,
		}
	}
	defer resp.Body.Close()

	return CheckResult{
		Success:      resp.StatusCode == http.StatusOK,
		ResponseTime: elapsed,
		Error:        nil,
	}
}
