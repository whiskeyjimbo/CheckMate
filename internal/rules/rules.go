package rules

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/expr-lang/expr"
)

type Rule struct {
	Name          string   `yaml:"name"`
	Condition     string   `yaml:"condition"`
	Tags          []string `yaml:"tags"`
	Notifications []string `yaml:"notifications"`
}

type RuleResult struct {
	Satisfied bool
	Message   string
	Error     error
}

var (
	ErrEmptyCondition = errors.New("rule condition cannot be empty")
	ErrInvalidSyntax  = errors.New("invalid rule syntax")
)

func EvaluateRule(rule Rule, downtime, responseTime time.Duration) RuleResult {
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
