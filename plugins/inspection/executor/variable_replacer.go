package executor

import (
	"fmt"
	"strings"
)

// VariableReplacer 变量替换器
type VariableReplacer struct{}

// Replace 替换内容中的变量占位符
func (r *VariableReplacer) Replace(content string, variables map[string]string) string {
	if len(variables) == 0 {
		return content
	}

	result := content
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}
