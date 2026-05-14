package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	alertdata "github.com/ydcloud-dy/opshub/internal/data/alert"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PatrolService 巡检服务
type PatrolService struct {
	db             *gorm.DB
	patrolRepo     *alertdata.PatrolRepo
	reportRepo     *alertdata.PatrolReportRepo
	eventRepo      *alertdata.EventRepo
	subRepo        *alertdata.SubscriptionRepo
	subRuleRepo    *alertdata.SubscriptionRuleRepo
	subChannelRepo *alertdata.SubscriptionChannelRepo
	subUserRepo    *alertdata.SubscriptionUserRepo
	channelRepo    *alertdata.ChannelRepo
	notifySvc      *NotifyService
}

func NewPatrolService(db *gorm.DB, notifySvc *NotifyService) *PatrolService {
	return &PatrolService{
		db:             db,
		patrolRepo:     alertdata.NewPatrolRepo(db),
		reportRepo:     alertdata.NewPatrolReportRepo(db),
		eventRepo:      alertdata.NewEventRepo(db),
		subRepo:        alertdata.NewSubscriptionRepo(db),
		subRuleRepo:    alertdata.NewSubscriptionRuleRepo(db),
		subChannelRepo: alertdata.NewSubscriptionChannelRepo(db),
		subUserRepo:    alertdata.NewSubscriptionUserRepo(db),
		channelRepo:    alertdata.NewChannelRepo(db),
		notifySvc:      notifySvc,
	}
}

// GetPatrolRepo 获取巡检配置仓储
func (s *PatrolService) GetPatrolRepo() *alertdata.PatrolRepo {
	return s.patrolRepo
}

// GetReportRepo 获取巡检报告仓储
func (s *PatrolService) GetReportRepo() *alertdata.PatrolReportRepo {
	return s.reportRepo
}

// Start 启动巡检服务（定时任务）
func (s *PatrolService) Start(ctx context.Context) {
	appLogger.Info("告警巡检服务启动")
	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			appLogger.Info("告警巡检服务停止")
			return
		case <-ticker.C:
			s.checkAndExecutePatrols(ctx)
		}
	}
}

// checkAndExecutePatrols 检查并执行到期的巡检任务
func (s *PatrolService) checkAndExecutePatrols(ctx context.Context) {
	now := time.Now()

	// 查询所有启用的巡检配置
	patrols, err := s.patrolRepo.ListEnabledPatrols(ctx)
	if err != nil {
		appLogger.Error("查询巡检配置失败", zap.Error(err))
		return
	}

	appLogger.Debug("检查巡检任务", zap.Int("启用的巡检数量", len(patrols)), zap.Time("当前时间", now))

	for _, patrol := range patrols {
		// 检查是否需要执行巡检
		if s.shouldExecutePatrol(patrol, now) {
			appLogger.Info("执行巡检任务",
				zap.Uint("patrolID", patrol.ID),
				zap.Uint("subscriptionRuleID", patrol.SubscriptionRuleID),
				zap.Int("巡检间隔", patrol.PatrolInterval))
			go s.executePatrol(context.Background(), patrol)
		} else {
			if patrol.NextPatrolAt != nil {
				appLogger.Debug("巡检未到期",
					zap.Uint("patrolID", patrol.ID),
					zap.Time("下次巡检时间", *patrol.NextPatrolAt),
					zap.Duration("距离下次巡检", patrol.NextPatrolAt.Sub(now)))
			}
		}
	}
}

// shouldExecutePatrol 判断是否需要执行巡检
func (s *PatrolService) shouldExecutePatrol(patrol *biz.AlertSubscriptionRulePatrol, now time.Time) bool {
	// 如果 NextPatrolAt 为空，说明是首次执行
	if patrol.NextPatrolAt == nil {
		return true
	}

	// 检查是否到达下次巡检时间
	return now.After(*patrol.NextPatrolAt) || now.Equal(*patrol.NextPatrolAt)
}

