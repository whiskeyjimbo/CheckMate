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

func (b *BaseChecker) checkHost(ctx context.Context, host string, checkFn func() error) HostCheckResult {
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

	if err := checkFn(); err != nil {
		result.Error = err
		result.Success = false
	}

	result.ResponseTime = time.Since(start)
	return result
}

func (b *BaseChecker) GetTimeout() time.Duration {
	return b.timeout
}

func (b *BaseChecker) SetTimeout(timeout time.Duration) error {
	var err error
	b.timeout, err = b.ValidateTimeout(timeout)
	if err != nil {
		return err
	}
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
