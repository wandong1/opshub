package executor

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// 断言类型常量
const (
	// 原有类型（用于 command/script）
	AssertionTypeGT          = "gt"
	AssertionTypeGTE         = "gte"
	AssertionTypeLT          = "lt"
	AssertionTypeLTE         = "lte"
	AssertionTypeEQ          = "eq"
	AssertionTypeContains    = "contains"
	AssertionTypeNotContains = "not_contains"
	AssertionTypeRegex       = "regex"
	AssertionTypeNotRegex    = "not_regex"

	// 新增 PromQL 专用类型
	AssertionTypePromQLThreshold = "promql_threshold" // PromQL 阈值判断
	AssertionTypePromQLRange     = "promql_range"     // PromQL 区间判断
	AssertionTypePromQLExists    = "promql_exists"    // PromQL 存在性判断
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

	// PromQL 专用断言类型
	if strings.HasPrefix(assertionType, "promql_") {
		return v.validatePromQL(assertionType, assertionValue, output)
	}

	// 原有断言类型
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

// ===== PromQL 专用断言方法 =====

// PromQLQueryResult PromQL 查询结果结构
type PromQLQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`  // Instant Query: [timestamp, "value"]
			Values [][]interface{}   `json:"values"` // Range Query: [[timestamp, "value"], ...]
		} `json:"result"`
	} `json:"data"`
}

// validatePromQL 处理 PromQL 断言
func (v *AssertionValidator) validatePromQL(assertionType, assertionValue, promqlResult string) *AssertionResult {
	// 解析 PromQL 查询结果
	var result PromQLQueryResult
	if err := json.Unmarshal([]byte(promqlResult), &result); err != nil {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("PromQL 结果解析失败: %v", err)}
	}

	// 检查查询状态
	if result.Status != "success" {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("PromQL 查询失败，状态: %s", result.Status)}
	}

	// 解析断言配置
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(assertionValue), &config); err != nil {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("断言配置解析失败: %v", err)}
	}

	switch assertionType {
	case AssertionTypePromQLThreshold:
		return v.validatePromQLThreshold(config, result)
	case AssertionTypePromQLRange:
		return v.validatePromQLRange(config, result)
	case AssertionTypePromQLExists:
		return v.validatePromQLExists(config, result)
	default:
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("不支持的 PromQL 断言类型: %s", assertionType)}
	}
}

// validatePromQLThreshold 阈值断言
func (v *AssertionValidator) validatePromQLThreshold(config map[string]interface{}, result PromQLQueryResult) *AssertionResult {
	// 检查是否有结果
	if len(result.Data.Result) == 0 {
		return &AssertionResult{Pass: false, Message: "PromQL 查询无结果（指标可能不存在）"}
	}

	// 提取指标值
	firstResult := result.Data.Result[0]
	var valueStr string
	var ok bool

	// 判断是 Instant Query 还是 Range Query
	if len(firstResult.Value) >= 2 {
		// Instant Query
		valueStr, ok = firstResult.Value[1].(string)
	} else if len(firstResult.Values) > 0 {
		// Range Query: 取最后一个时间点的值
		lastValue := firstResult.Values[len(firstResult.Values)-1]
		if len(lastValue) >= 2 {
			valueStr, ok = lastValue[1].(string)
		}
	}

	if !ok || valueStr == "" {
		return &AssertionResult{Pass: false, Message: "指标值格式错误"}
	}

	metricValue, err := v.parsePromQLValue(valueStr)
	if err != nil {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("指标值解析失败: %v", err)}
	}

	// 获取断言配置
	operator, _ := config["operator"].(string)
	threshold, _ := config["value"].(float64)
	message, _ := config["message"].(string)
	unit, _ := config["unit"].(string)

	if operator == "" {
		return &AssertionResult{Pass: false, Message: "断言配置缺少 operator 字段"}
	}

	// 执行比较
	var pass bool
	switch operator {
	case ">":
		pass = metricValue > threshold
	case ">=":
		pass = metricValue >= threshold
	case "<":
		pass = metricValue < threshold
	case "<=":
		pass = metricValue <= threshold
	case "==":
		pass = metricValue == threshold
	case "!=":
		pass = metricValue != threshold
	default:
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("不支持的操作符: %s", operator)}
	}

	if !pass {
		return &AssertionResult{
			Pass: false,
			Message: fmt.Sprintf("%s (实际值: %.2f%s, 阈值: %s %.2f%s)",
				message, metricValue, unit, operator, threshold, unit),
		}
	}

	return &AssertionResult{
		Pass: true,
		Message: fmt.Sprintf("通过: 实际值 %.2f%s %s 阈值 %.2f%s",
			metricValue, unit, operator, threshold, unit),
	}
}

// validatePromQLRange 区间断言
func (v *AssertionValidator) validatePromQLRange(config map[string]interface{}, result PromQLQueryResult) *AssertionResult {
	if len(result.Data.Result) == 0 {
		return &AssertionResult{Pass: false, Message: "PromQL 查询无结果"}
	}

	// 提取指标值
	firstResult := result.Data.Result[0]
	var valueStr string
	var ok bool

	if len(firstResult.Value) >= 2 {
		valueStr, ok = firstResult.Value[1].(string)
	} else if len(firstResult.Values) > 0 {
		lastValue := firstResult.Values[len(firstResult.Values)-1]
		if len(lastValue) >= 2 {
			valueStr, ok = lastValue[1].(string)
		}
	}

	if !ok || valueStr == "" {
		return &AssertionResult{Pass: false, Message: "指标值格式错误"}
	}

	metricValue, err := v.parsePromQLValue(valueStr)
	if err != nil {
		return &AssertionResult{Pass: false, Message: fmt.Sprintf("指标值解析失败: %v", err)}
	}

	// 获取断言配置
	min, _ := config["min"].(float64)
	max, _ := config["max"].(float64)
	invert, _ := config["invert"].(bool)
	message, _ := config["message"].(string)

	// 判断是否在区间内
	inRange := metricValue >= min && metricValue <= max
	pass := inRange != invert // XOR 逻辑

	if !pass {
		if invert {
			return &AssertionResult{
				Pass:    false,
				Message: fmt.Sprintf("%s (实际值 %.2f 应在区间 [%.2f, %.2f] 外)", message, metricValue, min, max),
			}
		}
		return &AssertionResult{
			Pass:    false,
			Message: fmt.Sprintf("%s (实际值 %.2f 应在区间 [%.2f, %.2f] 内)", message, metricValue, min, max),
		}
	}

	return &AssertionResult{
		Pass:    true,
		Message: fmt.Sprintf("通过: 实际值 %.2f 符合区间要求", metricValue),
	}
}

// validatePromQLExists 存在性断言
func (v *AssertionValidator) validatePromQLExists(config map[string]interface{}, result PromQLQueryResult) *AssertionResult {
	expect, _ := config["expect"].(bool)
	message, _ := config["message"].(string)

	hasData := len(result.Data.Result) > 0
	pass := hasData == expect

	if !pass {
		if expect {
			return &AssertionResult{
				Pass:    false,
				Message: fmt.Sprintf("%s (期望有数据，但查询无结果，可能服务已停止)", message),
			}
		}
		return &AssertionResult{
			Pass:    false,
			Message: fmt.Sprintf("%s (期望无数据，但查询有结果)", message),
		}
	}

	return &AssertionResult{
		Pass:    true,
		Message: "通过: 存在性判断符合预期",
	}
}

// parsePromQLValue 解析 PromQL 指标值（支持特殊值）
func (v *AssertionValidator) parsePromQLValue(s string) (float64, error) {
	// 处理特殊值
	switch s {
	case "NaN":
		return 0, fmt.Errorf("值为 NaN")
	case "+Inf":
		return 0, fmt.Errorf("值为 +Inf")
	case "-Inf":
		return 0, fmt.Errorf("值为 -Inf")
	}

	var value float64
	_, err := fmt.Sscanf(s, "%f", &value)
	return value, err
}
