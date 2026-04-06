package alert

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// listSilenceRules 查询屏蔽规则列表
func (s *HTTPServer) listSilenceRules(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	list, total, err := s.silenceRuleRepo.List(c.Request.Context(), page, pageSize)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}
	response.Success(c, gin.H{"total": total, "page": page, "pageSize": pageSize, "data": list})
}

// createSilenceRule 创建屏蔽规则
func (s *HTTPServer) createSilenceRule(c *gin.Context) {
	var req biz.AlertSilenceRule
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	// 计算屏蔽截止时间
	if req.Type == "fixed" && req.Duration != "" {
		dur, err := time.ParseDuration(req.Duration)
		if err != nil {
			response.ErrorCode(c, http.StatusBadRequest, "屏蔽时长格式错误")
			return
		}
		until := time.Now().Add(dur)
		req.SilenceUntil = &until
	}

	if err := s.silenceRuleRepo.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败")
		return
	}
	response.Success(c, req)
}

// getSilenceRule 获取屏蔽规则详情
func (s *HTTPServer) getSilenceRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	rule, err := s.silenceRuleRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "屏蔽规则不存在")
		return
	}
	response.Success(c, rule)
}

// updateSilenceRule 更新屏蔽规则
func (s *HTTPServer) updateSilenceRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	rule, err := s.silenceRuleRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "屏蔽规则不存在")
		return
	}

	if err := c.ShouldBindJSON(rule); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	// 重新计算屏蔽截止时间
	if rule.Type == "fixed" && rule.Duration != "" {
		dur, err := time.ParseDuration(rule.Duration)
		if err != nil {
			response.ErrorCode(c, http.StatusBadRequest, "屏蔽时长格式错误")
			return
		}
		until := time.Now().Add(dur)
		rule.SilenceUntil = &until
	}

	if err := s.silenceRuleRepo.Update(c.Request.Context(), rule); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败")
		return
	}
	response.Success(c, rule)
}

// deleteSilenceRule 删除屏蔽规则
func (s *HTTPServer) deleteSilenceRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.silenceRuleRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}
	response.Success(c, nil)
}

// toggleSilenceRule 启用/禁用屏蔽规则
func (s *HTTPServer) toggleSilenceRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	rule, err := s.silenceRuleRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "屏蔽规则不存在")
		return
	}

	rule.Enabled = !rule.Enabled
	if err := s.silenceRuleRepo.Update(c.Request.Context(), rule); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "操作失败")
		return
	}
	response.Success(c, rule)
}
