package alert

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	alertdata "github.com/ydcloud-dy/opshub/internal/data/alert"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"gorm.io/gorm"
)

// InhibitHandler 抑制规则处理器
type InhibitHandler struct {
	repo *alertdata.InhibitRuleRepo
}

// NewInhibitHandler 创建抑制规则处理器
func NewInhibitHandler(db *gorm.DB) *InhibitHandler {
	return &InhibitHandler{
		repo: alertdata.NewInhibitRuleRepo(db),
	}
}

// List 查询抑制规则列表
func (h *InhibitHandler) List(c *gin.Context) {
	subscriptionID, _ := strconv.ParseUint(c.Query("subscriptionId"), 10, 64)

	var rules []*biz.AlertInhibitRule
	var err error

	if subscriptionID > 0 {
		rules, err = h.repo.ListBySubscription(c.Request.Context(), uint(subscriptionID))
	} else {
		rules, err = h.repo.List(c.Request.Context())
	}

	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}

	response.Success(c, rules)
}

// Get 查询抑制规则详情
func (h *InhibitHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	rule, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "规则不存在")
		return
	}

	response.Success(c, rule)
}

// Create 创建抑制规则
func (h *InhibitHandler) Create(c *gin.Context) {
	var rule biz.AlertInhibitRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.repo.Create(c.Request.Context(), &rule); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败")
		return
	}

	response.Success(c, rule)
}

// Update 更新抑制规则
func (h *InhibitHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var rule biz.AlertInhibitRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	rule.ID = uint(id)
	if err := h.repo.Update(c.Request.Context(), &rule); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败")
		return
	}

	response.Success(c, rule)
}

// Delete 删除抑制规则
func (h *InhibitHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.repo.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}

	response.Success(c, nil)
}