// executePatrol 执行巡检任务（基于单个推送规则）
func (s *PatrolService) executePatrol(ctx context.Context, patrol *biz.AlertSubscriptionRulePatrol) {
	now := time.Now()

	// 1. 查询订阅规则
	subRule, err := s.subRuleRepo.GetByID(ctx, patrol.SubscriptionRuleID)
	if err != nil {
		appLogger.Warn("订阅规则不存在", zap.Uint("subscriptionRuleID", patrol.SubscriptionRuleID))
		s.updateNextPatrolTime(ctx, patrol, now)
		return
	}

	// 2. 查询订阅信息
	sub, err := s.subRepo.GetByID(ctx, subRule.SubscriptionID)
	if err != nil || !sub.Enabled {
		appLogger.Warn("订阅不存在或未启用", zap.Uint("subscriptionID", subRule.SubscriptionID))
		s.updateNextPatrolTime(ctx, patrol, now)
		return
	}

	// 3. 收集符合该推送规则的告警
	var allAlerts []*biz.AlertEvent
	if subRule.RuleID == 0 {
		// rule_id=0 表示订阅所有规则
		allAlerts, err = s.eventRepo.ListAllFiring(ctx)
	} else {
		allAlerts, err = s.eventRepo.ListFiringByRuleID(ctx, subRule.RuleID)
	}

	if err != nil {
		appLogger.Error("查询告警失败", zap.Error(err))
		s.updateNextPatrolTime(ctx, patrol, now)
		return
	}

	// 4. 应用该推送规则的过滤条件
	filteredAlerts := s.filterAlertsByRule(ctx, allAlerts, subRule, patrol, now)

	// 5. 如果包含已恢复告警，查询并添加
	var resolvedAlerts []*biz.AlertEvent
	if patrol.IncludeResolved && patrol.TimeRange > 0 {
		since := now.Add(-time.Duration(patrol.TimeRange) * time.Second)
		if subRule.RuleID == 0 {
			resolvedAlerts, _ = s.eventRepo.ListResolvedSince(ctx, since)
		} else {
			resolvedAlerts, _ = s.eventRepo.ListResolvedByRuleIDsSince(ctx, []uint{subRule.RuleID}, since)
		}
		resolvedAlerts = s.filterAlertsByRule(ctx, resolvedAlerts, subRule, patrol, now)
	}

	// 6. 生成巡检报告
	report := s.generatePatrolReport(ctx, patrol, subRule, filteredAlerts, resolvedAlerts)

	appLogger.Info("生成巡检报告",
		zap.Uint("patrolID", patrol.ID),
		zap.Int("活跃告警数", report.FiringCount),
		zap.Int("已恢复告警数", report.ResolvedCount))

	// 7. 保存报告
	if err := s.reportRepo.Create(ctx, report); err != nil {
		appLogger.Error("保存巡检报告失败", zap.Error(err))
	}

	// 8. 推送报告
	shouldSend := s.shouldSendReport(patrol, report)
	appLogger.Info("判断是否推送报告",
		zap.Uint("patrolID", patrol.ID),
		zap.Bool("是否推送", shouldSend),
		zap.String("推送模式", patrol.SendMode),
		zap.Int("活跃告警数", report.FiringCount))

	if shouldSend {
		s.sendPatrolReport(ctx, patrol, sub, subRule, report, filteredAlerts, resolvedAlerts)
	}

	// 9. 更新下次巡检时间
	s.updateNextPatrolTime(ctx, patrol, now)
}

// filterAlertsByRule 根据推送规则过滤告警
func (s *PatrolService) filterAlertsByRule(ctx context.Context, alerts []*biz.AlertEvent, subRule *biz.AlertSubscriptionRule, patrol *biz.AlertSubscriptionRulePatrol, now time.Time) []*biz.AlertEvent {
	filtered := make([]*biz.AlertEvent, 0)

	for _, alert := range alerts {
		// 检查时间范围
		if patrol.TimeRange > 0 {
			alertAge := now.Sub(alert.FiredAt).Seconds()
			if alertAge > float64(patrol.TimeRange) {
				continue
			}
		}

		// 检查是否匹配该推送规则
		if s.matchSubscriptionRule(alert, subRule, now) {
			filtered = append(filtered, alert)
		}
	}

	// 限制数量
	if len(filtered) > patrol.MaxAlertsPerReport {
		// 按触发时间排序，取最新的
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].FiredAt.After(filtered[j].FiredAt)
		})
		filtered = filtered[:patrol.MaxAlertsPerReport]
	}

	return filtered
}

