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

package rules

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/expr-lang/expr"
)

type RuleType string

const (
	StandardRule RuleType = "standard"
	CertRule     RuleType = "cert"
)

type Rule struct {
	Name            string   `yaml:"name"`
	Type            RuleType `yaml:"type"`
	Condition       string   `yaml:"condition,omitempty"`
	Tags            []string `yaml:"tags"`
	Notifications   []string `yaml:"notifications"`
	MinDaysValidity int      `yaml:"minDaysValidity,omitempty"`
}

type RuleResult struct {
	Error     error
	Message   string
	Satisfied bool
}

var (
	ErrEmptyCondition = errors.New("rule condition cannot be empty")
	ErrInvalidSyntax  = errors.New("invalid rule syntax")
)

type EvaluationParams struct {
	CertExpiryTime time.Time
	Downtime       time.Duration
	ResponseTime   time.Duration
}

func (r Rule) Validate() error {
	if r.Type == "" {
		return fmt.Errorf("rule type must be specified")
	}
	switch r.Type {
	case StandardRule, CertRule:
		return nil
	default:
		return fmt.Errorf("invalid rule type: %s", r.Type)
	}
}

func EvaluateRule(rule Rule, params EvaluationParams) RuleResult {
	if err := rule.Validate(); err != nil {
		return RuleResult{Error: err}
	}

	switch rule.Type {
	case StandardRule:
		return evaluateStandardRule(rule, params.Downtime, params.ResponseTime)
	case CertRule:
		return evaluateCertRule(rule, params.CertExpiryTime)
	}
	return RuleResult{Error: fmt.Errorf("unsupported rule type: %s", rule.Type)}
}

func evaluateStandardRule(rule Rule, downtime, responseTime time.Duration) RuleResult {
	if rule.Condition == "" {
		return RuleResult{Error: ErrEmptyCondition}
	}

	env := map[string]interface{}{
		"downtime":     timeDurationToSeconds(downtime),
		"responseTime": timeDurationToSeconds(responseTime),
	}

	condition, err := normalizeCondition(rule.Condition)
	if err != nil {
		return RuleResult{Error: fmt.Errorf("failed to normalize condition: %w", err)}
	}

	program, err := expr.Compile(condition, expr.Env(env))
	if err != nil {
		return RuleResult{Error: fmt.Errorf("%w: %v", ErrInvalidSyntax, err)}
	}

	result, err := expr.Run(program, env)
	if err != nil {
		return RuleResult{Error: fmt.Errorf("rule evaluation failed: %w", err)}
	}

	satisfied, ok := result.(bool)
	if !ok {
		return RuleResult{Error: fmt.Errorf("rule must evaluate to boolean, got %T", result)}
	}

	return RuleResult{
		Satisfied: satisfied,
		Error:     nil,
	}
}

func normalizeCondition(condition string) (string, error) {
	words := strings.Split(condition, " ")
	for i, word := range words {
		dur, err := time.ParseDuration(word)
		if err == nil {
			words[i] = fmt.Sprintf("%d", int(dur.Seconds()))
		}
	}
	return strings.Join(words, " "), nil
}

func timeDurationToSeconds(d time.Duration) int {
	return int(d.Seconds())
}

func (r Rule) GetNotificationTypes() []string {
	return r.Notifications
}
