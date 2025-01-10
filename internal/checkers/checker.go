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
	"time"
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

// Checker defines the interface for all protocol checkers
type Checker interface {
	Protocol() Protocol
	Check(ctx context.Context, hosts []string, port string) []HostCheckResult
}

// BaseChecker provides common functionality for all checkers
type BaseChecker struct {
	timeout time.Duration
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
