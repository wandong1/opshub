// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package rbac

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// MiddlewarePermissionService 中间件权限HTTP服务
type MiddlewarePermissionService struct {
	useCase *rbac.MiddlewarePermissionUseCase
}

// NewMiddlewarePermissionService 创建中间件权限服务
func NewMiddlewarePermissionService(useCase *rbac.MiddlewarePermissionUseCase) *MiddlewarePermissionService {
	return &MiddlewarePermissionService{useCase: useCase}
}

// CreateMiddlewarePermission 创建中间件权限
func (s *MiddlewarePermissionService) CreateMiddlewarePermission(c *gin.Context) {
	var req rbac.MiddlewarePermissionCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if req.Permissions == 0 {
		req.Permissions = rbac.MWPermView
	}

	// 支持多分组：优先使用 assetGroupIds，兼容单个 assetGroupId
	groupIDs := req.AssetGroupIDs
	if len(groupIDs) == 0 && req.AssetGroupID > 0 {
		groupIDs = []uint{req.AssetGroupID}
	}
	if len(groupIDs) == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请选择至少一个业务分组")
		return
	}

	for _, gid := range groupIDs {
		if err := s.useCase.Create(c.Request.Context(), req.RoleID, gid, req.MiddlewareIDs, req.MiddlewareType, req.Permissions); err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
			return
		}
	}

	response.SuccessWithMessage(c, "创建成功", nil)
}

// UpdateMiddlewarePermission 更新中间件权限
func (s *MiddlewarePermissionService) UpdateMiddlewarePermission(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的权限ID")
		return
	}

	var req rbac.MiddlewarePermissionUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if req.Permissions == 0 {
		req.Permissions = rbac.MWPermView
	}

	if err := s.useCase.Update(c.Request.Context(), uint(id), req.RoleID, req.AssetGroupID, req.MiddlewareIDs, req.MiddlewareType, req.Permissions); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteMiddlewarePermission 删除中间件权限
func (s *MiddlewarePermissionService) DeleteMiddlewarePermission(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的权限ID")
		return
	}

	if err := s.useCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// ListMiddlewarePermissions 中间件权限列表
func (s *MiddlewarePermissionService) ListMiddlewarePermissions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	var roleID *uint
	if roleIDStr := c.Query("roleId"); roleIDStr != "" {
		id, err := strconv.ParseUint(roleIDStr, 10, 32)
		if err == nil {
			val := uint(id)
			roleID = &val
		}
	}

	var assetGroupID *uint
	if gidStr := c.Query("assetGroupId"); gidStr != "" {
		id, err := strconv.ParseUint(gidStr, 10, 32)
		if err == nil {
			val := uint(id)
			assetGroupID = &val
		}
	}

	list, total, err := s.useCase.List(c.Request.Context(), page, pageSize, roleID, assetGroupID)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"total": total,
		"list":  list,
	})
}
