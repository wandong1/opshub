package alert

import (
	"testing"
	"time"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
)

func TestRenderTemplate_ValueFormatting(t *testing.T) {
	event := &biz.AlertEvent{
		RuleName:  "CPU 使用率过高",
		Severity:  "critical",
		Value:     85.6789,
		FiredAt:   time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC),
		Labels:    `{"instance":"192.168.1.1:9100"}`,
		Annotations: `{"title":"告警标题","description":"告警描述"}`,
	}

	tpl := "当前值: {{.Value}}"
	result := renderTemplate(tpl, event, []string{"13800138000"})

	expected := "当前值: 85.68"
	if result != expected {
		t.Errorf("Value formatting failed: got %q, want %q", result, expected)
	}
}

func TestRenderTemplate_ResolveValueFormatting(t *testing.T) {
	resolveValue := 75.1234
	event := &biz.AlertEvent{
		RuleName:     "CPU 使用率过高",
		Severity:     "critical",
		Value:        85.6789,
		ResolveValue: &resolveValue,
		FiredAt:      time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC),
		Labels:       `{"instance":"192.168.1.1:9100"}`,
		Annotations:  `{"title":"告警标题","description":"告警描述"}`,
	}

	tpl := "恢复值: {{.ResolveValue}}"
	result := renderTemplate(tpl, event, []string{"13800138000"})

	expected := "恢复值: 75.12"
	if result != expected {
		t.Errorf("ResolveValue formatting failed: got %q, want %q", result, expected)
	}
}

func TestRenderTemplate_MentionsLogic(t *testing.T) {
	event := &biz.AlertEvent{
		RuleName:    "测试规则",
		Severity:    "critical",
		Value:       85.6789,
		FiredAt:     time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC),
		Labels:      `{}`,
		Annotations: `{}`,
	}

	tests := []struct {
		name     string
		phones   []string
		expected string
	}{
		{
			name:     "nil phones should @all",
			phones:   nil,
			expected: "@all",
		},
		{
			name:     "empty slice should not mention anyone",
			phones:   []string{},
			expected: "",
		},
		{
			name:     "single phone should mention that user",
			phones:   []string{"13800138000"},
			expected: "@13800138000",
		},
		{
			name:     "multiple phones should mention all users",
			phones:   []string{"13800138000", "13900139000"},
			expected: "@13800138000 @13900139000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl := "{{.Mentions}}"
			result := renderTemplate(tpl, event, tt.phones)
			if result != tt.expected {
				t.Errorf("Mentions logic failed: got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRenderTemplate_ValueRawField(t *testing.T) {
	event := &biz.AlertEvent{
		RuleName:    "测试规则",
		Severity:    "critical",
		Value:       85.6789,
		FiredAt:     time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC),
		Labels:      `{}`,
		Annotations: `{}`,
	}

	// 测试格式化值
	tpl1 := "格式化值: {{.Value}}"
	result1 := renderTemplate(tpl1, event, []string{})
	expected1 := "格式化值: 85.68"
	if result1 != expected1 {
		t.Errorf("Formatted value failed: got %q, want %q", result1, expected1)
	}

	// 测试原始数值（用于需要数值计算的场景）
	tpl2 := "原始值: {{.ValueRaw}}"
	result2 := renderTemplate(tpl2, event, []string{})
	expected2 := "原始值: 85.6789"
	if result2 != expected2 {
		t.Errorf("Raw value failed: got %q, want %q", result2, expected2)
	}
}
