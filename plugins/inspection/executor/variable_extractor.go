package executor

import (
	"fmt"
	"regexp"
)

// VariableExtractor 变量提取器
type VariableExtractor struct{}

// Extract 从输出中提取变量
func (e *VariableExtractor) Extract(variableName, variableRegex, output string) (map[string]string, error) {
	if variableName == "" || variableRegex == "" {
		return nil, nil
	}

	re, err := regexp.Compile(variableRegex)
	if err != nil {
		return nil, fmt.Errorf("正则表达式编译失败: %v", err)
	}

	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return nil, fmt.Errorf("未匹配到变量值")
	}

	// 使用第一个捕获组作为变量值
	return map[string]string{
		variableName: matches[1],
	}, nil
}
