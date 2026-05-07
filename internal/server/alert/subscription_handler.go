package alert

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	alertsvc "github.com/ydcloud-dy/opshub/internal/service/alert"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// subscriptionRequest 创建/更新订阅的请求体
type subscriptionRequest struct {
	biz.AlertSubscription
	Rules      []subscriptionRuleReq `json:"rules"`
	ChannelIDs []uint                `json:"channelIds"`
	UserIDs    []uint                `json:"userIds"`
}

type subscriptionRuleReq struct {
	RuleID        uint            `json:"ruleId"`
	TimeRanges    json.RawMessage `json:"timeRanges"`    // JSON array of TimeRange
	Severities    json.RawMessage `json:"severities"`    // JSON array of severity strings
	ChannelIDs    json.RawMessage `json:"channelIds"`    // JSON array of channel IDs
	UserIDs       json.RawMessage `json:"userIds"`       // JSON array of user IDs
	DataSourceIDs json.RawMessage `json:"dataSourceIds"` // JSON array of data source IDs
	LabelMatchers json.RawMessage `json:"labelMatchers"` // JSON array of label matchers
}

// subscriptionVO 订阅任务详情视图
type subscriptionVO struct {
	biz.AlertSubscription
	Rules      []*biz.AlertSubscriptionRule    `json:"rules"`
	Channels   []*biz.AlertSubscriptionChannel `json:"channels"`
	Users      []*biz.AlertSubscriptionUser    `json:"users"`
	RuleCount  int                             `json:"ruleCount"`
	ChannelCount int                           `json:"channelCount"`
}

func (s *HTTPServer) listSubscriptions(c *gin.Context) {
	assetGroupID, _ := strconv.ParseUint(c.Query("assetGroupId"), 10, 64)
	list, err := s.subRepo.List(c.Request.Context(), uint(assetGroupID))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}
	// 附加规则数和通道数
	var vos []subscriptionVO
	for _, sub := range list {
		rules, _ := s.subRuleRepo.ListBySubscription(c.Request.Context(), sub.ID)

		// 统计所有推送规则中配置的唯一通道数
		channelSet := make(map[uint]bool)
		for _, rule := range rules {
			// 解析 channel_ids JSON 数组
			if rule.ChannelIDs != "" && rule.ChannelIDs != "[]" && rule.ChannelIDs != "null" {
				var channelIDs []uint
				if err := json.Unmarshal([]byte(rule.ChannelIDs), &channelIDs); err == nil {
					for _, chID := range channelIDs {
						channelSet[chID] = true
					}
				}
			}
		}

		vos = append(vos, subscriptionVO{
			AlertSubscription: *sub,
			RuleCount:         len(rules),
			ChannelCount:      len(channelSet),
		})
	}
	response.Success(c, vos)
}

func (s *HTTPServer) createSubscription(c *gin.Context) {
	var req subscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	// 校验配置
	if err := s.validateSubscriptionRequest(c, &req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.subRepo.Create(c.Request.Context(), &req.AlertSubscription); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败")
		return
	}
	s.saveSubRelations(c, req.AlertSubscription.ID, req.Rules, req.ChannelIDs, req.UserIDs)
	response.Success(c, req.AlertSubscription)
}

func (s *HTTPServer) getSubscription(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	sub, err := s.subRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "订阅任务不存在")
		return
	}
	rules, _ := s.subRuleRepo.ListBySubscription(c.Request.Context(), uint(id))
	channels, _ := s.subChannelRepo.ListBySubscription(c.Request.Context(), uint(id))
	users, _ := s.subUserRepo.ListBySubscription(c.Request.Context(), uint(id))
	vo := subscriptionVO{
		AlertSubscription: *sub,
		Rules:             rules,
		Channels:          channels,
		Users:             users,
		RuleCount:         len(rules),
		ChannelCount:      len(channels),
	}
	response.Success(c, vo)
}

func (s *HTTPServer) updateSubscription(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req subscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	// 校验配置
	if err := s.validateSubscriptionRequest(c, &req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	req.AlertSubscription.ID = uint(id)
	if err := s.subRepo.Update(c.Request.Context(), &req.AlertSubscription); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败")
		return
	}
	s.saveSubRelations(c, uint(id), req.Rules, req.ChannelIDs, req.UserIDs)
	response.Success(c, req.AlertSubscription)
}

func (s *HTTPServer) deleteSubscription(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.subRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}
	response.Success(c, nil)
}

// toggleSubscription 切换订阅启用状态（只更新 enabled 字段）
func (s *HTTPServer) toggleSubscription(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	// 解析请求体
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	// 查询订阅
	sub, err := s.subRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "订阅任务不存在")
		return
	}

	// 只更新 enabled 字段
	sub.Enabled = req.Enabled
	if err := s.subRepo.Update(c.Request.Context(), sub); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败")
		return
	}

	response.Success(c, sub)
}

