package checkers

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type HTTPChecker struct {
	protocol Protocol
	client   *http.Client
}

func NewHTTPChecker(protocol Protocol) *HTTPChecker {
	return &HTTPChecker{
		protocol: protocol,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *HTTPChecker) Protocol() Protocol {
	return c.protocol
}

func (c *HTTPChecker) Check(ctx context.Context, address string) CheckResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s", address), nil)
	if err != nil {
		return newFailedResult(time.Since(start), err)
	}

	resp, err := c.client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return newFailedResult(elapsed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return newFailedResult(elapsed, fmt.Errorf("HTTP status code: %d", resp.StatusCode))
	}

	return CheckResult{
		Success:      true,
		ResponseTime: elapsed,
		Error:        nil,
	}
}

func newFailedResult(duration time.Duration, err error) CheckResult {
	return CheckResult{
		Success:      false,
		ResponseTime: duration,
		Error:        err,
	}
}
