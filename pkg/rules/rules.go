package rules

import (
	"fmt"
	"strings"
	"time"

	"github.com/expr-lang/expr"
)

type Rule struct {
	Name      string `yaml:"name"`
	Condition string `yaml:"condition"`
}

// Main function to evaluate a rule
func EvaluateRule(rule Rule, downtime time.Duration, responseTime time.Duration) (bool, error) {
	env := createExprEnv(downtime, responseTime)
	program, err := expr.Compile(rule.Condition, expr.Env(env))
	if err != nil {
		processedCondition, err := processCondition(rule.Condition, downtime, responseTime)
		if err != nil {
			return false, fmt.Errorf("failed to process condition: %w", err)
		}
		program, err = expr.Compile(processedCondition, expr.Env(env))
		if err != nil {
			return false, fmt.Errorf("failed to compile rule condition: %w", err)
		}
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate rule condition: %w", err)
	}

	result, ok := output.(bool)
	if !ok {
		return false, fmt.Errorf("rule condition did not evaluate to a boolean")
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
