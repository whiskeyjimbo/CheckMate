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
	"errors"
	"sync"
	"time"
)

const (
	GlobalMinTimeout     = 2 * time.Second
	GlobalMaxTimeout     = 20 * time.Second
	GlobalDefaultTimeout = 10 * time.Second
)

type CheckResult struct {
	Error        error
	Metadata     map[string]interface{}
	ResponseTime time.Duration
	Success      bool
}

type HostCheckResult struct {
	Host string
	CheckResult
}

type Checker interface {
	Protocol() Protocol
	Check(ctx context.Context, hosts []string, port string) []HostCheckResult
	GetTimeout() time.Duration
	SetTimeout(timeout time.Duration) error
}

type TimeoutBounds struct {
	Min     time.Duration
	Max     time.Duration
	Default time.Duration
}

type BaseChecker struct {
	timeout time.Duration
	bounds  TimeoutBounds
	mu      sync.RWMutex
}

func NewBaseChecker(bounds TimeoutBounds) BaseChecker {
	if bounds.Min == 0 {
		bounds.Min = GlobalMinTimeout
	}
	if bounds.Max == 0 {
		bounds.Max = GlobalMaxTimeout
	}
	if bounds.Default == 0 {
		bounds.Default = GlobalDefaultTimeout
	}

	return BaseChecker{
		timeout: bounds.Default,
		bounds:  bounds,
	}
}

func (b *BaseChecker) checkHost(ctx context.Context, host string, checkFn func() (map[string]interface{}, error)) HostCheckResult {
	start := time.Now()
	result := HostCheckResult{
		Host: host,
		CheckResult: CheckResult{
			Success: true,
		},
	}

	if err := ctx.Err(); err != nil {
		result.Error = err
		result.Success = false
		return result
	}

	metadata, err := checkFn()
	if err != nil {
		result.Error = err
		result.Success = false
	}
	result.Metadata = metadata
	result.ResponseTime = time.Since(start)
	return result
}

func (b *BaseChecker) GetTimeout() time.Duration {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.timeout
}

func (b *BaseChecker) SetTimeout(timeout time.Duration) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	validTimeout, err := b.ValidateTimeout(timeout)
	if err != nil {
		return err
	}
	b.timeout = validTimeout
	return nil
}

func (b *BaseChecker) GetTimeoutBounds() TimeoutBounds {
	return b.bounds
}

func (b *BaseChecker) ValidateTimeout(timeout time.Duration) (time.Duration, error) {
	if timeout == 0 {
		return b.bounds.Default, nil
	}
	if timeout < b.bounds.Min {
		return b.bounds.Min, errors.New("timeout is less than the minimum allowed")
	}
	if timeout > b.bounds.Max {
		return b.bounds.Max, errors.New("timeout is greater than the maximum allowed")
	}
	return timeout, nil
}

func (b *BaseChecker) Check(ctx context.Context, hosts []string, port string, checkFn func(ctx context.Context, host string, port string) (map[string]interface{}, error)) []HostCheckResult {
	return b.parallelCheck(ctx, hosts, port, checkFn)
}

func (b *BaseChecker) parallelCheck(ctx context.Context, hosts []string, port string, checkFn func(ctx context.Context, host string, port string) (map[string]interface{}, error)) []HostCheckResult {
	checkCtx, cancel := context.WithTimeout(ctx, b.GetTimeout())
	defer cancel()

	results := make([]HostCheckResult, len(hosts))
	for i, host := range hosts {
		results[i].Host = host
	}

	var wg sync.WaitGroup
	wg.Add(len(hosts))

	for i, host := range hosts {
		go func(index int, host string) {
			defer wg.Done()
			results[index] = b.checkHost(checkCtx, host, func() (map[string]interface{}, error) {
				return checkFn(checkCtx, host, port)
			})
		}(i, host)
	}

	wg.Wait()
	return results
}

type CheckError struct {
	err      error
	metadata map[string]interface{}
}

func (e *CheckError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return ""
}

func (e *CheckError) Metadata() map[string]interface{} {
	return e.metadata
}
