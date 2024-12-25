package rules

import (
	"fmt"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"errors"
)

type Rule struct {
	Name      string `yaml:"name"`
	Condition string `yaml:"condition"`
}

var (
	ErrConditionProcessing = errors.New("failed to process condition")
	ErrConditionCompile    = errors.New("failed to compile rule condition")
	ErrConditionEval      = errors.New("failed to evaluate rule condition")
	ErrNotBoolean         = errors.New("rule condition did not evaluate to a boolean")
)

// Main function to evaluate a rule
func EvaluateRule(rule Rule, downtime time.Duration, responseTime time.Duration) (bool, error) {
	env := createExprEnv(downtime, responseTime)
	program, err := expr.Compile(rule.Condition, expr.Env(env))
	if err != nil {
		processedCondition, err := processCondition(rule.Condition, downtime, responseTime)
		if err != nil {
			return false, fmt.Errorf("%w: %v", ErrConditionProcessing, err)
		}
		program, err = expr.Compile(processedCondition, expr.Env(env))
		if err != nil {
			return false, fmt.Errorf("%w: %v", ErrConditionCompile, err)
		}
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrConditionEval, err)
	}

	result, ok := output.(bool)
	if !ok {
		return false, ErrNotBoolean
	}

	return result, nil
}

func createExprEnv(downtime, responseTime time.Duration) map[string]interface{} {
	return map[string]interface{}{
		"downtime":     timeDurationToSeconds(downtime),
		"responseTime": timeDurationToSeconds(responseTime),
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
