package alert

import (
	"encoding/json"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
)

// EvaluateConditions 评估多条件阈值
func EvaluateConditions(value float64, conditionsJSON string) bool {
	if conditionsJSON == "" || conditionsJSON == "[]" || conditionsJSON == "null" {
		return false
	}

	var conditions []biz.ThresholdCondition
	if err := json.Unmarshal([]byte(conditionsJSON), &conditions); err != nil {
		return false
	}

	if len(conditions) == 0 {
		return false
	}

	// 评估第一个条件
	result := evaluateSingleCondition(value, conditions[0])

	// 处理后续条件
	for i := 1; i < len(conditions); i++ {
		prevLogic := conditions[i-1].Logic
		currentResult := evaluateSingleCondition(value, conditions[i])

		if prevLogic == "OR" {
			result = result || currentResult
		} else { // AND
			result = result && currentResult
		}
	}

	return result
}

func evaluateSingleCondition(value float64, cond biz.ThresholdCondition) bool {
	switch cond.Operator {
	case ">":
		return value > cond.Value
	case ">=":
		return value >= cond.Value
	case "<":
		return value < cond.Value
	case "<=":
		return value <= cond.Value
	case "==":
		return value == cond.Value
	case "!=":
		return value != cond.Value
	default:
		return false
	}
}
