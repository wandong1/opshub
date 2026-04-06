package alert

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

func (s *HTTPServer) listActiveEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	assetGroupID, _ := strconv.ParseUint(c.Query("assetGroupId"), 10, 64)
	severity := c.Query("severity")
	keyword := c.Query("keyword")

	list, total, err := s.eventRepo.ListActive(c.Request.Context(), page, pageSize, uint(assetGroupID), severity, keyword)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}
	response.Success(c, gin.H{"total": total, "page": page, "pageSize": pageSize, "data": list})
}

func (s *HTTPServer) listHistoryEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	assetGroupID, _ := strconv.ParseUint(c.Query("assetGroupId"), 10, 64)
	severity := c.Query("severity")
	status := c.Query("status")
	resolveType := c.Query("resolveType")
	keyword := c.Query("keyword")

	var startTime, endTime *time.Time
	if v := c.Query("startTime"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			startTime = &t
		}
	}
	if v := c.Query("endTime"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			endTime = &t
		}
	}

	list, total, err := s.eventRepo.ListHistory(c.Request.Context(), page, pageSize, uint(assetGroupID), severity, status, resolveType, keyword, startTime, endTime)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}
	response.Success(c, gin.H{"total": total, "page": page, "pageSize": pageSize, "data": list})
}

func (s *HTTPServer) silenceEvent(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Duration string `json:"duration"` // e.g. "2h"
		Reason   string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	event, err := s.eventRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "告警事件不存在")
		return
	}
	dur, err := parseSilenceDuration(req.Duration)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "屏蔽时长格式错误")
		return
	}
	until := time.Now().Add(dur)
	event.Silenced = true
	event.SilenceUntil = &until
	event.SilenceReason = req.Reason
	if err := s.eventRepo.Update(c.Request.Context(), event); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "操作失败")
		return
	}
	response.Success(c, gin.H{"silenceUntil": until})
}

