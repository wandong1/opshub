package rbac

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

type DepartmentService struct {
	deptUseCase *rbac.DepartmentUseCase
}

func NewDepartmentService(deptUseCase *rbac.DepartmentUseCase) *DepartmentService {
	return &DepartmentService{
		deptUseCase: deptUseCase,
	}
}

// CreateDepartment 创建部门
func (s *DepartmentService) CreateDepartment(c *gin.Context) {
	var req rbac.SysDepartment
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.deptUseCase.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, req)
}

// UpdateDepartment 更新部门
func (s *DepartmentService) UpdateDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的部门ID")
		return
	}

	var req rbac.SysDepartment
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	if err := s.deptUseCase.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, req)
}

// DeleteDepartment 删除部门
func (s *DepartmentService) DeleteDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的部门ID")
		return
	}

	if err := s.deptUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetDepartment 获取部门详情
func (s *DepartmentService) GetDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的部门ID")
		return
	}

	dept, err := s.deptUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "部门不存在")
		return
	}

	response.Success(c, dept)
}

// GetDepartmentTree 获取部门树
func (s *DepartmentService) GetDepartmentTree(c *gin.Context) {
	tree, err := s.deptUseCase.GetTree(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, tree)
}