func rawJSONStr(raw json.RawMessage, fallback string) string {
	if len(raw) == 0 {
		return fallback
	}
	var check interface{}
	if json.Unmarshal(raw, &check) == nil {
		return string(raw)
	}
	return fallback
}

func (s *HTTPServer) saveSubRelations(c *gin.Context, subID uint, rules []subscriptionRuleReq, channelIDs, userIDs []uint) {
	var subRules []*biz.AlertSubscriptionRule
	for _, r := range rules {
		subRules = append(subRules, &biz.AlertSubscriptionRule{
			SubscriptionID: subID,
			RuleID:         r.RuleID,
			TimeRanges:     rawJSONStr(r.TimeRanges, "[]"),
			Severities:     rawJSONStr(r.Severities, "[]"),
			ChannelIDs:     rawJSONStr(r.ChannelIDs, "[]"),
			UserIDs:        rawJSONStr(r.UserIDs, "[]"),
			DataSourceIDs:  rawJSONStr(r.DataSourceIDs, "[]"),
			LabelMatchers:  rawJSONStr(r.LabelMatchers, "[]"),
		})
	}
	_ = s.subRuleRepo.SetRules(c.Request.Context(), subID, subRules)
	// 保留全局 channelIds/userIds 向下兼容（当规则行未配置时使用）
	_ = s.subChannelRepo.SetChannels(c.Request.Context(), subID, channelIDs)
	_ = s.subUserRepo.SetUsers(c.Request.Context(), subID, userIDs)
}

// validateSubscriptionRequest 校验订阅请求
func (s *HTTPServer) validateSubscriptionRequest(c *gin.Context, req *subscriptionRequest) error {
	ctx := c.Request.Context()

	// 如果订阅已禁用，跳过通道校验（允许用户关闭订阅）
	// 但仍然校验数据源和标签匹配器的语法
	if !req.Enabled {
		// 校验数据源 ID 有效性
		for _, rule := range req.Rules {
			var dsIDs []uint
			if err := json.Unmarshal(rule.DataSourceIDs, &dsIDs); err == nil && len(dsIDs) > 0 {
				for _, dsID := range dsIDs {
					if _, err := s.dsRepo.GetByID(ctx, dsID); err != nil {
						return fmt.Errorf("数据源 ID %d 无效", dsID)
					}
				}
			}
		}

		// 校验标签匹配器语法
		for _, rule := range req.Rules {
			var matchers []map[string]string
			if err := json.Unmarshal(rule.LabelMatchers, &matchers); err == nil {
				for _, m := range matchers {
					op, ok := m["op"]
					if !ok {
						return fmt.Errorf("标签匹配器缺少 op 字段")
					}
					if op != "=" && op != "!=" && op != "=~" && op != "!~" {
						return fmt.Errorf("标签匹配器操作符无效: %s", op)
					}
					// 校验正则表达式语法
					if (op == "=~" || op == "!~") && m["value"] != "" {
						if _, err := regexp.Compile(m["value"]); err != nil {
							return fmt.Errorf("标签匹配器正则表达式无效: %s", m["value"])
						}
					}
				}
			}
		}

		return nil // 禁用的订阅不校验通道
	}

	// 以下是启用订阅的校验逻辑
	// 1. 校验至少有一个通道（rule-level 或 subscription-level）
	hasChannel := len(req.ChannelIDs) > 0
	if !hasChannel {
		for _, rule := range req.Rules {
			var channelIDs []uint
			if err := json.Unmarshal(rule.ChannelIDs, &channelIDs); err == nil && len(channelIDs) > 0 {
				hasChannel = true
				break
			}
		}
	}
	if !hasChannel {
		return fmt.Errorf("至少需要配置一个通知通道")
	}

	// 2. 校验通道 ID 有效性
	allChannelIDs := append([]uint{}, req.ChannelIDs...)
	for _, rule := range req.Rules {
		var channelIDs []uint
		if err := json.Unmarshal(rule.ChannelIDs, &channelIDs); err == nil {
			allChannelIDs = append(allChannelIDs, channelIDs...)
		}
	}
	if len(allChannelIDs) > 0 {
		channels, err := s.channelRepo.ListByIDs(ctx, allChannelIDs)
		if err != nil {
			return fmt.Errorf("查询通道失败")
		}
		if len(channels) != len(allChannelIDs) {
			return fmt.Errorf("部分通道 ID 无效")
		}
	}

	// 3. 校验数据源 ID 有效性
	for _, rule := range req.Rules {
		var dsIDs []uint
		if err := json.Unmarshal(rule.DataSourceIDs, &dsIDs); err == nil && len(dsIDs) > 0 {
			for _, dsID := range dsIDs {
				if _, err := s.dsRepo.GetByID(ctx, dsID); err != nil {
					return fmt.Errorf("数据源 ID %d 无效", dsID)
				}
			}
		}
	}

	// 4. 校验标签匹配器语法
	for _, rule := range req.Rules {
		var matchers []map[string]string
		if err := json.Unmarshal(rule.LabelMatchers, &matchers); err == nil {
			for _, m := range matchers {
				op, ok := m["op"]
				if !ok {
					return fmt.Errorf("标签匹配器缺少 op 字段")
				}
				if op != "=" && op != "!=" && op != "=~" && op != "!~" {
					return fmt.Errorf("标签匹配器操作符无效: %s", op)
				}
				// 校验正则表达式语法
				if (op == "=~" || op == "!~") && m["value"] != "" {
					if _, err := regexp.Compile(m["value"]); err != nil {
						return fmt.Errorf("标签匹配器正则表达式无效: %s", m["value"])
					}
				}
			}
		}
	}

	return nil
}

