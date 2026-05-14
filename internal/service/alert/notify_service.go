package alert

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"text/template"
	"time"

	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	alertdata "github.com/ydcloud-dy/opshub/internal/data/alert"
)

// NotifyService 告警通知服务
type NotifyService struct {
	channelRepo *alertdata.ChannelRepo
	db          *gorm.DB
}

func NewNotifyService(channelRepo *alertdata.ChannelRepo, db *gorm.DB) *NotifyService {
	return &NotifyService{
		channelRepo: channelRepo,
		db:          db,
	}
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
// userIDs: 接收用户ID列表，用于邮件通道直接查询邮箱
func (s *NotifyService) Send(ctx context.Context, ch *biz.AlertNotifyChannel, event *biz.AlertEvent, isResolve bool, phones []string, userIDs []uint) {
	appLogger.Info("发送通知",
		zap.String("channel", ch.Name),
		zap.Bool("isResolve", isResolve),
		zap.Int("alertTemplateLen", len(ch.AlertTemplate)),
		zap.Int("resolveTemplateLen", len(ch.ResolveTemplate)),
		zap.Uints("userIDs", userIDs))

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
	case "email":
		err = s.sendEmail(ctx, ch.Config, msg, userIDs)
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

// sendEmail 发送邮件通知
func (s *NotifyService) sendEmail(ctx context.Context, configJSON, msg string, userIDs []uint) error {
	var cfg struct {
		SMTPHost     string `json:"smtpHost"`
		SMTPPort     int    `json:"smtpPort"`
		SMTPUser     string `json:"smtpUser"`
		SMTPPassword string `json:"smtpPassword"`
		FromEmail    string `json:"fromEmail"`
		FromName     string `json:"fromName"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		appLogger.Error("解析邮件配置失败", zap.Error(err))
		return err
	}

	// 邮件发送必须指定接收用户，不支持 @all
	// userIDs == nil 或包含 0 表示 @all（IM 工具），但邮件不支持，直接跳过
	if userIDs == nil || len(userIDs) == 0 {
		appLogger.Info("邮件通道未指定接收用户，跳过发送")
		return nil
	}

	// 检查是否包含 userID=0（@all）
	for _, uid := range userIDs {
		if uid == 0 {
			appLogger.Info("邮件通道不支持 @all，跳过发送")
			return nil
		}
	}

	appLogger.Info("准备发送邮件",
		zap.String("smtpHost", cfg.SMTPHost),
		zap.Int("smtpPort", cfg.SMTPPort),
		zap.Uints("userIDs", userIDs))

	// 直接通过用户ID查询邮箱地址
	type userRow struct {
		Email    string
		RealName string
	}
	var users []userRow
	if err := s.db.WithContext(ctx).Table("sys_user").
		Select("email, real_name").
		Where("id IN ? AND email != ''", userIDs).
		Scan(&users).Error; err != nil {
		appLogger.Error("查询用户邮箱失败", zap.Error(err))
		return err
	}

	if len(users) == 0 {
		appLogger.Warn("未找到有效的用户邮箱", zap.Uints("userIDs", userIDs))
		return fmt.Errorf("未找到有效的用户邮箱")
	}

	var emails []string
	for _, u := range users {
		if u.Email != "" {
			emails = append(emails, u.Email)
		}
	}

	appLogger.Info("找到接收邮箱", zap.Strings("emails", emails), zap.Int("count", len(emails)))

	// 构建邮件主题
	subject := "SreHub 告警通知"
	if strings.Contains(msg, "恢复") || strings.Contains(msg, "✅") {
		subject = "SreHub 恢复通知"
	}

	// 构建邮件头
	from := fmt.Sprintf("%s <%s>", cfg.FromName, cfg.FromEmail)
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = strings.Join(emails, ", ")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// 构建邮件内容
	var mailContent string
	for k, v := range headers {
		mailContent += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	mailContent += "\r\n" + msg

	// 发送邮件
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)
	var err error
	if cfg.SMTPPort == 465 {
		// 端口 465 使用直接 TLS 连接
		err = sendEmailWithTLS(addr, cfg.SMTPHost, cfg.SMTPUser, cfg.SMTPPassword, cfg.FromEmail, emails, []byte(mailContent))
	} else {
		// 端口 25/587 使用 STARTTLS
		err = sendEmailWithSTARTTLS(addr, cfg.SMTPHost, cfg.SMTPUser, cfg.SMTPPassword, cfg.FromEmail, emails, []byte(mailContent))
	}

	if err != nil {
		appLogger.Error("邮件发送失败",
			zap.String("smtpHost", cfg.SMTPHost),
			zap.Int("smtpPort", cfg.SMTPPort),
			zap.Strings("emails", emails),
			zap.Error(err))
		return err
	}

	appLogger.Info("邮件发送成功", zap.Strings("emails", emails), zap.Int("count", len(emails)))
	return nil
}

// sendEmailWithTLS 使用直接 TLS 连接发送邮件（端口 465）
func sendEmailWithTLS(addr, host, user, password, from string, to []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: false,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS 连接失败: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", user, password, host)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 认证失败: %w", err)
	}

	if err = client.Mail(from); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}

	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("设置收件人失败: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("获取数据写入器失败: %w", err)
	}
	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("关闭数据写入器失败: %w", err)
	}

	return client.Quit()
}

// sendEmailWithSTARTTLS 使用 STARTTLS 发送邮件（端口 25/587）
func sendEmailWithSTARTTLS(addr, host, user, password, from string, to []string, msg []byte) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("TCP 连接失败: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Close()

	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: false,
	}
	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS 升级失败: %w", err)
	}

	auth := smtp.PlainAuth("", user, password, host)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 认证失败: %w", err)
	}

	if err = client.Mail(from); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}

	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("设置收件人失败: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("获取数据写入器失败: %w", err)
	}
	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("关闭数据写入器失败: %w", err)
	}

	return client.Quit()
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

// sendAIAgentPatrolReport 发送巡检报告到 AI Agent
func sendAIAgentPatrolReport(configJSON, content string) error {
	var cfg struct {
		HookURL string `json:"hookUrl"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return err
	}
	if cfg.HookURL == "" {
		return nil
	}
	// content 已经是 JSON 格式，直接解析后发送
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		// 如果不是 JSON，则包装成简单格式
		payload = map[string]interface{}{
			"type":    "patrol_report",
			"content": content,
		}
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

// SendPatrolReport 发送巡检报告
func (s *NotifyService) SendPatrolReport(ctx context.Context, ch *biz.AlertNotifyChannel, content string, phones []string, userIDs []uint) {
	appLogger.Info("发送巡检报告",
		zap.String("channel", ch.Name),
		zap.String("type", ch.Type))

	var err error
	switch ch.Type {
	case "email":
		err = s.sendEmail(ctx, ch.Config, content, userIDs)
	case "wechat_work":
		err = sendWechatWork(ch.Config, content, phones, false)
	case "dingtalk":
		err = sendDingTalk(ch.Config, content, phones)
	case "sms":
		err = sendSMS(ch.Config, content, phones)
	case "phone":
		err = sendPhone(ch.Config, content, phones)
	case "ai_agent":
		if ch.AIHookEnabled {
			// AI Agent 通道需要构造一个临时的 AlertEvent 对象
			// 这里简化处理，直接发送 JSON 格式的内容
			err = sendAIAgentPatrolReport(ch.Config, content)
		}
	default:
		appLogger.Warn("巡检报告不支持该通道类型", zap.String("type", ch.Type))
		return
	}

	if err != nil {
		appLogger.Error("发送巡检报告失败",
			zap.String("channel", ch.Name),
			zap.Error(err))
	} else {
		appLogger.Info("发送巡检报告成功", zap.String("channel", ch.Name))
	}
}
