package inspection_mgmt

import (
	"regexp"
	"strings"
)

// VariableReplacer 变量替换器，负责将文本中的 {{variableName}} 替换为实际值
type VariableReplacer struct{}

// NewVariableReplacer 创建变量替换器
func NewVariableReplacer() *VariableReplacer {
	return &VariableReplacer{}
}

// Replace 替换文本中的变量引用
// 支持格式：{{variableName}}
func (r *VariableReplacer) Replace(text string, variables map[string]string) string {
	if text == "" {
		return text
	}

	// 如果没有变量，直接返回原文本
	if variables == nil || len(variables) == 0 {
		return text
	}

	// 使用正则表达式匹配 {{variableName}} 格式
	re := regexp.MustCompile(`\{\{([a-zA-Z0-9_]+)\}\}`)

	result := re.ReplaceAllStringFunc(text, func(match string) string {
		// 提取变量名（去掉 {{ 和 }}）
		varName := strings.TrimPrefix(match, "{{")
		varName = strings.TrimSuffix(varName, "}}")

		// 查找变量值
		if value, exists := variables[varName]; exists {
			return value
		}

		// 如果变量不存在，保持原样
		return match
	})

	return result
}

// ReplaceCommand 替换命令中的变量
func (r *VariableReplacer) ReplaceCommand(command string, variables map[string]string) string {
	return r.Replace(command, variables)
}

// ReplaceScriptContent 替换脚本内容中的变量
func (r *VariableReplacer) ReplaceScriptContent(scriptContent string, variables map[string]string) string {
	return r.Replace(scriptContent, variables)
}

// ReplacePromQL 替换 PromQL 查询中的变量
func (r *VariableReplacer) ReplacePromQL(promql string, variables map[string]string) string {
	return r.Replace(promql, variables)
}

// ExtractVariableNames 提取文本中引用的所有变量名
func (r *VariableReplacer) ExtractVariableNames(text string) []string {
	if text == "" {
		return []string{}
	}

	re := regexp.MustCompile(`\{\{([a-zA-Z0-9_]+)\}\}`)
	matches := re.FindAllStringSubmatch(text, -1)

	varNames := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			if !seen[varName] {
				varNames = append(varNames, varName)
				seen[varName] = true
			}
		}
	}

	return varNames
}
