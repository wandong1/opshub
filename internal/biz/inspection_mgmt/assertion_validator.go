package inspection_mgmt

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ProbeDetailsForAssertion 拨测详细信息（用于断言解析）
// 注意：这是 internal/service/inspection_mgmt/ProbeDetails 的简化版本，避免循环导入
type ProbeDetailsForAssertion struct {
	ProbeType        string                   `json:"probe_type"`
	Success          bool                     `json:"success"`
	LatencyMs        float64                  `json:"latency_ms"`
	StatusCode       int                      `json:"status_code"`
	AssertionResults []AssertionResultDetail  `json:"assertion_results"`
	AssertionPass    int                      `json:"assertion_pass"`
	AssertionFail    int                      `json:"assertion_fail"`
}

// AssertionResultDetail 原始断言结果详情
type AssertionResultDetail struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Actual  string `json:"actual"`
	Error   string `json:"error"`
}

// AssertionValidator 断言校验器
type AssertionValidator struct{}

// AssertionResult 断言结果
type AssertionResult struct {
	Pass    bool   `json:"pass"`
	Message string `json:"message"`
	Skip    bool   `json:"skip"` // 标识是否跳过断言（无断言规则时为 true）
}

// Validate 执行断言校验
func (v *AssertionValidator) Validate(assertionType, assertionValue, output string) *AssertionResult {
	if assertionType == "" {
		return &AssertionResult{Pass: true, Message: "无断言规则，跳过校验", Skip: true}
	}

	// 拨测专用断言类型（需要解析 ProbeDetails）
	if strings.HasPrefix(assertionType, "probe_") {
		probeDetails := v.parseProbeDetails(output)
		if probeDetails == nil {
			return &AssertionResult{
				Pass:    false,
				Message: "无法解析拨测结果，请确保执行类型为 probe",
			}
		}

		switch assertionType {
		case "probe_success":
			return v.validateProbeSuccess(probeDetails)
		case "probe_latency_lt":
			return v.validateProbeLatency(probeDetails, assertionValue)
		case "probe_assertion_all":
			return v.validateProbeAssertion(probeDetails)
		case "probe_status_code":
			return v.validateProbeStatusCode(probeDetails, assertionValue)
		default:
			return &AssertionResult{Pass: false, Message: fmt.Sprintf("未知的拨测断言类型: %s", assertionType)}
		}
	}

	// 通用断言类型
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

// ==================== 拨测专用断言方法 ====================

// parseProbeDetails 从 output 中解析 ProbeDetails
// output 应该是包含 probe_details 的 JSON 字符串
func (v *AssertionValidator) parseProbeDetails(output string) *ProbeDetailsForAssertion {
	// 尝试直接解析为 ProbeDetails
	var details ProbeDetailsForAssertion
	if err := json.Unmarshal([]byte(output), &details); err == nil && details.ProbeType != "" {
		return &details
	}

	// 尝试解析为包含 probe_details 字段的对象
	var wrapper struct {
		ProbeDetails *ProbeDetailsForAssertion `json:"probe_details"`
	}
	if err := json.Unmarshal([]byte(output), &wrapper); err == nil && wrapper.ProbeDetails != nil {
		return wrapper.ProbeDetails
	}

	return nil
}

// validateProbeSuccess 验证拨测是否成功
func (v *AssertionValidator) validateProbeSuccess(details *ProbeDetailsForAssertion) *AssertionResult {
	statusText := "失败"
	if details.Success {
		statusText = "成功"
	}

	return &AssertionResult{
		Pass:    details.Success,
		Message: fmt.Sprintf("拨测%s", statusText),
	}
}

// validateProbeLatency 验证响应时间小于阈值
func (v *AssertionValidator) validateProbeLatency(details *ProbeDetailsForAssertion, threshold string) *AssertionResult {
	thresholdMs, err := strconv.ParseFloat(threshold, 64)
	if err != nil {
		return &AssertionResult{
			Pass:    false,
			Message: fmt.Sprintf("阈值格式错误: %v", err),
		}
	}

	pass := details.LatencyMs < thresholdMs
	symbol := ">="
	if pass {
		symbol = "<"
	}

	return &AssertionResult{
		Pass:    pass,
		Message: fmt.Sprintf("响应时间 %.2fms %s 阈值 %.2fms", details.LatencyMs, symbol, thresholdMs),
	}
}

// validateProbeAssertion 验证原始断言全部通过
func (v *AssertionValidator) validateProbeAssertion(details *ProbeDetailsForAssertion) *AssertionResult {
	// 如果没有断言结果，视为跳过
	if len(details.AssertionResults) == 0 {
		return &AssertionResult{
			Pass:    true,
			Skip:    true,
			Message: "无原始断言，跳过校验",
		}
	}

	pass := details.AssertionFail == 0 && details.AssertionPass > 0

	return &AssertionResult{
		Pass:    pass,
		Message: fmt.Sprintf("原始断言: %d通过/%d失败", details.AssertionPass, details.AssertionFail),
	}
}

// validateProbeStatusCode 验证 HTTP 状态码
func (v *AssertionValidator) validateProbeStatusCode(details *ProbeDetailsForAssertion, expected string) *AssertionResult {
	expectedCode, err := strconv.Atoi(expected)
	if err != nil {
		return &AssertionResult{
			Pass:    false,
			Message: fmt.Sprintf("期望状态码格式错误: %v", err),
		}
	}

	pass := details.StatusCode == expectedCode
	symbol := "!="
	if pass {
		symbol = "=="
	}

	return &AssertionResult{
		Pass:    pass,
		Message: fmt.Sprintf("状态码 %d %s 期望 %d", details.StatusCode, symbol, expectedCode),
	}
}

// ==================== 多条断言支持 ====================

// AssertionRule 断言规则
type AssertionRule struct {
	Type        string `json:"type"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// ValidateMultiple 执行多条断言校验
// logic: "and" 表示所有断言必须通过，"or" 表示任一断言通过即可
func (v *AssertionValidator) ValidateMultiple(assertions []AssertionRule, logic string, output string) *AssertionResult {
	if len(assertions) == 0 {
		return &AssertionResult{Pass: true, Message: "无断言规则，跳过校验", Skip: true}
	}

	// 限制断言数量
	if len(assertions) > 10 {
		return &AssertionResult{
			Pass:    false,
			Message: "断言数量超过限制（最多10条）",
		}
	}

	var passedAssertions []string
	var failedAssertions []string
	var skippedCount int

	for i, rule := range assertions {
		result := v.Validate(rule.Type, rule.Value, output)

		desc := rule.Description
		if desc == "" {
			desc = fmt.Sprintf("断言%d", i+1)
		}

		if result.Skip {
			skippedCount++
			continue
		}

		if result.Pass {
			passedAssertions = append(passedAssertions, fmt.Sprintf("✓ %s: %s", desc, result.Message))
		} else {
			failedAssertions = append(failedAssertions, fmt.Sprintf("✗ %s: %s", desc, result.Message))
		}
	}

	totalAssertions := len(assertions) - skippedCount
	passedCount := len(passedAssertions)
	failedCount := len(failedAssertions)

	// AND 逻辑：所有断言必须通过
	if logic == "and" || logic == "" {
		if failedCount == 0 && passedCount > 0 {
			return &AssertionResult{
				Pass:    true,
				Message: fmt.Sprintf("所有断言通过 (%d/%d)\n%s", passedCount, totalAssertions, strings.Join(passedAssertions, "\n")),
			}
		}

		allMessages := append(failedAssertions, passedAssertions...)
		return &AssertionResult{
			Pass:    false,
			Message: fmt.Sprintf("断言失败 (%d/%d)\n%s", failedCount, totalAssertions, strings.Join(allMessages, "\n")),
		}
	}

	// OR 逻辑：任一断言通过即可
	if logic == "or" {
		if passedCount > 0 {
			return &AssertionResult{
				Pass:    true,
				Message: fmt.Sprintf("断言通过 (%d/%d)\n%s", passedCount, totalAssertions, strings.Join(passedAssertions, "\n")),
			}
		}

		return &AssertionResult{
			Pass:    false,
			Message: fmt.Sprintf("所有断言失败 (%d/%d)\n%s", failedCount, totalAssertions, strings.Join(failedAssertions, "\n")),
		}
	}

	return &AssertionResult{
		Pass:    false,
		Message: fmt.Sprintf("未知的断言逻辑: %s", logic),
	}
}