// testSubscriptionRequest 测试订阅请求
type testSubscriptionRequest struct {
	RuleID   uint              `json:"ruleId"`
	Severity string            `json:"severity"`
	Labels   map[string]string `json:"labels"`
	Value    float64           `json:"value"`
}

// testSubscriptionResponse 测试订阅响应
type testSubscriptionResponse struct {
	Matched       bool                   `json:"matched"`
	MatchResult   map[string]interface{} `json:"matchResult"`
	DenoiseResult map[string]interface{} `json:"denoiseResult"`
	NotifyResult  map[string]interface{} `json:"notifyResult"`
}

// testSubscription 测试订阅规则
func (s *HTTPServer) testSubscription(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req testSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()

	// 构造临时告警事件
	labelsJSON, _ := json.Marshal(req.Labels)
	event := &biz.AlertEvent{
		AlertRuleID: req.RuleID,
		Severity:    req.Severity,
		Labels:      string(labelsJSON),
		Value:       req.Value,
	}

	// 查询订阅规则
	subRules, err := s.subRuleRepo.ListBySubscription(ctx, uint(id))
	if err != nil || len(subRules) == 0 {
		response.ErrorCode(c, http.StatusNotFound, "未找到订阅规则")
		return
	}

	// 模拟匹配逻辑
	matchResult := make(map[string]interface{})
	denoiseResult := make(map[string]interface{})
	notifyResult := make(map[string]interface{})
	matched := true

	sr := subRules[0] // 使用第一个规则测试

	// 检查级别
	severityMatched := true
	if sevList := parseSeverityList(sr.Severities); len(sevList) > 0 {
		severityMatched = false
		for _, s := range sevList {
			if s == event.Severity {
				severityMatched = true
				break
			}
		}
	}
	matchResult["severity"] = severityMatched
	if !severityMatched {
		matched = false
	}

	// 检查数据源
	dsMatched := alertsvc.MatchSubscriptionDataSource(event.Labels, sr.DataSourceIDs)
	matchResult["dataSource"] = dsMatched
	if !dsMatched {
		matched = false
	}

	// 检查标签
	labelMatched := alertsvc.MatchSubscriptionLabels(event.Labels, sr.LabelMatchers)
	matchResult["labelMatchers"] = labelMatched
	if !labelMatched {
		matched = false
	}

	// 模拟降噪结果（测试模式不实际执行）
	denoiseResult["deduplicated"] = false
	denoiseResult["grouped"] = false
	denoiseResult["inhibited"] = false

	// 模拟推送结果
	var channelIDs []uint
	if perRuleCh := parseUintList(sr.ChannelIDs); len(perRuleCh) > 0 {
		channelIDs = perRuleCh
	} else {
		subChannels, _ := s.subChannelRepo.ListBySubscription(ctx, uint(id))
		for _, sc := range subChannels {
			channelIDs = append(channelIDs, sc.ChannelID)
		}
	}

	channels, _ := s.channelRepo.ListByIDs(ctx, channelIDs)
	var channelResults []map[string]interface{}
	for _, ch := range channels {
		if ch.Enabled {
			channelResults = append(channelResults, map[string]interface{}{
				"id":   ch.ID,
				"name": ch.Name,
				"type": ch.Type,
			})
		}
	}
	notifyResult["channels"] = channelResults

	resp := testSubscriptionResponse{
		Matched:       matched,
		MatchResult:   matchResult,
		DenoiseResult: denoiseResult,
		NotifyResult:  notifyResult,
	}

	response.Success(c, resp)
}

// getSubscriptionLogs 查询订阅日志
func (s *HTTPServer) getSubscriptionLogs(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	logs, total, err := s.subLogRepo.List(c.Request.Context(), uint(id), page, pageSize)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}

	response.Pagination(c, total, page, pageSize, logs)
}

// 辅助函数

func parseSeverityList(severitiesJSON string) []string {
	if severitiesJSON == "" || severitiesJSON == "[]" {
		return nil
	}
	var list []string
	json.Unmarshal([]byte(severitiesJSON), &list)
	return list
}

func parseUintList(uintListJSON string) []uint {
	if uintListJSON == "" || uintListJSON == "[]" {
		return nil
	}
	var list []uint
	json.Unmarshal([]byte(uintListJSON), &list)
	return list
}

