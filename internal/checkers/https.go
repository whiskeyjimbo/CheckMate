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
			InsecureSkipVerify: false, // Enforce certificate validation
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	return &HTTPSChecker{client: client}
}

func (c *HTTPSChecker) Check(ctx context.Context, address string) CheckResult {
	start := time.Now()
	url := fmt.Sprintf("https://%s", address)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return CheckResult{Success: false, Error: err}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return CheckResult{Success: false, Error: err}
	}
	defer resp.Body.Close()

	// Get certificate information
	certInfo := c.getCertificateInfo(resp.TLS)

	result := CheckResult{
		Success:      resp.StatusCode < 400,
		ResponseTime: time.Since(start),
		Error:        nil,
		Metadata: map[string]interface{}{
			"certificate": certInfo,
		},
	}

	return result
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
