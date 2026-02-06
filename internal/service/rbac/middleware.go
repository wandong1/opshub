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
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	rbacdata "github.com/ydcloud-dy/opshub/internal/data/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

const (
	UserIdKey   = "user_id"
	UsernameKey = "username"
)

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) uint {
	if userID, exists := c.Get(UserIdKey); exists {
		if id, ok := userID.(uint); ok {
			return id
		}
	}
	return 0
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) string {
	if username, exists := c.Get(UsernameKey); exists {
		if name, ok := username.(string); ok {
			return name
		}
	}
	return ""
}

// AuthMiddleware JWT认证中间件
type AuthMiddleware struct {
	authService         *AuthService
	assetPermissionRepo rbac.AssetPermissionRepo
	menuRepo            rbac.MenuRepo
	permissionCache     *rbacdata.PermissionCache
}

func NewAuthMiddleware(authService *AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// SetPermissionDeps 设置权限检查依赖
func (m *AuthMiddleware) SetPermissionDeps(menuRepo rbac.MenuRepo, cache *rbacdata.PermissionCache) {
	m.menuRepo = menuRepo
	m.permissionCache = cache
}

// SetAssetPermissionRepo 设置资产权限仓储
func (m *AuthMiddleware) SetAssetPermissionRepo(repo rbac.AssetPermissionRepo) {
	m.assetPermissionRepo = repo
}

// AuthRequired JWT认证
func (m *AuthMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// 优先从 Authorization header 获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}

		// 如果 header 中没有，尝试从 query 参数获取（用于 WebSocket 连接）
		if token == "" {
			token = c.Query("token")
		}

		// 如果都没有，返回未授权
		if token == "" {
			response.ErrorCode(c, http.StatusUnauthorized, "未登录")
			c.Abort()
			return
		}

		claims, err := m.authService.ParseToken(token)
		if err != nil {
			response.ErrorCode(c, http.StatusUnauthorized, "token无效或已过期")
			c.Abort()
			return
		}

		c.Set(UserIdKey, claims.UserID)
		c.Set(UsernameKey, claims.Username)
		c.Next()
	}
}

// RequireAdmin 检查是否为管理员
func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			response.ErrorCode(c, http.StatusUnauthorized, "未登录")
			c.Abort()
			return
		}

		// 获取用户角色
		roles, err := m.authService.roleUseCase.GetByUserID(c.Request.Context(), userID)
		if err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "获取用户角色失败")
			c.Abort()
			return
		}

		// 检查是否有admin角色
		hasAdminRole := false
		for _, role := range roles {
			if role.Code == "admin" {
				hasAdminRole = true
				break
			}
		}

		if !hasAdminRole {
			response.ErrorCode(c, http.StatusForbidden, "权限不足：此操作仅限管理员执行")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireHostPermission 检查主机操作权限的中间件
func (m *AuthMiddleware) RequireHostPermission(operation uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.assetPermissionRepo == nil {
			response.ErrorCode(c, http.StatusInternalServerError, "权限检查未初始化")
			c.Abort()
			return
		}

		// 获取用户ID
		userID := GetUserID(c)
		if userID == 0 {
			response.ErrorCode(c, http.StatusUnauthorized, "未登录")
			c.Abort()
			return
		}

		// 获取主机ID
		hostIDStr := c.Param("id")
		if hostIDStr == "" {
			// 对于创建操作，暂不检查具体权限（创建权限通过分组权限检查）
			c.Next()
			return
		}

		hostID, err := strconv.ParseUint(hostIDStr, 10, 32)
		if err != nil {
			response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
			c.Abort()
			return
		}

		// 检查权限
		hasPermission, err := m.assetPermissionRepo.CheckHostOperationPermission(
			c.Request.Context(),
			userID,
			uint(hostID),
			operation,
		)

		if err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "权限检查失败")
			c.Abort()
			return
		}

		if !hasPermission {
			response.ErrorCode(c, http.StatusForbidden, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}

// 白名单路由，无需权限检查
var permissionWhitelist = map[string]bool{
	"/api/v1/profile":          true,
	"/api/v1/profile/password": true,
	"/api/v1/menus/user":       true,
	"/api/v1/profile/avatar":   true,
	"/api/v1/upload/avatar":    true,
}

// RequirePermission API权限检查中间件
func (m *AuthMiddleware) RequirePermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 权限依赖未初始化时直接放行
		if m.menuRepo == nil || m.permissionCache == nil {
			c.Next()
			return
		}

		// 白名单路由直接放行
		fullPath := c.FullPath()
		if fullPath == "" {
			c.Next()
			return
		}
		if permissionWhitelist[fullPath] {
			c.Next()
			return
		}

		userID := GetUserID(c)
		if userID == 0 {
			c.Next()
			return
		}

		ctx := c.Request.Context()

		// 检查是否admin（优先缓存）
		isAdmin, found := m.permissionCache.GetUserIsAdmin(ctx, userID)
		if !found {
			isAdmin = m.checkIsAdmin(ctx, userID)
			m.permissionCache.SetUserIsAdmin(ctx, userID, isAdmin)
		}
		if isAdmin {
			c.Next()
			return
		}

		method := c.Request.Method

		// 获取已注册API列表（优先缓存）
		registeredAPIs, err := m.permissionCache.GetRegisteredAPIs(ctx)
		if err != nil || registeredAPIs == nil {
			registeredAPIs, err = m.menuRepo.GetAllRegisteredAPIs(ctx)
			if err != nil {
				c.Next()
				return
			}
			m.permissionCache.SetRegisteredAPIs(ctx, registeredAPIs)
		}

		// 检查该API是否在已注册列表中，未注册的API放行（渐进式迁移）
		apiRegistered := false
		for _, api := range registeredAPIs {
			if api.ApiPath == fullPath && api.ApiMethod == method {
				apiRegistered = true
				break
			}
		}
		if !apiRegistered {
			c.Next()
			return
		}

		// 获取用户权限列表（优先缓存）
		userPerms, err := m.permissionCache.GetUserPermissions(ctx, userID)
		if err != nil || userPerms == nil {
			userPerms, err = m.menuRepo.GetAPIPermissionsByUserID(ctx, userID)
			if err != nil {
				response.ErrorCode(c, http.StatusInternalServerError, "权限检查失败")
				c.Abort()
				return
			}
			m.permissionCache.SetUserPermissions(ctx, userID, userPerms)
		}

		// 匹配权限
		for _, perm := range userPerms {
			if perm.ApiPath == fullPath && perm.ApiMethod == method {
				c.Next()
				return
			}
		}

		response.ErrorCode(c, http.StatusForbidden, "权限不足：无此接口访问权限")
		c.Abort()
	}
}

// checkIsAdmin 检查用户是否拥有admin角色
func (m *AuthMiddleware) checkIsAdmin(ctx context.Context, userID uint) bool {
	roles, err := m.authService.roleUseCase.GetByUserID(ctx, userID)
	if err != nil {
		return false
	}
	for _, role := range roles {
		if role.Code == "admin" {
			return true
		}
	}
	return false
}
