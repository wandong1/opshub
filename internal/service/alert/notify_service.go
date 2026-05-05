package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	alertdata "github.com/ydcloud-dy/opshub/internal/data/alert"
)

// NotifyService 告警通知服务
type NotifyService struct {
	channelRepo *alertdata.ChannelRepo
}

func NewNotifyService(channelRepo *alertdata.ChannelRepo) *NotifyService {
	return &NotifyService{channelRepo: channelRepo}
}

// notifyTplData 模板渲染数据
type notifyTplData struct {
	RuleName          string
	Severity          string
	SeverityLabel     string // 中文级别名
	Value             float64
	ResolveValue      *float64
	FiredAt           string
	ResolvedAt        string
	Labels            string
	Annotations       string
	LabelsDetail      string // 格式化标签列表
	AnnotationsDetail string // 格式化注解列表
	Title             string // annotations.title
	Description       string // annotations.description
	Mentions          string // @用户列表（手机号或userid）
}

// Send 发送通知（firing 或 resolved）
// phones: 接收用户手机号列表，nil=@all，空切片=不@任何人，非空切片=@指定用户
func (s *NotifyService) Send(ctx context.Context, ch *biz.AlertNotifyChannel, event *biz.AlertEvent, isResolve bool, phones []string) {
	appLogger.Info("发送通知",
		zap.String("channel", ch.Name),
		zap.Bool("isResolve", isResolve),
		zap.Int("alertTemplateLen", len(ch.AlertTemplate)),
		zap.Int("resolveTemplateLen", len(ch.ResolveTemplate)))

	tplStr := ch.AlertTemplate
	if isResolve {
		tplStr = ch.ResolveTemplate
		appLogger.Info("使用恢复模板", zap.String("template", tplStr))
	} else {
		appLogger.Info("使用告警模板", zap.String("template", tplStr))
	}
	if tplStr == "" {
		tplStr = defaultTemplate(ch.Type, isResolve)
		appLogger.Info("使用默认模板", zap.String("template", tplStr))
	}

	msg := renderTemplate(tplStr, event, phones)

	var err error
	switch ch.Type {
	case "wechat_work":
		err = sendWechatWork(ch.Config, msg, phones, isResolve)
	case "dingtalk":
		err = sendDingTalk(ch.Config, msg, phones)
	case "sms":
		err = sendSMS(ch.Config, msg, phones)
	case "phone":
		err = sendPhone(ch.Config, msg, phones)
	case "ai_agent":
		if ch.AIHookEnabled {
			err = sendAIAgent(ch.Config, msg, event)
		}
	default:
		appLogger.Warn("未知通知通道类型", zap.String("type", ch.Type))
		return
	}
	if err != nil {
		appLogger.Error("告警通知发送失败", zap.String("channel", ch.Name), zap.Error(err))
	}
}

func severityLabel(s string) string {
	switch s {
	case "critical":
		return "紧急(P1)"
	case "major":
		return "严重(P2)"
	case "minor":
		return "一般(P3)"
	case "warning":
		return "提示(P4)"
	}
	return s
}

