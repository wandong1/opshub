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

var _ *biz.AlertEvent