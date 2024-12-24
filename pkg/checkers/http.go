package checkers

import (
	"context"
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

func (c *HTTPChecker) Check(ctx context.Context, address string) CheckResult {
	start := time.Now()
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s", address), nil)
	if err != nil {
		return CheckResult{
			Success:      false,
			ResponseTime: time.Since(start),
			Error:        err,
		}
	}
	
	resp, err := client.Do(req)
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
