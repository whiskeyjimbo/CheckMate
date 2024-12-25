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

func EvaluateRule(rule Rule, downtime time.Duration, responseTime time.Duration) (bool, error) {
	// prob convert the parsedduration to seconds and any rule to seconds to do the comparison
	env := map[string]interface{}{
		"downtime":     timeDurationToSeconds(downtime),
		"responseTime": timeDurationToSeconds(responseTime),
	}
	program, err := expr.Compile(rule.Condition, expr.Env(env))
	if err != nil {
		condition := strings.ReplaceAll(rule.Condition, "${downtime}", fmt.Sprintf("%d", timeDurationToSeconds(downtime)))
		condition = strings.ReplaceAll(condition, "${responseTime}", fmt.Sprintf("%d", timeDurationToSeconds(responseTime)))
		words := strings.Split(condition, " ")
		for i, word := range words {
			dur, err := time.ParseDuration(word)
			if err == nil {
				words[i] = fmt.Sprintf("%d", int(dur.Seconds()))
			}
		}
		condition = strings.Join(words, " ")
		program, err = expr.Compile(condition, expr.Env(env))
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

func timeDurationToSeconds(d time.Duration) int {
	return int(d.Seconds())
}
