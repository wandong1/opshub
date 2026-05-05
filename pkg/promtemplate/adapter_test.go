package promtemplate

import (
	"testing"
)

func TestConvertTemplate(t *testing.T) {
	adapter := NewAdapter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "转换 $labels.instance",
			input:    "主机 {{ $labels.instance }} 异常",
			expected: "主机 {{ .instance }} 异常",
		},
		{
			name:     "转换 $value",
			input:    "CPU 使用率为 {{ $value }}%",
			expected: "CPU 使用率为 {{ .Value }}%",
		},
		{
			name:     "转换多个 $labels",
			input:    "{{ $labels.instance }} 的 {{ $labels.job }} 任务异常",
			expected: "{{ .instance }} 的 {{ .job }} 任务异常",
		},
		{
			name:     "混合转换",
			input:    "主机 {{ $labels.instance }} CPU 使用率为 {{ $value | printf \"%.2f\" }}%",
			expected: "主机 {{ .instance }} CPU 使用率为 {{ .Value | printf \"%.2f\" }}%",
		},
		{
			name:     "转换 $externalLabels",
			input:    "集群: {{ $externalLabels.cluster }}",
			expected: "集群: {{ .ExternalLabels.cluster }}",
		},
		{
			name:     "不转换已经是 Go template 的语法",
			input:    "主机 {{ .instance }} CPU 使用率为 {{ .Value }}%",
			expected: "主机 {{ .instance }} CPU 使用率为 {{ .Value }}%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.ConvertTemplate(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRender(t *testing.T) {
	adapter := NewAdapter()

	tests := []struct {
		name     string
		template string
		data     map[string]interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "渲染 Prometheus 语法 - $labels.instance",
			template: "主机 {{ $labels.instance }} 异常",
			data: map[string]interface{}{
				"instance": "192.168.1.1:9100",
			},
			expected: "主机 192.168.1.1:9100 异常",
			wantErr:  false,
		},
		{
			name:     "渲染 Prometheus 语法 - $value",
			template: "CPU 使用率为 {{ $value }}%",
			data: map[string]interface{}{
				"Value": 85.67,
			},
			expected: "CPU 使用率为 85.67%",
			wantErr:  false,
		},
		{
			name:     "渲染 Prometheus 语法 - printf 格式化",
			template: "CPU 使用率为 {{ $value | printf \"%.2f\" }}%",
			data: map[string]interface{}{
				"Value": 85.6789,
			},
			expected: "CPU 使用率为 85.68%",
			wantErr:  false,
		},
		{
			name:     "渲染 Go template 语法",
			template: "主机 {{ .instance }} CPU 使用率为 {{ .Value | printf \"%.2f\" }}%",
			data: map[string]interface{}{
				"instance": "192.168.1.1:9100",
				"Value":    85.6789,
			},
			expected: "主机 192.168.1.1:9100 CPU 使用率为 85.68%",
			wantErr:  false,
		},
		{
			name:     "混合使用 Prometheus 和 Go template 语法",
			template: "规则: {{ .RuleName }}, 实例: {{ $labels.instance }}, 值: {{ $value | printf \"%.2f\" }}",
			data: map[string]interface{}{
				"RuleName": "CPU 使用率过高",
				"instance": "192.168.1.1:9100",
				"Value":    85.6789,
			},
			expected: "规则: CPU 使用率过高, 实例: 192.168.1.1:9100, 值: 85.68",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := adapter.Render(tt.template, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("Render() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHumanize(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected string
	}{
		{"零值", 0, "0"},
		{"小于 1000", 999, "999"},
		{"1000", 1000, "1k"},
		{"1500", 1500, "1.5k"},
		{"1000000", 1000000, "1M"},
		{"1500000", 1500000, "1.5M"},
		{"1000000000", 1000000000, "1G"},
		{"负数", -1500, "-1.5k"},
		{"小数", 0.5, "500m"},
		{"更小的数", 0.0005, "500u"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := humanize(tt.value)
			if result != tt.expected {
				t.Errorf("humanize(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestHumanize1024(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected string
	}{
		{"零值", 0, "0"},
		{"小于 1024", 1023, "1023"},
		{"1024", 1024, "1Ki"},
		{"1536", 1536, "1.5Ki"},
		{"1048576", 1048576, "1Mi"},
		{"1073741824", 1073741824, "1Gi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := humanize1024(tt.value)
			if result != tt.expected {
				t.Errorf("humanize1024(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestHumanizeDuration(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected string
	}{
		{"零值", 0, "0s"},
		{"1 秒", 1, "1s"},
		{"60 秒", 60, "1m"},
		{"61 秒", 61, "1m1s"},
		{"3661 秒", 3661, "1h1m1s"},
		{"86400 秒", 86400, "1d"},
		{"90061 秒", 90061, "1d1h1m1s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := humanizeDuration(tt.value)
			if result != tt.expected {
				t.Errorf("humanizeDuration(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestHumanizePercentage(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected string
	}{
		{"0", 0, "0%"},
		{"0.5", 0.5, "50%"},
		{"0.8567", 0.8567, "85.67%"},
		{"1", 1, "100%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := humanizePercentage(tt.value)
			if result != tt.expected {
				t.Errorf("humanizePercentage(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestHumanizeTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected string
	}{
		{"Unix 纪元", 0, "1970-01-01 00:00:00"},
		{"2021-01-01", 1609459200, "2021-01-01 00:00:00"},
		{"2026-05-05 08:00", 1777968000, "2026-05-05 08:00:00"}, // 修正为实际的 UTC 时间
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := humanizeTimestamp(tt.value)
			if result != tt.expected {
				t.Errorf("humanizeTimestamp(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestReReplaceAll(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		repl     string
		text     string
		expected string
	}{
		{
			name:     "替换数字",
			pattern:  `\d+`,
			repl:     "X",
			text:     "port 9100",
			expected: "port X",
		},
		{
			name:     "替换多个匹配",
			pattern:  `\d+`,
			repl:     "X",
			text:     "192.168.1.1:9100",
			expected: "X.X.X.X:X",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reReplaceAll(tt.pattern, tt.repl, tt.text)
			if result != tt.expected {
				t.Errorf("reReplaceAll() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func BenchmarkConvertTemplate(b *testing.B) {
	adapter := NewAdapter()
	template := "主机 {{ $labels.instance }} CPU 使用率为 {{ $value | printf \"%.2f\" }}%"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.ConvertTemplate(template)
	}
}

func BenchmarkRender(b *testing.B) {
	adapter := NewAdapter()
	template := "主机 {{ $labels.instance }} CPU 使用率为 {{ $value | printf \"%.2f\" }}%"
	data := map[string]interface{}{
		"instance": "192.168.1.1:9100",
		"Value":    85.6789,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.Render(template, data)
	}
}
