// Copyright (C) 2025 Jeff Rose
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package checkers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultHTTPSTimeout = 10 * time.Second
)

type CertInfo struct {
	ExpiresAt time.Time
	IssuedBy  string
}

type HTTPSChecker struct {
	BaseChecker
	client *http.Client
}

func NewHTTPSChecker() *HTTPSChecker {
	return &HTTPSChecker{
		BaseChecker: BaseChecker{
			timeout: defaultHTTPSTimeout,
		},
		client: &http.Client{
			Timeout: defaultHTTPSTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{},
			},
		},
	}
}

func (c *HTTPSChecker) Protocol() Protocol {
	return "HTTPS"
}

func (c *HTTPSChecker) Check(ctx context.Context, hosts []string, port string) []HostCheckResult {
	results := make([]HostCheckResult, 0, len(hosts))

	for _, host := range hosts {
		url := fmt.Sprintf("https://%s:%s", host, port)
		var certInfo *CertInfo
		result := c.checkHost(ctx, host, func() error {
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			resp, err := c.client.Do(req)
			if err != nil {
				return fmt.Errorf("https request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				return fmt.Errorf("https status error: %d", resp.StatusCode)
			}

			if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
				cert := resp.TLS.PeerCertificates[0]
				certInfo = &CertInfo{
					ExpiresAt: cert.NotAfter,
					IssuedBy:  cert.Issuer.CommonName,
				}
			}
			return nil
		})
		if certInfo != nil {
			result.Metadata = map[string]interface{}{
				"cert_info": certInfo,
			}
		}
		results = append(results, result)
	}

	return results
}

func init() {
	RegisterChecker("HTTPS", func() Checker { return NewHTTPSChecker() })
}