// formatKVDetail 将 JSON map 格式化为 markdown 列表
func formatKVDetail(jsonStr string) string {
	if jsonStr == "" || jsonStr == "{}" || jsonStr == "null" {
		return ""
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	for k, v := range m {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func renderTemplate(tplStr string, event *biz.AlertEvent, phones []string) string {
	tpl, err := template.New("").Parse(tplStr)
	if err != nil {
		return tplStr
	}
	// 解析 annotations JSON 取 title/description
	var annTitle, annDesc string
	if event.Annotations != "" {
		var ann map[string]string
		if json.Unmarshal([]byte(event.Annotations), &ann) == nil {
			annTitle = ann["title"]
			annDesc = ann["description"]
		}
	}
	// 构建 @提及字符串
	// 修复：phones 为 nil 时才 @all，空切片表示不 @任何人
	mentions := ""
	if phones == nil {
		mentions = "@all"
	} else if len(phones) > 0 {
		var sb strings.Builder
		for _, p := range phones {
			sb.WriteString("@")
			sb.WriteString(p)
			sb.WriteString(" ")
		}
		mentions = strings.TrimSpace(sb.String())
	}

	// 解析 Labels 为 map，支持模板中直接引用标签字段
	labelsMap := make(map[string]string)
	if event.Labels != "" {
		json.Unmarshal([]byte(event.Labels), &labelsMap)
	}

	// 格式化数值：保留两位小数
	valueFormatted := fmt.Sprintf("%.2f", event.Value)
	var resolveValueFormatted string
	if event.ResolveValue != nil {
		resolveValueFormatted = fmt.Sprintf("%.2f", *event.ResolveValue)
	}

	// 构建模板数据（使用 map 支持动态字段）
	data := map[string]interface{}{
		"RuleName":          event.RuleName,
		"Severity":          event.Severity,
		"SeverityLabel":     severityLabel(event.Severity),
		"Value":             valueFormatted,           // 格式化后的字符串
		"ValueRaw":          event.Value,              // 原始数值（兼容需要数值计算的场景）
		"ResolveValue":      resolveValueFormatted,    // 格式化后的字符串
		"ResolveValueRaw":   event.ResolveValue,       // 原始数值
		"FiredAt":           event.FiredAt.Format("2006-01-02 15:04:05"),
		"Labels":            event.Labels,
		"Annotations":       event.Annotations,
		"LabelsDetail":      formatKVDetail(event.Labels),
		"AnnotationsDetail": formatKVDetail(event.Annotations),
		"Title":             annTitle,
		"Description":       annDesc,
		"Mentions":          mentions,
	}
	if event.ResolvedAt != nil {
		data["ResolvedAt"] = event.ResolvedAt.Format("2006-01-02 15:04:05")
	}

	// 将 Labels 中的字段添加到模板数据中，支持 {{.instance}}、{{.job}} 等
	for k, v := range labelsMap {
		data[k] = v
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return tplStr
	}
	return buf.String()
}

// func sendWechatWork(configJSON, msg string, phones []string) error {
// 	var cfg struct {
// 		WebhookURL string `json:"webhookUrl"`
// 	}
// 	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
// 		return err
// 	}
// 	payloadText :=  map[string]interface{}{
// 		"msgtype":  "text",
// 		"text": map[string]string{"content": "告警信息请关注！"},
// 	}
// 	payload := map[string]interface{}{
// 		"msgtype":  "markdown",
// 		"markdown": map[string]string{"content": msg},
// 	}
// 	if len(phones) == 0 {
// 		// @所有人
// 		payloadText["text"]["mentioned_mobile_list"] = []string{"@all"}
// 	} else {
// 		payload["mentioned_mobile_list"] = phones
// 	}
// 	return postJSON(cfg.WebhookURL, payload)
// }

func sendWechatWork(configJSON, msg string, phones []string, isResolve bool) error {
	var cfg struct {
		WebhookURL string `json:"webhookUrl"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return err
	}

	// 修复：phones 为 nil 时才 @all，空切片表示不 @任何人
	var mentionList []string
	if phones == nil {
		mentionList = []string{"@all"}
	} else {
		mentionList = phones
	}

	noticeText := "告警通知，请相关人员及时处理！"
	if isResolve {
		noticeText = "告警已恢复，请知悉！"
	}

	textPayload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]interface{}{
			"content":               noticeText,
			"mentioned_mobile_list": mentionList,
		},
	}

	if err := postJSON(cfg.WebhookURL, textPayload); err != nil {
		return err
	}

	markdownPayload := map[string]interface{}{
		"msgtype":  "markdown",
		"markdown": map[string]string{"content": msg},
	}

	return postJSON(cfg.WebhookURL, markdownPayload)
}

func sendDingTalk(configJSON, msg string, phones []string) error {
	var cfg struct {
		WebhookURL string `json:"webhookUrl"`
		Secret     string `json:"secret"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return err
	}
	// 修复：phones 为 nil 时才 @all，空切片表示不 @任何人
	var atObj map[string]interface{}
	if phones == nil {
		atObj = map[string]interface{}{"isAtAll": true}
	} else if len(phones) > 0 {
		atObj = map[string]interface{}{"atMobiles": phones, "isAtAll": false}
	} else {
		atObj = map[string]interface{}{"isAtAll": false}
	}
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": "SreHub 告警通知",
			"text":  msg,
		},
		"at": atObj,
	}
	webhookURL := cfg.WebhookURL
	if cfg.Secret != "" {
		timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
		webhookURL = fmt.Sprintf("%s&timestamp=%s", webhookURL, timestamp)
	}
	return postJSON(webhookURL, payload)
}

func sendSMS(configJSON, msg string, phones []string) error {
	// 预留：调用阿里云/腾讯云短信 SDK，当前仅记录日志
	appLogger.Info("[SMS通知预留]", zap.String("msg", msg), zap.Strings("phones", phones))
	return nil
}

func sendPhone(configJSON, msg string, phones []string) error {
	// 预留：调用语音通话 SDK
	appLogger.Info("[电话通知预留]", zap.String("msg", msg), zap.Strings("phones", phones))
	return nil
}

func sendAIAgent(configJSON, msg string, event *biz.AlertEvent) error {
	var cfg struct {
		HookURL string `json:"hookUrl"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return err
	}
	if cfg.HookURL == "" {
		return nil
	}
	payload := map[string]interface{}{
		"event":   event,
		"message": msg,
	}
	return postJSON(cfg.HookURL, payload)
}

func postJSON(url string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func defaultTemplate(channelType string, isResolve bool) string {
	if isResolve {
		switch channelType {
		case "wechat_work":
			return strings.TrimSpace(`
## ✅ SreHub 恢复通知
> **规则**: {{.RuleName}}
> **级别**: {{.SeverityLabel}}
> **恢复值**: {{if .ResolveValue}}{{.ResolveValue}}{{else}}{{.Value}}{{end}}
> **恢复时间**: {{.ResolvedAt}}
> **触发时间**: {{.FiredAt}}

**标签详情**:
{{.LabelsDetail}}

**注解详情**:
{{.AnnotationsDetail}}
`)
		case "dingtalk":
			return strings.TrimSpace(`
## ✅ SreHub 恢复通知
- **规则**: {{.RuleName}}
- **级别**: {{.SeverityLabel}}
- **恢复值**: {{if .ResolveValue}}{{.ResolveValue}}{{else}}{{.Value}}{{end}}
- **恢复时间**: {{.ResolvedAt}}
- **触发时间**: {{.FiredAt}}

**标签详情**:
{{.LabelsDetail}}

**注解详情**:
{{.AnnotationsDetail}}
`)
		default:
			return strings.TrimSpace(`【SreHub恢复】规则: {{.RuleName}} | 级别: {{.SeverityLabel}} | 值: {{if .ResolveValue}}{{.ResolveValue}}{{else}}{{.Value}}{{end}} | 恢复时间: {{.ResolvedAt}}`)
		}
	}
	switch channelType {
	case "wechat_work":
		return strings.TrimSpace(`
## 🔴 SreHub 告警通知
> **规则**: <font color=\"warning\">{{.RuleName}}</font>
> **级别**: {{.SeverityLabel}}
> **当前值**: {{.Value}}
> **触发时间**: {{.FiredAt}}

**标签详情**:
{{.LabelsDetail}}

**注解详情**:
{{.AnnotationsDetail}}
`)
	case "dingtalk":
		return strings.TrimSpace(`
## 🔴 SreHub 告警通知
- **规则**: {{.RuleName}}
- **级别**: {{.SeverityLabel}}
- **当前值**: {{.Value}}
- **触发时间**: {{.FiredAt}}

**标签详情**:
{{.LabelsDetail}}

**注解详情**:
{{.AnnotationsDetail}}
`)
	default:
		return strings.TrimSpace(`【SreHub告警】规则: {{.RuleName}} | 级别: {{.SeverityLabel}} | 值: {{.Value}} | 时间: {{.FiredAt}}`)
	}
}
