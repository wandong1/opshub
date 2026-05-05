package alert

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// MatchLabels 检查告警标签是否匹配屏蔽规则标签（子集匹配）
// eventLabelsJSON: 告警的完整标签 {"job":"prometheus","instance":"localhost:9090","pod":"pod-123"}
// ruleLabelsJSON: 屏蔽规则的标签（用户可能移除了部分标签）{"job":"prometheus","instance":"localhost:9090"}
// 返回：告警标签是否包含规则标签的所有键值对
func MatchLabels(eventLabelsJSON string, ruleLabelsJSON string) bool {
	eventMap := parseLabels(eventLabelsJSON)
	ruleMap := parseLabels(ruleLabelsJSON)

	// 规则标签必须是告警标签的子集
	for k, v := range ruleMap {
		if eventMap[k] != v {
			return false
		}
	}
	return true
}

// parseLabels 解析 JSON 标签为 map
func parseLabels(labelsJSON string) map[string]string {
	if labelsJSON == "" || labelsJSON == "{}" || labelsJSON == "null" {
		return make(map[string]string)
	}
	var labels map[string]string
	if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
		return make(map[string]string)
	}
	return labels
}

// MatchLabelFilter 支持模糊搜索（用于前端搜索功能）
// filter: "job=prome*" 或 "instance=*:9090"
func MatchLabelFilter(eventLabelsJSON string, filter string) bool {
	if filter == "" {
		return true
	}

	// 解析 key=value 格式
	parts := strings.SplitN(filter, "=", 2)
	if len(parts) != 2 {
		return false
	}

	key := strings.TrimSpace(parts[0])
	pattern := strings.TrimSpace(parts[1])

	labels := parseLabels(eventLabelsJSON)
	value, exists := labels[key]
	if !exists {
		return false
	}

	// 支持通配符 * 匹配
	return matchWildcard(value, pattern)
}

// matchWildcard 通配符匹配
func matchWildcard(value, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// 简单通配符实现：支持前缀、后缀、包含匹配
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		// *xxx* - 包含匹配
		substr := strings.Trim(pattern, "*")
		return strings.Contains(value, substr)
	} else if strings.HasPrefix(pattern, "*") {
		// *xxx - 后缀匹配
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(value, suffix)
	} else if strings.HasSuffix(pattern, "*") {
		// xxx* - 前缀匹配
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(value, prefix)
	}

	// 精确匹配
	return value == pattern
}

var _ context.Context // 避免未使用导入错误

// LabelMatcher 标签匹配器（用于订阅规则）
type LabelMatcher struct {
	Key   string `json:"key"`
	Op    string `json:"op"` // =, !=, =~, !~
	Value string `json:"value"`
}

// MatchSubscriptionLabels 检查告警标签是否匹配订阅规则的标签匹配器
func MatchSubscriptionLabels(eventLabelsJSON string, matchersJSON string) bool {
	if matchersJSON == "" {
		return true
	}

	var matchers []LabelMatcher
	if err := json.Unmarshal([]byte(matchersJSON), &matchers); err != nil {
		return true
	}

	if len(matchers) == 0 {
		return true
	}

	labels := parseLabels(eventLabelsJSON)

	for _, m := range matchers {
		labelValue, exists := labels[m.Key]

		switch m.Op {
		case "=":
			if !exists || labelValue != m.Value {
				return false
			}
		case "!=":
			if exists && labelValue == m.Value {
				return false
			}
		case "=~":
			if !exists {
				return false
			}
			matched, err := regexp.MatchString(m.Value, labelValue)
			if err != nil || !matched {
				return false
			}
		case "!~":
			if !exists {
				continue
			}
			matched, err := regexp.MatchString(m.Value, labelValue)
			if err != nil || matched {
				return false
			}
		default:
			return false
		}
	}

	return true
}

// MatchSubscriptionDataSource 检查告警数据源是否匹配订阅规则
func MatchSubscriptionDataSource(eventLabelsJSON string, dataSourceIDsJSON string) bool {
	if dataSourceIDsJSON == "" {
		return true
	}

	var dsIDs []uint
	if err := json.Unmarshal([]byte(dataSourceIDsJSON), &dsIDs); err != nil {
		return true
	}

	if len(dsIDs) == 0 {
		return true
	}

	labels := parseLabels(eventLabelsJSON)
	dsIDStr, exists := labels["datasource_id"]
	if !exists {
		return false
	}

	eventDSID, err := strconv.ParseUint(dsIDStr, 10, 32)
	if err != nil {
		return false
	}

	for _, id := range dsIDs {
		if id == uint(eventDSID) {
			return true
		}
	}

	return false
}

