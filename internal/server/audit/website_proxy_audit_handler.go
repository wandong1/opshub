// Copyright (c) 2026 DYCloud J.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED.

package audit

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// ListWebsiteProxyAuditLogs 查询网站代理访问审计日志列表
func (s *HTTPService) ListWebsiteProxyAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 构建过滤条件
	filters := make(map[string]interface{})
	if username := c.Query("username"); username != "" {
		filters["username"] = username
	}
	if websiteName := c.Query("websiteName"); websiteName != "" {
		filters["websiteName"] = websiteName
	}
	if websiteIDStr := c.Query("websiteId"); websiteIDStr != "" {
		if websiteID, err := strconv.ParseUint(websiteIDStr, 10, 32); err == nil {
			filters["websiteId"] = uint(websiteID)
		}
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if method := c.Query("method"); method != "" {
		filters["method"] = method
	}
	if startTime := c.Query("startTime"); startTime != "" {
		filters["startTime"] = startTime
	}
	if endTime := c.Query("endTime"); endTime != "" {
		filters["endTime"] = endTime
	}

	logs, total, err := s.websiteProxyAuditQueryService.List(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询审计日志失败")
		return
	}

	response.Pagination(c, total, page, pageSize, logs)
}

// DeleteWebsiteProxyAuditLog 删除网站代理访问审计日志
func (s *HTTPService) DeleteWebsiteProxyAuditLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的日志ID")
		return
	}

	if err := s.websiteProxyAuditQueryService.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除审计日志失败")
		return
	}

	response.Success(c, nil)
}

// DeleteWebsiteProxyAuditLogsBatch 批量删除网站代理访问审计日志
func (s *HTTPService) DeleteWebsiteProxyAuditLogsBatch(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误")
		return
	}

	if err := s.websiteProxyAuditQueryService.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "批量删除审计日志失败")
		return
	}

	response.Success(c, nil)
}
