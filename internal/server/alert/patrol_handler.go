package alert

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"go.uber.org/zap"
)

// GetPatrolConfig 获取巡检配置
func (s *HTTPServer) GetPatrolConfig(c *gin.Context) {
	ruleID, err := strconv.ParseUint(c.Param("ruleId"), 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "规则ID无效")
		return
	}

	appLogger.Info("获取巡检配置", zap.Uint("subscriptionRuleID", uint(ruleID)))

	patrol, err := s.patrolService.GetPatrolRepo().GetBySubscriptionRuleID(c.Request.Context(), uint(ruleID))
	if err != nil {
		// 如果不存在，返回默认配置
		appLogger.Info("巡检配置不存在，返回默认配置", zap.Uint("subscriptionRuleID", uint(ruleID)))
		patrol = &biz.AlertSubscriptionRulePatrol{
			SubscriptionRuleID: uint(ruleID),
			Enabled:            false,
			PatrolMode:         "interval",
			PatrolInterval:     3600,
			IncludeResolved:    false,
			TimeRange:          0,
			MaxAlertsPerReport: 100,
			SendMode:           "always",
			ReportStyle:        "detailed",
			GroupBy:            "severity",
		}
	} else {
		appLogger.Info("找到巡检配置",
			zap.Uint("id", patrol.ID),
			zap.Uint("subscriptionRuleID", patrol.SubscriptionRuleID),
			zap.Bool("enabled", patrol.Enabled))
	}

	response.Success(c, patrol)
}

// SavePatrolConfig 保存巡检配置
func (s *HTTPServer) SavePatrolConfig(c *gin.Context) {
	var req biz.AlertSubscriptionRulePatrol
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	appLogger.Info("保存巡检配置",
		zap.Uint("subscriptionRuleID", req.SubscriptionRuleID),
		zap.Bool("enabled", req.Enabled),
		zap.Int("patrolInterval", req.PatrolInterval))

	// 检查订阅规则是否存在
	_, err := s.subRuleRepo.GetByID(c.Request.Context(), req.SubscriptionRuleID)
	if err != nil {
		appLogger.Warn("订阅规则不存在", zap.Uint("subscriptionRuleID", req.SubscriptionRuleID))
		response.ErrorCode(c, http.StatusNotFound, "订阅规则不存在")
		return
	}

	// 查询是否已存在配置
	existing, err := s.patrolService.GetPatrolRepo().GetBySubscriptionRuleID(c.Request.Context(), req.SubscriptionRuleID)
	if err != nil {
		// 不存在，创建新配置
		appLogger.Info("创建新的巡检配置", zap.Uint("subscriptionRuleID", req.SubscriptionRuleID))
		if err := s.patrolService.GetPatrolRepo().Create(c.Request.Context(), &req); err != nil {
			appLogger.Error("创建巡检配置失败", zap.Error(err))
			response.ErrorCode(c, http.StatusInternalServerError, "创建巡检配置失败")
			return
		}
		appLogger.Info("创建巡检配置成功", zap.Uint("id", req.ID))
	} else {
		// 已存在，更新配置
		appLogger.Info("更新已有的巡检配置",
			zap.Uint("existingID", existing.ID),
			zap.Uint("subscriptionRuleID", req.SubscriptionRuleID))
		req.ID = existing.ID
		req.CreatedAt = existing.CreatedAt
		if err := s.patrolService.GetPatrolRepo().Update(c.Request.Context(), &req); err != nil {
			appLogger.Error("更新巡检配置失败", zap.Error(err))
			response.ErrorCode(c, http.StatusInternalServerError, "更新巡检配置失败")
			return
		}
		appLogger.Info("更新巡检配置成功", zap.Uint("id", req.ID))
	}

	response.Success(c, req)
}

// GetPatrolReports 获取巡检报告列表
func (s *HTTPServer) GetPatrolReports(c *gin.Context) {
	ruleID, err := strconv.ParseUint(c.Param("ruleId"), 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "规则ID无效")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	reports, total, err := s.patrolService.GetReportRepo().ListBySubscriptionRule(c.Request.Context(), uint(ruleID), page, pageSize)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询巡检报告失败")
		return
	}

	response.Pagination(c, total, page, pageSize, reports)
}

// ExecutePatrol 手动执行巡检
func (s *HTTPServer) ExecutePatrol(c *gin.Context) {
	ruleID, err := strconv.ParseUint(c.Param("ruleId"), 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "规则ID无效")
		return
	}

	if err := s.patrolService.ExecuteManually(c.Request.Context(), uint(ruleID)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "巡检任务已启动"})
}
