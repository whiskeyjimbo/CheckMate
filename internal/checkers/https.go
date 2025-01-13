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
	"sync"
	"time"
)

const (
	httpsMinTimeout     = 3 * time.Second // Slightly longer than HTTP due to TLS handshake
	httpsMaxTimeout     = 20 * time.Second
	httpsDefaultTimeout = 12 * time.Second
)

type CertInfo struct {
	ExpiresAt time.Time
	IssuedBy  string
}

type HTTPSChecker struct {
	BaseChecker
	client     *http.Client
	mu         sync.RWMutex
	verifyCert bool
}

func NewHTTPSChecker() *HTTPSChecker {
	return &HTTPSChecker{
		BaseChecker: NewBaseChecker(TimeoutBounds{
			Min:     httpsMinTimeout,
			Max:     httpsMaxTimeout,
			Default: httpsDefaultTimeout,
		}),
		client: &http.Client{
			Timeout: httpsDefaultTimeout,
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
	return c.BaseChecker.Check(ctx, hosts, port, c.checkHTTPS)
}

func (c *HTTPSChecker) checkHTTPS(ctx context.Context, host string, port string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://%s:%s", host, port)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := c.client
	if !c.verifyCert {
		client = &http.Client{
			Timeout: c.GetTimeout(),
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("https request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("https status error: %d", resp.StatusCode)
	}

	// Collect cert info if available
	var metadata map[string]interface{}
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		metadata = map[string]interface{}{
			"cert_info": &CertInfo{
				ExpiresAt: cert.NotAfter,
				IssuedBy:  cert.Issuer.CommonName,
			},
		}
	}

	return metadata, nil
}

func init() {
	RegisterChecker("HTTPS", func() Checker { return NewHTTPSChecker() })
}