// matchSubscriptionRule 检查告警是否匹配订阅规则
func (s *PatrolService) matchSubscriptionRule(alert *biz.AlertEvent, sr *biz.AlertSubscriptionRule, now time.Time) bool {
	// 检查生效时间
	if !isInTimeRanges(sr.TimeRanges, now) {
		return false
	}

	// 检查级别过滤
	if sevList := parseSeverityList(sr.Severities); len(sevList) > 0 {
		matched := false
		for _, s := range sevList {
			if s == alert.Severity {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查数据源过滤
	if !MatchSubscriptionDataSource(alert.Labels, sr.DataSourceIDs) {
		return false
	}

	// 检查标签匹配器
	if !MatchSubscriptionLabels(alert.Labels, sr.LabelMatchers) {
		return false
	}

	return true
}

// generatePatrolReport 生成巡检报告
func (s *PatrolService) generatePatrolReport(ctx context.Context, patrol *biz.AlertSubscriptionRulePatrol, subRule *biz.AlertSubscriptionRule, firingAlerts, resolvedAlerts []*biz.AlertEvent) *biz.AlertPatrolReport {
	report := &biz.AlertPatrolReport{
		SubscriptionID:     subRule.SubscriptionID,
		SubscriptionRuleID: patrol.SubscriptionRuleID,
		PatrolConfigID:     patrol.ID,
		FiringCount:        len(firingAlerts),
		ResolvedCount:      len(resolvedAlerts),
	}

	// 统计各级别告警数量
	for _, alert := range firingAlerts {
		switch alert.Severity {
		case "critical":
			report.CriticalCount++
		case "major":
			report.MajorCount++
		case "minor":
			report.MinorCount++
		case "warning":
			report.WarningCount++
		}
	}

	// 生成告警ID列表
	alertIDs := make([]uint, 0, len(firingAlerts))
	for _, alert := range firingAlerts {
		alertIDs = append(alertIDs, alert.ID)
	}
	alertIDsJSON, _ := json.Marshal(alertIDs)
	report.AlertIDs = string(alertIDsJSON)

	// 生成告警摘要（按配置的分组维度）
	summary := s.generateAlertSummary(firingAlerts, patrol.GroupBy)
	summaryJSON, _ := json.Marshal(summary)
	report.AlertSummary = string(summaryJSON)

	return report
}

// generateAlertSummary 生成告警摘要
func (s *PatrolService) generateAlertSummary(alerts []*biz.AlertEvent, groupBy string) []biz.AlertSummaryItem {
	summaryMap := make(map[string]*biz.AlertSummaryItem)

	for _, alert := range alerts {
		var key string
		switch groupBy {
		case "severity":
			key = alert.Severity
		case "ruleName":
			key = alert.RuleName
		case "assetGroup":
			var labels map[string]string
			json.Unmarshal([]byte(alert.Labels), &labels)
			key = labels["asset_group"]
			if key == "" {
				key = "未分组"
			}
		default:
			key = alert.Severity
		}

		if item, exists := summaryMap[key]; exists {
			item.Count++
			if alert.FiredAt.Before(item.OldestFiredAt) {
				item.OldestFiredAt = alert.FiredAt
			}
			item.LatestValue = alert.Value
		} else {
			summaryMap[key] = &biz.AlertSummaryItem{
				RuleName:      alert.RuleName,
				Severity:      alert.Severity,
				Count:         1,
				OldestFiredAt: alert.FiredAt,
				LatestValue:   alert.Value,
			}
		}
	}

	// 转换为切片并排序
	summary := make([]biz.AlertSummaryItem, 0, len(summaryMap))
	for _, item := range summaryMap {
		summary = append(summary, *item)
	}

	// 按数量降序排序
	sort.Slice(summary, func(i, j int) bool {
		return summary[i].Count > summary[j].Count
	})

	return summary
}

// shouldSendReport 判断是否应该推送报告
func (s *PatrolService) shouldSendReport(patrol *biz.AlertSubscriptionRulePatrol, report *biz.AlertPatrolReport) bool {
	if patrol.SendMode == "always" {
		return true
	}
	// only_firing 模式：仅有告警时推送
	return report.FiringCount > 0
}

// sendPatrolReport 推送巡检报告
func (s *PatrolService) sendPatrolReport(ctx context.Context, patrol *biz.AlertSubscriptionRulePatrol, sub *biz.AlertSubscription, subRule *biz.AlertSubscriptionRule, report *biz.AlertPatrolReport, firingAlerts, resolvedAlerts []*biz.AlertEvent) {
	// 使用推送规则配置的通道和用户
	var channelIDs []uint
	var userIDs []uint

	// 解析推送规则的通道
	if subRule.ChannelIDs != "" && subRule.ChannelIDs != "[]" {
		json.Unmarshal([]byte(subRule.ChannelIDs), &channelIDs)
	}

	// 解析推送规则的用户
	if subRule.UserIDs != "" && subRule.UserIDs != "[]" {
		json.Unmarshal([]byte(subRule.UserIDs), &userIDs)
	}

	appLogger.Info("准备推送巡检报告",
		zap.Uint("patrolID", patrol.ID),
		zap.Uints("通道IDs", channelIDs),
		zap.Uints("用户IDs", userIDs))

	// 获取用户手机号
	var phones []string
	if len(userIDs) > 0 {
		phones = patrolGetUserPhones(ctx, s.db, userIDs)
	}

	// 获取通道列表
	channels, _ := s.channelRepo.ListByIDs(ctx, channelIDs)

	appLogger.Info("获取到通道列表",
		zap.Uint("patrolID", patrol.ID),
		zap.Int("通道数量", len(channels)))

	// 发送到各个通道（根据通道类型生成不同格式的内容）
	var channelResults []map[string]interface{}
	for _, ch := range channels {
		if !ch.Enabled {
			appLogger.Warn("通道未启用，跳过",
				zap.Uint("patrolID", patrol.ID),
				zap.String("通道名称", ch.Name))
			continue
		}
		channelResults = append(channelResults, map[string]interface{}{
			"id":   ch.ID,
			"name": ch.Name,
			"type": ch.Type,
		})

		// 根据通道类型生成不同格式的报告内容
		content := s.buildPatrolReportContentForChannel(ch.Type, patrol, sub, report, firingAlerts, resolvedAlerts)

		appLogger.Info("发送巡检报告到通道",
			zap.Uint("patrolID", patrol.ID),
			zap.String("通道名称", ch.Name),
			zap.String("通道类型", ch.Type))
		go s.notifySvc.SendPatrolReport(ctx, ch, content, phones, userIDs)
	}

	// 更新报告推送状态
	now := time.Now()
	report.Sent = true
	report.SentAt = &now
	channelsJSON, _ := json.Marshal(channelResults)
	usersJSON, _ := json.Marshal(userIDs)
	report.Channels = string(channelsJSON)
	report.Users = string(usersJSON)
	s.reportRepo.Update(ctx, report)

	appLogger.Info("巡检报告推送完成",
		zap.Uint("patrolID", patrol.ID),
		zap.Int("推送通道数", len(channelResults)))
}

// buildPatrolReportContent 构建巡检报告内容
// buildPatrolReportContentForChannel 根据通道类型生成不同格式的巡检报告内容
func (s *PatrolService) buildPatrolReportContentForChannel(channelType string, patrol *biz.AlertSubscriptionRulePatrol, sub *biz.AlertSubscription, report *biz.AlertPatrolReport, firingAlerts, resolvedAlerts []*biz.AlertEvent) string {
	switch channelType {
	case "wechat_work", "dingtalk":
		return s.buildMarkdownPatrolReport(patrol, sub, report, firingAlerts, resolvedAlerts)
	case "sms", "phone":
		return s.buildTextPatrolReport(patrol, sub, report, firingAlerts, resolvedAlerts)
	case "email":
		return s.buildHTMLPatrolReport(patrol, sub, report, firingAlerts, resolvedAlerts)
	case "ai_agent":
		return s.buildJSONPatrolReport(patrol, sub, report, firingAlerts, resolvedAlerts)
	default:
		return s.buildMarkdownPatrolReport(patrol, sub, report, firingAlerts, resolvedAlerts)
	}
}

// buildMarkdownPatrolReport 构建 Markdown 格式的巡检报告（企业微信/钉钉）
func (s *PatrolService) buildMarkdownPatrolReport(patrol *biz.AlertSubscriptionRulePatrol, sub *biz.AlertSubscription, report *biz.AlertPatrolReport, firingAlerts, resolvedAlerts []*biz.AlertEvent) string {
	var content strings.Builder

	// 报告标题
	if report.FiringCount == 0 {
		content.WriteString("## ✅ 【告警巡检报告】一切正常\n\n")
	} else {
		content.WriteString(fmt.Sprintf("## 🚨 【告警巡检报告】发现 %d 条活跃告警\n\n", report.FiringCount))
	}

	// 订阅信息
	content.WriteString(fmt.Sprintf("**订阅名称**：%s\n\n", sub.Name))
	content.WriteString(fmt.Sprintf("**巡检时间**：%s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 统计信息（使用颜色标签）
	content.WriteString("---\n\n")
	content.WriteString("### 📊 统计概览\n\n")
	content.WriteString(fmt.Sprintf("<font color=\"warning\">🔴 紧急(P1)：%d 条</font>\n\n", report.CriticalCount))
	content.WriteString(fmt.Sprintf("<font color=\"warning\">🟠 严重(P2)：%d 条</font>\n\n", report.MajorCount))
	content.WriteString(fmt.Sprintf("<font color=\"info\">🟡 一般(P3)：%d 条</font>\n\n", report.MinorCount))
	content.WriteString(fmt.Sprintf("<font color=\"info\">🔵 提示(P4)：%d 条</font>\n\n", report.WarningCount))
	if patrol.IncludeResolved {
		content.WriteString(fmt.Sprintf("✅ 已恢复：%d 条\n\n", report.ResolvedCount))
	}

	// 如果有告警，显示详情
	if report.FiringCount > 0 {
		// 解析告警摘要
		var summary []biz.AlertSummaryItem
		json.Unmarshal([]byte(report.AlertSummary), &summary)

		if patrol.ReportStyle == "summary" {
			// 摘要模式：只显示分组统计
			content.WriteString("---\n\n")
			content.WriteString("### 📋 告警摘要\n\n")
			for i, item := range summary {
				if i >= 10 { // 最多显示10条
					break
				}
				duration := time.Since(item.OldestFiredAt)
				severityIcon := getSeverityIcon(item.Severity)
				content.WriteString(fmt.Sprintf("%d. %s **%s** (%s) - %d条\n\n", i+1, severityIcon, item.RuleName, patrolSeverityLabel(item.Severity), item.Count))
				content.WriteString(fmt.Sprintf("   > 最早触发：%s前\n\n", formatDuration(duration)))
			}
		} else {
			// 详细模式：显示每条告警
			content.WriteString("---\n\n")
			content.WriteString("### 📋 告警详情\n\n")
			for i, alert := range firingAlerts {
				if i >= 20 { // 最多显示20条
					content.WriteString(fmt.Sprintf("\n> ... 还有 %d 条告警未显示\n\n", len(firingAlerts)-20))
					break
				}
				duration := time.Since(alert.FiredAt)
				severityIcon := getSeverityIcon(alert.Severity)
				content.WriteString(fmt.Sprintf("**%d. %s %s**\n\n", i+1, severityIcon, alert.RuleName))
				content.WriteString(fmt.Sprintf("- 级别：%s\n", patrolSeverityLabel(alert.Severity)))
				content.WriteString(fmt.Sprintf("- 当前值：%.2f\n", alert.Value))
				content.WriteString(fmt.Sprintf("- 持续时长：%s\n", formatDuration(duration)))

				// 解析 annotations
				var annotations map[string]string
				json.Unmarshal([]byte(alert.Annotations), &annotations)
				if desc, ok := annotations["description"]; ok && desc != "" {
					content.WriteString(fmt.Sprintf("- 描述：%s\n", desc))
				}
				content.WriteString("\n")
			}
		}
	}

	content.WriteString("---\n\n")
	content.WriteString("### 💡 温馨提示\n\n")
	if report.FiringCount > 0 {
		content.WriteString("> 请及时处理活跃告警，避免影响业务运行。\n")
	} else {
		content.WriteString("> 当前无活跃告警，系统运行正常。\n")
	}

	return content.String()
}

// buildTextPatrolReport 构建纯文本格式的巡检报告（短信/电话）
func (s *PatrolService) buildTextPatrolReport(patrol *biz.AlertSubscriptionRulePatrol, sub *biz.AlertSubscription, report *biz.AlertPatrolReport, firingAlerts, resolvedAlerts []*biz.AlertEvent) string {
	var content strings.Builder

	// 精简标题
	if report.FiringCount == 0 {
		content.WriteString("【告警巡检】一切正常\n")
	} else {
		content.WriteString(fmt.Sprintf("【告警巡检】发现%d条活跃告警\n", report.FiringCount))
	}

	// 统计信息（精简）
	content.WriteString(fmt.Sprintf("紧急P1:%d 严重P2:%d 一般P3:%d 提示P4:%d\n",
		report.CriticalCount, report.MajorCount, report.MinorCount, report.WarningCount))

	// 只显示前3条最严重的告警
	if report.FiringCount > 0 && len(firingAlerts) > 0 {
		content.WriteString("告警详情:\n")
		count := 3
		if len(firingAlerts) < count {
			count = len(firingAlerts)
		}
		for i := 0; i < count; i++ {
			alert := firingAlerts[i]
			duration := time.Since(alert.FiredAt)
			content.WriteString(fmt.Sprintf("%d.%s(%s) 持续%s\n",
				i+1, alert.RuleName, patrolSeverityLabel(alert.Severity), formatDuration(duration)))
		}
		if len(firingAlerts) > 3 {
			content.WriteString(fmt.Sprintf("...还有%d条\n", len(firingAlerts)-3))
		}
	}

	return content.String()
}

// buildHTMLPatrolReport 构建 HTML 格式的巡检报告（邮件）
func (s *PatrolService) buildHTMLPatrolReport(patrol *biz.AlertSubscriptionRulePatrol, sub *biz.AlertSubscription, report *biz.AlertPatrolReport, firingAlerts, resolvedAlerts []*biz.AlertEvent) string {
	var content strings.Builder

	content.WriteString("<html><head><style>")
	content.WriteString("body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }")
	content.WriteString("h2 { color: #2c3e50; border-bottom: 2px solid #3498db; padding-bottom: 10px; }")
	content.WriteString("table { width: 100%; border-collapse: collapse; margin: 20px 0; }")
	content.WriteString("th, td { padding: 12px; text-align: left; border: 1px solid #ddd; }")
	content.WriteString("th { background-color: #3498db; color: white; }")
	content.WriteString("tr:nth-child(even) { background-color: #f2f2f2; }")
	content.WriteString(".critical { color: #e74c3c; font-weight: bold; }")
	content.WriteString(".major { color: #e67e22; font-weight: bold; }")
	content.WriteString(".minor { color: #f39c12; }")
	content.WriteString(".warning { color: #3498db; }")
	content.WriteString(".stats { background-color: #ecf0f1; padding: 15px; border-radius: 5px; margin: 20px 0; }")
	content.WriteString("</style></head><body>")

	// 标题
	if report.FiringCount == 0 {
		content.WriteString("<h2>✅ 告警巡检报告 - 一切正常</h2>")
	} else {
		content.WriteString(fmt.Sprintf("<h2>🚨 告警巡检报告 - 发现 %d 条活跃告警</h2>", report.FiringCount))
	}

	// 基本信息
	content.WriteString(fmt.Sprintf("<p><strong>订阅名称：</strong>%s</p>", sub.Name))
	content.WriteString(fmt.Sprintf("<p><strong>巡检时间：</strong>%s</p>", time.Now().Format("2006-01-02 15:04:05")))

	// 统计信息
	content.WriteString("<div class='stats'>")
	content.WriteString("<h3>📊 统计概览</h3>")
	content.WriteString("<table>")
	content.WriteString("<tr><th>告警级别</th><th>数量</th></tr>")
	content.WriteString(fmt.Sprintf("<tr><td class='critical'>🔴 紧急(P1)</td><td>%d</td></tr>", report.CriticalCount))
	content.WriteString(fmt.Sprintf("<tr><td class='major'>🟠 严重(P2)</td><td>%d</td></tr>", report.MajorCount))
	content.WriteString(fmt.Sprintf("<tr><td class='minor'>🟡 一般(P3)</td><td>%d</td></tr>", report.MinorCount))
	content.WriteString(fmt.Sprintf("<tr><td class='warning'>🔵 提示(P4)</td><td>%d</td></tr>", report.WarningCount))
	if patrol.IncludeResolved {
		content.WriteString(fmt.Sprintf("<tr><td>✅ 已恢复</td><td>%d</td></tr>", report.ResolvedCount))
	}
	content.WriteString("</table>")
	content.WriteString("</div>")

	// 告警详情
	if report.FiringCount > 0 && len(firingAlerts) > 0 {
		content.WriteString("<h3>📋 告警详情</h3>")
		content.WriteString("<table>")
		content.WriteString("<tr><th>序号</th><th>规则名称</th><th>级别</th><th>当前值</th><th>持续时长</th><th>描述</th></tr>")

		count := 20
		if len(firingAlerts) < count {
			count = len(firingAlerts)
		}
		for i := 0; i < count; i++ {
			alert := firingAlerts[i]
			duration := time.Since(alert.FiredAt)
			severityClass := getSeverityClass(alert.Severity)

			var annotations map[string]string
			json.Unmarshal([]byte(alert.Annotations), &annotations)
			desc := annotations["description"]
			if desc == "" {
				desc = "-"
			}

			content.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td><td class='%s'>%s</td><td>%.2f</td><td>%s</td><td>%s</td></tr>",
				i+1, alert.RuleName, severityClass, patrolSeverityLabel(alert.Severity), alert.Value, formatDuration(duration), desc))
		}

		if len(firingAlerts) > 20 {
			content.WriteString(fmt.Sprintf("<tr><td colspan='6' style='text-align:center;'>... 还有 %d 条告警未显示</td></tr>", len(firingAlerts)-20))
		}
		content.WriteString("</table>")
	}

	// 温馨提示
	content.WriteString("<div style='margin-top: 30px; padding: 15px; background-color: #d5f4e6; border-left: 4px solid #27ae60;'>")
	content.WriteString("<h3>💡 温馨提示</h3>")
	if report.FiringCount > 0 {
		content.WriteString("<p>请及时处理活跃告警，避免影响业务运行。</p>")
	} else {
		content.WriteString("<p>当前无活跃告警，系统运行正常。</p>")
	}
	content.WriteString("</div>")

	content.WriteString("</body></html>")
	return content.String()
}

// buildJSONPatrolReport 构建 JSON 格式的巡检报告（AI Agent）
func (s *PatrolService) buildJSONPatrolReport(patrol *biz.AlertSubscriptionRulePatrol, sub *biz.AlertSubscription, report *biz.AlertPatrolReport, firingAlerts, resolvedAlerts []*biz.AlertEvent) string {
	data := map[string]interface{}{
		"type":           "patrol_report",
		"subscription":   sub.Name,
		"patrol_time":    time.Now().Format("2006-01-02 15:04:05"),
		"total_firing":   report.FiringCount,
		"total_resolved": report.ResolvedCount,
		"statistics": map[string]int{
			"critical": report.CriticalCount,
			"major":    report.MajorCount,
			"minor":    report.MinorCount,
			"warning":  report.WarningCount,
		},
		"alerts": []map[string]interface{}{},
	}

	// 添加告警详情
	alerts := []map[string]interface{}{}
	for _, alert := range firingAlerts {
		duration := time.Since(alert.FiredAt)
		var annotations map[string]string
		json.Unmarshal([]byte(alert.Annotations), &annotations)

		alerts = append(alerts, map[string]interface{}{
			"rule_name":   alert.RuleName,
			"severity":    alert.Severity,
			"value":       alert.Value,
			"duration":    formatDuration(duration),
			"description": annotations["description"],
			"fired_at":    alert.FiredAt.Format("2006-01-02 15:04:05"),
		})
	}
	data["alerts"] = alerts

	jsonBytes, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonBytes)
}

// getSeverityIcon 获取告警级别图标
func getSeverityIcon(severity string) string {
	switch severity {
	case "critical":
		return "🔴"
	case "major":
		return "🟠"
	case "minor":
		return "🟡"
	case "warning":
		return "🔵"
	default:
		return "⚪"
	}
}

// getSeverityClass 获取告警级别 CSS 类名
func getSeverityClass(severity string) string {
	switch severity {
	case "critical":
		return "critical"
	case "major":
		return "major"
	case "minor":
		return "minor"
	case "warning":
		return "warning"
	default:
		return ""
	}
}

// updateNextPatrolTime 更新下次巡检时间
func (s *PatrolService) updateNextPatrolTime(ctx context.Context, patrol *biz.AlertSubscriptionRulePatrol, now time.Time) {
	var nextTime time.Time

	if patrol.PatrolMode == "fixed" {
		// 固定时间模式
		nextTime = s.calculateNextFixedTime(patrol.PatrolTimes, now)
	} else {
		// 间隔模式
		nextTime = now.Add(time.Duration(patrol.PatrolInterval) * time.Second)
	}

	appLogger.Info("更新巡检时间",
		zap.Uint("patrolID", patrol.ID),
		zap.Time("当前时间", now),
		zap.Time("下次巡检时间", nextTime),
		zap.Duration("间隔", nextTime.Sub(now)))

	if err := s.patrolRepo.UpdateNextPatrolTime(ctx, patrol.ID, nextTime); err != nil {
		appLogger.Error("更新巡检时间失败", zap.Uint("patrolID", patrol.ID), zap.Error(err))
	}
}

// calculateNextFixedTime 计算下次固定时间
func (s *PatrolService) calculateNextFixedTime(timesJSON string, now time.Time) time.Time {
	if timesJSON == "" || timesJSON == "[]" {
		// 默认每天9点
		return s.getNextTimePoint(now, "09:00")
	}

	var times []string
	if err := json.Unmarshal([]byte(timesJSON), &times); err != nil || len(times) == 0 {
		return s.getNextTimePoint(now, "09:00")
	}

	// 找到下一个时间点
	for _, t := range times {
		nextTime := s.getNextTimePoint(now, t)
		if nextTime.After(now) {
			return nextTime
		}
	}

	// 如果今天所有时间点都过了，返回明天的第一个时间点
	return s.getNextTimePoint(now.AddDate(0, 0, 1), times[0])
}

// getNextTimePoint 获取下一个时间点
func (s *PatrolService) getNextTimePoint(base time.Time, timeStr string) time.Time {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return base.Add(1 * time.Hour)
	}

	hour := 0
	minute := 0
	fmt.Sscanf(parts[0], "%d", &hour)
	fmt.Sscanf(parts[1], "%d", &minute)

	return time.Date(base.Year(), base.Month(), base.Day(), hour, minute, 0, 0, base.Location())
}

// ExecuteManually 手动执行巡检
func (s *PatrolService) ExecuteManually(ctx context.Context, subscriptionRuleID uint) error {
	patrol, err := s.patrolRepo.GetBySubscriptionRuleID(ctx, subscriptionRuleID)
	if err != nil {
		// 如果配置不存在，使用默认配置执行一次
		appLogger.Info("巡检配置不存在，使用默认配置执行", zap.Uint("subscriptionRuleID", subscriptionRuleID))
		patrol = &biz.AlertSubscriptionRulePatrol{
			SubscriptionRuleID: subscriptionRuleID,
			Enabled:            true,
			PatrolMode:         "interval",
			PatrolInterval:     3600,
			IncludeResolved:    false,
			TimeRange:          0,
			MaxAlertsPerReport: 100,
			SendMode:           "always",
			ReportStyle:        "detailed",
			GroupBy:            "severity",
		}
	}

	appLogger.Info("手动执行巡检", zap.Uint("subscriptionRuleID", subscriptionRuleID), zap.Uint("patrolID", patrol.ID))
	go s.executePatrol(context.Background(), patrol)
	return nil
}

// formatDuration 格式化持续时间
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 24 {
		days := hours / 24
		hours = hours % 24
		return fmt.Sprintf("%d天%d小时", days, hours)
	}

	if hours > 0 {
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	}
	return fmt.Sprintf("%d分钟", minutes)
}

// patrolSeverityLabel 告警级别标签（巡检专用）
func patrolSeverityLabel(severity string) string {
	switch severity {
	case "critical":
		return "紧急(P1)"
	case "major":
		return "严重(P2)"
	case "minor":
		return "一般(P3)"
	case "warning":
		return "提示(P4)"
	default:
		return severity
	}
}

// patrolGetUserPhones 获取用户手机号（巡检专用）
func patrolGetUserPhones(ctx context.Context, db *gorm.DB, userIDs []uint) []string {
	for _, uid := range userIDs {
		if uid == 0 {
			return nil // nil=@all
		}
	}
	type phoneRow struct{ Phone string }
	var rows []phoneRow
	db.WithContext(ctx).Table("sys_user").Select("phone").Where("id IN ? AND phone != ''", userIDs).Scan(&rows)
	var phones []string
	for _, r := range rows {
		if r.Phone != "" {
			phones = append(phones, r.Phone)
		}
	}
	return phones
}
