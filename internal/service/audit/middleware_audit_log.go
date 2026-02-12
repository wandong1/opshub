package audit

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/audit"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

type MiddlewareAuditLogService struct {
	useCase *audit.MiddlewareAuditLogUseCase
}

func NewMiddlewareAuditLogService(useCase *audit.MiddlewareAuditLogUseCase) *MiddlewareAuditLogService {
	return &MiddlewareAuditLogService{useCase: useCase}
}

// ListMiddlewareAuditLogs 中间件审计日志列表
func (s *MiddlewareAuditLogService) ListMiddlewareAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	username := c.Query("username")
	middlewareType := c.Query("middlewareType")
	commandType := c.Query("commandType")
	status := c.Query("status")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	var middlewareID *uint
	if midStr := c.Query("middlewareId"); midStr != "" {
		if mid, err := strconv.ParseUint(midStr, 10, 32); err == nil {
			v := uint(mid)
			middlewareID = &v
		}
	}

	logs, total, err := s.useCase.List(c.Request.Context(), page, pageSize, username, middlewareType, commandType, status, startTime, endTime, middlewareID)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}
	list := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		list = append(list, gin.H{
			"id":             log.ID,
			"userId":         log.UserID,
			"username":       log.Username,
			"middlewareId":   log.MiddlewareID,
			"middlewareName": log.MiddlewareName,
			"middlewareType": log.MiddlewareType,
			"database":       log.Database,
			"command":        log.Command,
			"commandType":    log.CommandType,
			"status":         log.Status,
			"errorMsg":       log.ErrorMsg,
			"duration":       log.Duration,
			"affectedRows":   log.AffectedRows,
			"ip":             log.IP,
			"createdAt":      log.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	response.Success(c, gin.H{
		"list":     list,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	})
}

// DeleteMiddlewareAuditLog 删除中间件审计日志
func (s *MiddlewareAuditLogService) DeleteMiddlewareAuditLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的日志ID")
		return
	}

	if err := s.useCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// DeleteMiddlewareAuditLogsBatch 批量删除中间件审计日志
func (s *MiddlewareAuditLogService) DeleteMiddlewareAuditLogsBatch(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.useCase.DeleteBatch(c.Request.Context(), req.IDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}
