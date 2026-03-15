package executor

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// AssertionValidator 断言校验器
type AssertionValidator struct{}

// AssertionResult 断言结果
type AssertionResult struct {
	Pass    bool   `json:"pass"`
	Message string `json:"message"`
}

// Validate 执行断言校验
func (v *AssertionValidator) Validate(assertionType, assertionValue, output string) *AssertionResult {
	if assertionType == "" {
		return &AssertionResult{Pass: true, Message: "无断言规则，跳过校验"}
	}

	switch assertionType {
	case "gt":
		return v.validateGreaterThan(assertionValue, output)
	case "gte":
		return v.validateGreaterThanOrEqual(assertionValue, output)
	case "lt":
		return v.validateLessThan(assertionValue, output)
	case "lte":
		return v.validateLessThanOrEqual(assertionValue, output)
	case "eq":
		return v.validateEqual(assertionValue, output)
	case "contains":
		return v.validateContains(assertionValue, output)
	case "not_contains":
		return v.validateNotContains(assertionValue, output)
	case "regex":
		return v.validateRegex(assertionValue, output)
	case "not_regex":
		return v.validateNotRegex(assertionValue, output)
	default:
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("未知的断言类型: %s", assertionType)}
	}
}

func (v *AssertionValidator) validateGreaterThan(expected, actual string) *AssertionResult {
	expectedVal, err := strconv.ParseFloat(expected, 64)
	if err != nil {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("断言值解析失败: %v", err)}
	}

	actualVal, err := v.extractNumber(actual)
	if err != nil {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("输出值解析失败: %v", err)}
	}

	pass := actualVal > expectedVal
	return &AssertionResult{
		Pass:    pass,
		Message: fmt.Sprintf("实际值 %.2f %s 期望值 %.2f", actualVal, v.getCompareSymbol(pass, ">"), expectedVal),
	}
}

func (v *AssertionValidator) validateGreaterThanOrEqual(expected, actual string) *AssertionResult {
	expectedVal, err := strconv.ParseFloat(expected, 64)
	if err != nil {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("断言值解析失败: %v", err)}
	}

	actualVal, err := v.extractNumber(actual)
	if err != nil {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("输出值解析失败: %v", err)}
	}

	pass := actualVal >= expectedVal
	return &AssertionResult{
		Pass:    pass,
		Message: fmt.Sprintf("实际值 %.2f %s 期望值 %.2f", actualVal, v.getCompareSymbol(pass, ">="), expectedVal),
	}
}

func (v *AssertionValidator) validateLessThan(expected, actual string) *AssertionResult {
	expectedVal, err := strconv.ParseFloat(expected, 64)
	if err != nil {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("断言值解析失败: %v", err)}
	}

	actualVal, err := v.extractNumber(actual)
	if err != nil {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("输出值解析失败: %v", err)}
	}

	pass := actualVal < expectedVal
	return &AssertionResult{
		Pass:    pass,
		Message: fmt.Sprintf("实际值 %.2f %s 期望值 %.2f", actualVal, v.getCompareSymbol(pass, "<"), expectedVal),
	}
}

func (v *AssertionValidator) validateLessThanOrEqual(expected, actual string) *AssertionResult {
	expectedVal, err := strconv.ParseFloat(expected, 64)
	if err != nil {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("断言值解析失败: %v", err)}
	}

	actualVal, err := v.extractNumber(actual)
	if err != nil {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("输出值解析失败: %v", err)}
	}

	pass := actualVal <= expectedVal
	return &AssertionResult{
		Pass:    pass,
		Message: fmt.Sprintf("实际值 %.2f %s 期望值 %.2f", actualVal, v.getCompareSymbol(pass, "<="), expectedVal),
	}
}

func (v *AssertionValidator) validateEqual(expected, actual string) *AssertionResult {
	// 尝试数值比较
	expectedVal, err1 := strconv.ParseFloat(expected, 64)
	actualVal, err2 := v.extractNumber(actual)
	if err1 == nil && err2 == nil {
		pass := actualVal == expectedVal
		return &AssertionResult{
			Pass:    pass,
			Message: fmt.Sprintf("实际值 %.2f %s 期望值 %.2f", actualVal, v.getCompareSymbol(pass, "=="), expectedVal),
		}
	}

	// 字符串比较
	pass := strings.TrimSpace(actual) == strings.TrimSpace(expected)
	return &AssertionResult{
		Pass:    pass,
		Message: fmt.Sprintf("实际值 '%s' %s 期望值 '%s'", actual, v.getCompareSymbol(pass, "=="), expected),
	}
}

func (v *AssertionValidator) validateContains(expected, actual string) *AssertionResult {
	pass := strings.Contains(actual, expected)
	return &AssertionResult{
		Pass:    pass,
		Message: fmt.Sprintf("输出 %s 包含 '%s'", v.getContainsSymbol(pass), expected),
	}
}

func (v *AssertionValidator) validateNotContains(expected, actual string) *AssertionResult {
	pass := !strings.Contains(actual, expected)
	return &AssertionResult{
		Pass:    pass,
		Message: fmt.Sprintf("输出 %s 包含 '%s'", v.getContainsSymbol(!pass), expected),
	}
}

func (v *AssertionValidator) validateRegex(pattern, actual string) *AssertionResult {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("正则表达式编译失败: %v", err)}
	}

	pass := re.MatchString(actual)
	return &AssertionResult{
		Pass:    pass,
		Message: fmt.Sprintf("输出 %s 匹配正则 '%s'", v.getMatchSymbol(pass), pattern),
	}
}

func (v *AssertionValidator) validateNotRegex(pattern, actual string) *AssertionResult {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("正则表达式编译失败: %v", err)}
	}

	pass := !re.MatchString(actual)
	return &AssertionResult{
		Pass:    pass,
		Message: fmt.Sprintf("输出 %s 匹配正则 '%s'", v.getMatchSymbol(!pass), pattern),
	}
}

// extractNumber 从字符串中提取数字
func (v *AssertionValidator) extractNumber(s string) (float64, error) {
	// 移除空白字符
	s = strings.TrimSpace(s)

	// 尝试直接解析
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return val, nil
	}

	// 使用正则提取第一个数字（支持小数和负数）
	re := regexp.MustCompile(`-?\d+\.?\d*`)
	match := re.FindString(s)
	if match == "" {
		return 0, fmt.Errorf("未找到数字")
	}

	return strconv.ParseFloat(match, 64)
}

func (v *AssertionValidator) getCompareSymbol(pass bool, symbol string) string {
	if pass {
		return symbol
	}
	switch symbol {
	case ">":
		return "<="
	case ">=":
		return "<"
	case "<":
		return ">="
	case "<=":
		return ">"
	case "==":
		return "!="
	default:
		return "!=" + symbol
	}
}

func (v *AssertionValidator) getContainsSymbol(contains bool) string {
	if contains {
		return "✓"
	}
	return "✗"
}

func (v *AssertionValidator) getMatchSymbol(match bool) string {
	if match {
		return "✓"
	}
	return "✗"
}
