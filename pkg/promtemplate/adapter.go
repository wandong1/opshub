package promtemplate

import (
	"bytes"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// Adapter Prometheus 模板适配器
type Adapter struct {
	funcMap template.FuncMap
}

// NewAdapter 创建新的 Prometheus 模板适配器
func NewAdapter() *Adapter {
	return &Adapter{
		funcMap: template.FuncMap{
			// Prometheus 内置函数
			"humanize":             humanize,
			"humanize1024":         humanize1024,
			"humanizeDuration":     humanizeDuration,
			"humanizePercentage":   humanizePercentage,
			"humanizeTimestamp":    humanizeTimestamp,
			"title":                strings.Title,
			"toUpper":              strings.ToUpper,
			"toLower":              strings.ToLower,
			"match":                regexMatch,
			"reReplaceAll":         reReplaceAll,
			"graphLink":            graphLink,
			"tableLink":            tableLink,
			"printf":               fmt.Sprintf,
			"stringsJoin":          strings.Join,
			"stringsTrimSpace":     strings.TrimSpace,
			"stringsTrimPrefix":    strings.TrimPrefix,
			"stringsTrimSuffix":    strings.TrimSuffix,
			"stringsContains":      strings.Contains,
			"stringsHasPrefix":     strings.HasPrefix,
			"stringsHasSuffix":     strings.HasSuffix,
		},
	}
}

// ConvertTemplate 转换 Prometheus 模板语法为 Go template 语法
func (a *Adapter) ConvertTemplate(promTpl string) string {
	// 1. 转换 $labels.xxx -> .xxx
	re1 := regexp.MustCompile(`\$labels\.(\w+)`)
	tpl := re1.ReplaceAllString(promTpl, ".$1")

	// 2. 转换 $value -> .Value
	re2 := regexp.MustCompile(`\$value`)
	tpl = re2.ReplaceAllString(tpl, ".Value")

	// 3. 转换 $externalLabels.xxx -> .ExternalLabels.xxx
	re3 := regexp.MustCompile(`\$externalLabels\.(\w+)`)
	tpl = re3.ReplaceAllString(tpl, ".ExternalLabels.$1")

	return tpl
}

// Render 渲染模板（自动转换 Prometheus 语法）
func (a *Adapter) Render(tplStr string, data map[string]interface{}) (string, error) {
	// 转换模板语法
	convertedTpl := a.ConvertTemplate(tplStr)

	// 创建模板并添加函数
	tpl, err := template.New("").Funcs(a.funcMap).Parse(convertedTpl)
	if err != nil {
		return "", fmt.Errorf("模板解析失败: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("模板渲染失败: %w", err)
	}

	return buf.String(), nil
}

// humanize 将数字转换为人类可读格式（1000 进制）
// 例如：1000 -> 1k, 1000000 -> 1M
func humanize(v float64) string {
	if v == 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Sprintf("%.4g", v)
	}

	if math.Abs(v) >= 1 {
		prefix := ""
		for _, p := range []string{"k", "M", "G", "T", "P", "E", "Z", "Y"} {
			if math.Abs(v) < 1000 {
				break
			}
			prefix = p
			v /= 1000
		}
		return fmt.Sprintf("%.4g%s", v, prefix)
	}

	prefix := ""
	for _, p := range []string{"m", "u", "n", "p", "f", "a", "z", "y"} {
		if math.Abs(v) >= 1 {
			break
		}
		prefix = p
		v *= 1000
	}
	return fmt.Sprintf("%.4g%s", v, prefix)
}

// humanize1024 将数字转换为人类可读格式（1024 进制）
// 例如：1024 -> 1Ki, 1048576 -> 1Mi
func humanize1024(v float64) string {
	if v == 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Sprintf("%.4g", v)
	}

	prefix := ""
	for _, p := range []string{"Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "Zi", "Yi"} {
		if math.Abs(v) < 1024 {
			break
		}
		prefix = p
		v /= 1024
	}
	return fmt.Sprintf("%.4g%s", v, prefix)
}

// humanizeDuration 将秒数转换为人类可读的时间格式
// 例如：3661 -> 1h1m1s
func humanizeDuration(v float64) string {
	if v == 0 {
		return "0s"
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Sprintf("%.4g", v)
	}
	if v < 0 {
		return fmt.Sprintf("%.4gs", v)
	}

	seconds := int64(v)
	duration := time.Duration(seconds) * time.Second

	// 格式化为更友好的格式
	days := duration / (24 * time.Hour)
	duration -= days * 24 * time.Hour
	hours := duration / time.Hour
	duration -= hours * time.Hour
	minutes := duration / time.Minute
	duration -= minutes * time.Minute
	secs := duration / time.Second

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if secs > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", secs))
	}

	return strings.Join(parts, "")
}

// humanizePercentage 将小数转换为百分比格式
// 例如：0.8567 -> 85.67%
func humanizePercentage(v float64) string {
	return fmt.Sprintf("%.4g%%", v*100)
}

// humanizeTimestamp 将 Unix 时间戳转换为人类可读的时间格式
// 例如：1609459200 -> 2021-01-01 00:00:00
func humanizeTimestamp(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Sprintf("%.4g", v)
	}
	t := time.Unix(int64(v), 0).UTC()
	return t.Format("2006-01-02 15:04:05")
}

// regexMatch 正则表达式匹配
func regexMatch(pattern, text string) (bool, error) {
	return regexp.MatchString(pattern, text)
}

// reReplaceAll 正则表达式替换
func reReplaceAll(pattern, repl, text string) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllString(text, repl)
}

// graphLink 生成 Prometheus 图表链接
func graphLink(expr string) string {
	return fmt.Sprintf("/graph?g0.expr=%s", url.QueryEscape(expr))
}

// tableLink 生成 Prometheus 表格链接
func tableLink(expr string) string {
	return fmt.Sprintf("/graph?g0.expr=%s&g0.tab=1", url.QueryEscape(expr))
}
