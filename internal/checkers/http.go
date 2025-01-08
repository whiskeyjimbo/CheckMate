package checkers

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type HTTPChecker struct {
	client *http.Client
}

func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *HTTPChecker) Protocol() Protocol {
	return ProtocolHTTP
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

	return newSuccessResult(elapsed)
}
