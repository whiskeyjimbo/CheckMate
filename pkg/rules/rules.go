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

var (
	ErrConditionProcessing = errors.New("failed to process condition")
	ErrConditionCompile    = errors.New("failed to compile rule condition")
	ErrConditionEval       = errors.New("failed to evaluate rule condition")
	ErrNotBoolean          = errors.New("rule condition did not evaluate to a boolean")
	ErrEmptyRuleValue      = errors.New("rule name or condition are empty")
)

type RuleEnvironment struct {
	Downtime     int `expr:"downtime"`
	ResponseTime int `expr:"responseTime"`
}

type RuleResult struct {
	Rule      Rule
	Satisfied bool
	EvalTime  time.Time
	Error     error
}

// Main function to evaluate a rule
func EvaluateRule(rule Rule, downtime time.Duration, responseTime time.Duration) RuleResult {
	result := RuleResult{
		Rule:     rule,
		EvalTime: time.Now(),
	}

	if err := rule.Validate(); err != nil {
		result.Error = fmt.Errorf("invalid rule: %w", err)
		return result
	}

	satisfied, err := evaluateRuleInternal(rule, downtime, responseTime)
	result.Satisfied = satisfied
	result.Error = err
	return result
}

func evaluateRuleInternal(rule Rule, downtime time.Duration, responseTime time.Duration) (bool, error) {
	env := createExprEnv(downtime, responseTime)

	program, err := expr.Compile(rule.Condition, expr.Env(env))
	if err != nil {
		processedCondition, procErr := processCondition(rule.Condition, downtime, responseTime)
		if procErr != nil {
			return false, procErr
		}

		program, err = expr.Compile(processedCondition, expr.Env(env))
		if err != nil {
			return false, fmt.Errorf("rule '%s': %w: %v", rule.Name, ErrConditionCompile, err)
		}
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("rule '%s': %w: %v", rule.Name, ErrConditionEval, err)
	}

	result, ok := output.(bool)
	if !ok {
		return false, ErrNotBoolean
	}

	return result, nil
}

func (r Rule) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("rule name: '%s' cannot be empty w: %v", r.Name, ErrEmptyRuleValue)
	}
	if r.Condition == "" {
		return fmt.Errorf("rule condition: '%s' cannot be empty w: %v", r.Condition, ErrEmptyRuleValue)
	}
	return nil
}

func createExprEnv(downtime, responseTime time.Duration) RuleEnvironment {
	return RuleEnvironment{
		Downtime:     int(downtime.Seconds()),
		ResponseTime: int(responseTime.Milliseconds()),
	}
}

func processCondition(condition string, downtime, responseTime time.Duration) (string, error) {
	condition = strings.ReplaceAll(condition, "${downtime}", fmt.Sprintf("%d", int(downtime.Seconds())))
	condition = strings.ReplaceAll(condition, "${responseTime}", fmt.Sprintf("%d", int(responseTime.Milliseconds())))

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
