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

// GroupHandler 分组规则处理器
type GroupHandler struct {
	repo *alertdata.GroupRuleRepo
}

// NewGroupHandler 创建分组规则处理器
func NewGroupHandler(db *gorm.DB) *GroupHandler {
	return &GroupHandler{
		repo: alertdata.NewGroupRuleRepo(db),
	}
}

// List 查询分组规则列表
func (h *GroupHandler) List(c *gin.Context) {
	subscriptionID, _ := strconv.ParseUint(c.Query("subscriptionId"), 10, 64)

	var rules []*biz.AlertGroupRule
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

// Get 查询分组规则详情
func (h *GroupHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	rule, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "规则不存在")
		return
	}

	response.Success(c, rule)
}

// Create 创建分组规则
func (h *GroupHandler) Create(c *gin.Context) {
	var rule biz.AlertGroupRule
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

// Update 更新分组规则
func (h *GroupHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var rule biz.AlertGroupRule
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

// Delete 删除分组规则
func (h *GroupHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.repo.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}

	response.Success(c, nil)
}
