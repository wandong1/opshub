package rbac

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

type RoleService struct {
	roleUseCase *rbac.RoleUseCase
}

func NewRoleService(roleUseCase *rbac.RoleUseCase) *RoleService {
	return &RoleService{
		roleUseCase: roleUseCase,
	}
}

// CreateRole 创建角色
func (s *RoleService) CreateRole(c *gin.Context) {
	var req rbac.SysRole
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.roleUseCase.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, req)
}

// UpdateRole 更新角色
func (s *RoleService) UpdateRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的角色ID")
		return
	}

	var req rbac.SysRole
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	if err := s.roleUseCase.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, req)
}

// DeleteRole 删除角色
func (s *RoleService) DeleteRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的角色ID")
		return
	}

	if err := s.roleUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetRole 获取角色详情
func (s *RoleService) GetRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的角色ID")
		return
	}

	role, err := s.roleUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "角色不存在")
		return
	}

	response.Success(c, role)
}

// ListRoles 角色列表
func (s *RoleService) ListRoles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	roles, total, err := s.roleUseCase.List(c.Request.Context(), page, pageSize, keyword)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     roles,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetAllRoles 获取所有角色（不分页）
func (s *RoleService) GetAllRoles(c *gin.Context) {
	roles, err := s.roleUseCase.GetAll(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, roles)
}

// AssignRoleMenus 分配角色菜单
type AssignRoleMenusRequest struct {
	MenuIDs []uint `json:"menuIds" binding:"required"`
}

func (s *RoleService) AssignRoleMenus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的角色ID")
		return
	}

	var req AssignRoleMenusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.roleUseCase.AssignMenus(c.Request.Context(), uint(id), req.MenuIDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "分配失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}
