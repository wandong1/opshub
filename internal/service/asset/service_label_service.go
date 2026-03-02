package asset

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// ServiceLabelService 服务标签HTTP服务
type ServiceLabelService struct {
	useCase *asset.ServiceLabelUseCase
}

// NewServiceLabelService 创建服务标签服务
func NewServiceLabelService(useCase *asset.ServiceLabelUseCase) *ServiceLabelService {
	return &ServiceLabelService{useCase: useCase}
}

// List 列表查询
func (s *ServiceLabelService) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	list, total, err := s.useCase.List(c.Request.Context(), page, pageSize, keyword)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"list": list, "total": total})
}

// Create 创建
func (s *ServiceLabelService) Create(c *gin.Context) {
	var req asset.ServiceLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	vo, err := s.useCase.Create(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}
	response.Success(c, vo)
}

// Update 更新
func (s *ServiceLabelService) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}
	var req asset.ServiceLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	vo, err := s.useCase.Update(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}
	response.Success(c, vo)
}

// Delete 删除
func (s *ServiceLabelService) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}
	if err := s.useCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}
	response.Success(c, nil)
}

// GetByID 获取详情
func (s *ServiceLabelService) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}
	vo, err := s.useCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}
	response.Success(c, vo)
}
