package checkers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

type HTTPSChecker struct {
	client *http.Client
}

type CertInfo struct {
	ExpiresAt time.Time
	IssuedBy  string
	IssuedFor []string
}

type HTTPSResult struct {
	CheckResult
	Certificate *CertInfo
}

func NewHTTPSChecker() *HTTPSChecker {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,            // Enforce certificate validation
			MinVersion:         tls.VersionTLS12, // Enforce minimum TLS version G402
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	return &HTTPSChecker{client: client}
}

func (c *HTTPSChecker) Check(ctx context.Context, hosts []string, port string) []HostCheckResult {
	results := make([]HostCheckResult, 0, len(hosts))

	for _, host := range hosts {
		address := fmt.Sprintf("%s:%s", host, port)
		url := fmt.Sprintf("https://%s", address)
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

		// Get certificate information
		certInfo := c.getCertificateInfo(resp.TLS)
		result := newSuccessResult(time.Since(start))
		result.Metadata = map[string]interface{}{"certificate": certInfo}

		resp.Body.Close()
		results = append(results, newHostResult(host, result))
	}

	return results
}

func (c *HTTPSChecker) getCertificateInfo(tlsState *tls.ConnectionState) *CertInfo {
	if tlsState == nil || len(tlsState.PeerCertificates) == 0 {
		return nil
	}

	cert := tlsState.PeerCertificates[0]
	return &CertInfo{
		ExpiresAt: cert.NotAfter,
		IssuedBy:  cert.Issuer.CommonName,
		IssuedFor: cert.DNSNames,
	}
}

func (c *HTTPSChecker) Protocol() Protocol {
	return HTTPS
}
