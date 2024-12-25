package rules

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/expr-lang/expr"
)

type Rule struct {
	Name      string `yaml:"name"`
	Condition string `yaml:"condition"`
}

var (
	ErrConditionProcessing = errors.New("failed to process condition")
	ErrConditionCompile    = errors.New("failed to compile rule condition")
	ErrConditionEval       = errors.New("failed to evaluate rule condition")
	ErrNotBoolean          = errors.New("rule condition did not evaluate to a boolean")
)

type RuleEnvironment struct {
	Downtime     int `expr:"downtime"`
	ResponseTime int `expr:"responseTime"`
}

// Main function to evaluate a rule
func EvaluateRule(rule Rule, downtime time.Duration, responseTime time.Duration) (bool, error) {
	env := createExprEnv(downtime, responseTime)

	program, err := expr.Compile(rule.Condition, expr.Env(env))
	if err != nil {
		processedCondition, procErr := processCondition(rule.Condition, downtime, responseTime)
		if procErr != nil {
			return false, procErr // No need to wrap as processCondition adds context
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

func createExprEnv(downtime, responseTime time.Duration) RuleEnvironment {
	return RuleEnvironment{
		Downtime:     timeDurationToSeconds(downtime),
		ResponseTime: timeDurationToSeconds(responseTime),
	}
}

func processCondition(condition string, downtime, responseTime time.Duration) (string, error) {
	condition = strings.ReplaceAll(condition, "${downtime}", fmt.Sprintf("%d", timeDurationToSeconds(downtime)))
	condition = strings.ReplaceAll(condition, "${responseTime}", fmt.Sprintf("%d", timeDurationToSeconds(responseTime)))

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
