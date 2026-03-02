package inspection

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	biz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// ProbeVariableService handles HTTP requests for probe variables.
type ProbeVariableService struct {
	useCase *biz.ProbeVariableUseCase
}

func NewProbeVariableService(uc *biz.ProbeVariableUseCase) *ProbeVariableService {
	return &ProbeVariableService{useCase: uc}
}

func (s *ProbeVariableService) Create(c *gin.Context) {
	var req biz.ProbeVariableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	model := &biz.ProbeVariable{
		Name:        req.Name,
		Value:       req.Value,
		VarType:     req.VarType,
		GroupIDs:    req.GroupIDs,
		Description: req.Description,
	}
	if err := s.useCase.Create(c.Request.Context(), model); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}
	response.Success(c, model)
}

func (s *ProbeVariableService) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req biz.ProbeVariableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	model := &biz.ProbeVariable{
		ID:          uint(id),
		Name:        req.Name,
		Value:       req.Value,
		VarType:     req.VarType,
		GroupIDs:    req.GroupIDs,
		Description: req.Description,
	}
	if err := s.useCase.Update(c.Request.Context(), model); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}
	response.Success(c, model)
}

func (s *ProbeVariableService) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.useCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}
	response.Success(c, nil)
}

func (s *ProbeVariableService) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	v, err := s.useCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "未找到: "+err.Error())
		return
	}
	response.Success(c, v)
}

func (s *ProbeVariableService) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")
	varType := c.Query("var_type")
	groupIDs := c.Query("group_ids")

	vars, total, err := s.useCase.List(c.Request.Context(), page, pageSize, keyword, varType, groupIDs)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}
	response.Pagination(c, total, page, pageSize, vars)
}
