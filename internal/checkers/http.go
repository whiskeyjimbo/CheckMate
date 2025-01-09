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
	return HTTP
}

func (c *HTTPChecker) Check(ctx context.Context, hosts []string, port string) []HostCheckResult {
	results := make([]HostCheckResult, 0, len(hosts))

	for _, host := range hosts {
		address := fmt.Sprintf("%s:%s", host, port)
		url := fmt.Sprintf("http://%s", address)
		start := time.Now()

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			results = append(results, newHostResult(host, newFailedResult(time.Since(start), err)))
			continue
		}

		resp, err := c.client.Do(req)
		if err != nil {
			results = append(results, newHostResult(host, newFailedResult(time.Since(start), err)))
			continue
		}
		resp.Body.Close()

		results = append(results, newHostResult(host, newSuccessResult(time.Since(start))))
	}

	return results
}