func (s *HTTPServer) handleEvent(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Note   string `json:"note"`
		UserID uint   `json:"userId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	event, err := s.eventRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "告警事件不存在")
		return
	}
	now := time.Now()
	// 只打「人工介入」标记，不改变 status（告警仍 firing）
	// 当指标恢复后由引擎自动 resolved，resolve_type 会被标记为 "manual_then_auto"
	event.ManualHandled = true
	event.HandledBy = &req.UserID
	event.HandledAt = &now
	event.HandledNote = req.Note
	if err := s.eventRepo.Update(c.Request.Context(), event); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "操作失败")
		return
	}
	response.Success(c, event)
}

func (s *HTTPServer) getEventStats(c *gin.Context) {
	stats, err := s.eventRepo.GetStats(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}
	response.Success(c, stats)
}

func (s *HTTPServer) getEventTrend(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	data, err := s.eventRepo.GetTrend(c.Request.Context(), days)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}
	response.Success(c, data)
}

func parseSilenceDuration(s string) (time.Duration, error) {
	if s == "" {
		return 1 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

// batchSilenceEvents 批量屏蔽告警
func (s *HTTPServer) batchSilenceEvents(c *gin.Context) {
	var req struct {
		EventIDs   []uint `json:"eventIds" binding:"required"`
		Type       string `json:"type" binding:"required"` // fixed / periodic
		Duration   string `json:"duration"`                // 固定时长："2h", "1d"
		TimeRanges string `json:"timeRanges"`              // 周期性：JSON 时间段
		EditLabels bool   `json:"editLabels"`              // 是否编辑标签
		Labels     string `json:"labels"`                  // 用户编辑后的标签
		Reason     string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	// 查询选中的告警事件
	var events []*biz.AlertEvent
	if err := s.eventRepo.GetDB().WithContext(c.Request.Context()).
		Where("id IN ?", req.EventIDs).Find(&events).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询告警失败")
		return
	}

	if len(events) == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "未找到告警事件")
		return
	}

	// 按 (Severity, RuleName, Labels) 分组
	type groupKey struct {
		Severity string
		RuleName string
		Labels   string
	}
	groups := make(map[groupKey][]*biz.AlertEvent)
	for _, event := range events {
		labels := event.Labels
		if req.EditLabels && req.Labels != "" {
			labels = req.Labels
		}
		key := groupKey{
			Severity: event.Severity,
			RuleName: event.RuleName,
			Labels:   labels,
		}
		groups[key] = append(groups[key], event)
	}

	// 为每个分组创建屏蔽规则
	now := time.Now()
	createdRules := 0
	for key, groupEvents := range groups {
		// 检查是否已存在相同的屏蔽规则
		existing, _ := s.silenceRuleRepo.FindMatchingRule(c.Request.Context(), key.Severity, key.RuleName, key.Labels)
		if existing != nil {
			// 已存在，跳过
			continue
		}

		rule := &biz.AlertSilenceRule{
			Severity:   key.Severity,
			RuleName:   key.RuleName,
			Labels:     key.Labels,
			Type:       req.Type,
			Reason:     req.Reason,
			CreatedBy:  0, // TODO: 从 JWT 获取用户 ID
			Enabled:    true,
		}

		if req.Type == "fixed" {
			rule.Duration = req.Duration
			dur, err := parseSilenceDuration(req.Duration)
			if err != nil {
				response.ErrorCode(c, http.StatusBadRequest, "屏蔽时长格式错误")
				return
			}
			until := now.Add(dur)
			rule.SilenceUntil = &until
		} else if req.Type == "periodic" {
			rule.TimeRanges = req.TimeRanges
		}

		if err := s.silenceRuleRepo.Create(c.Request.Context(), rule); err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "创建屏蔽规则失败")
			return
		}
		createdRules++

		// 更新事件的屏蔽状态
		for _, event := range groupEvents {
			event.Silenced = true
			now := time.Now()
			event.SilencedAt = &now
			if req.Type == "fixed" {
				event.SilenceUntil = rule.SilenceUntil
			} else if req.Type == "periodic" {
				event.SilenceTimeRanges = req.TimeRanges
			}
			event.SilenceReason = req.Reason
			event.SilenceType = req.Type
			s.eventRepo.Update(c.Request.Context(), event)
		}
	}

	response.Success(c, gin.H{
		"createdRules": createdRules,
		"totalGroups":  len(groups),
		"totalEvents":  len(events),
	})
}

// batchUnsilenceEvents 批量取消屏蔽
func (s *HTTPServer) batchUnsilenceEvents(c *gin.Context) {
	var req struct {
		EventIDs []uint `json:"eventIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	// 查询选中的告警事件
	var events []*biz.AlertEvent
	if err := s.eventRepo.GetDB().WithContext(c.Request.Context()).
		Where("id IN ?", req.EventIDs).Find(&events).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询告警失败")
		return
	}

	// 收集需要删除的屏蔽规则
	type ruleKey struct {
		Severity string
		RuleName string
		Labels   string
	}
	rulesToDelete := make(map[ruleKey]bool)
	for _, event := range events {
		if event.Silenced {
			key := ruleKey{
				Severity: event.Severity,
				RuleName: event.RuleName,
				Labels:   event.Labels,
			}
			rulesToDelete[key] = true
		}
	}

	// 删除屏蔽规则
	for key := range rulesToDelete {
		rule, err := s.silenceRuleRepo.FindMatchingRule(c.Request.Context(), key.Severity, key.RuleName, key.Labels)
		if err == nil && rule != nil {
			s.silenceRuleRepo.Delete(c.Request.Context(), rule.ID)
		}
	}

	// 更新事件状态
	for _, event := range events {
		event.Silenced = false
		event.SilenceUntil = nil
		event.SilenceReason = ""
		event.SilenceType = ""
		event.SilenceTimeRanges = ""
		event.SilencedAt = nil
		s.eventRepo.Update(c.Request.Context(), event)
	}

	response.Success(c, gin.H{
		"unsilencedEvents": len(events),
		"deletedRules":     len(rulesToDelete),
	})
}

// listSilencedEvents 查询已屏蔽告警列表
func (s *HTTPServer) listSilencedEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	severity := c.Query("severity")
	status := c.Query("status")
	keyword := c.Query("keyword")
	labelFilter := c.Query("labelFilter")

	list, total, err := s.eventRepo.ListSilenced(c.Request.Context(), page, pageSize, severity, status, keyword, labelFilter)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}
	response.Success(c, gin.H{"total": total, "page": page, "pageSize": pageSize, "data": list})
}

var _ *biz.AlertEvent