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

	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"gorm.io/gorm"
)

type middlewarePermissionRepo struct {
	db *gorm.DB
}

// NewMiddlewarePermissionRepo 创建中间件权限仓储
func NewMiddlewarePermissionRepo(db *gorm.DB) rbac.MiddlewarePermissionRepo {
	return &middlewarePermissionRepo{db: db}
}

// Create 创建中间件权限
func (r *middlewarePermissionRepo) Create(ctx context.Context, roleID, assetGroupID uint, middlewareIDs []uint, middlewareType string, permissions uint) error {
	// 先硬删除该角色对该分组的现有权限
	if err := r.db.WithContext(ctx).
		Where("role_id = ? AND asset_group_id = ?", roleID, assetGroupID).
		Unscoped().Delete(&rbac.SysRoleMiddlewarePermission{}).Error; err != nil {
		return err
	}

	if permissions == 0 {
		permissions = rbac.MWPermView
	}

	permission := &rbac.SysRoleMiddlewarePermission{
		RoleID:         roleID,
		AssetGroupID:   assetGroupID,
		MiddlewareIDs:  rbac.UintArray(middlewareIDs),
		MiddlewareType: middlewareType,
		Permissions:    permissions,
	}
	return r.db.WithContext(ctx).Create(permission).Error
}

// Delete 删除中间件权限
func (r *middlewarePermissionRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&rbac.SysRoleMiddlewarePermission{}, id).Error
}

// Update 更新中间件权限
func (r *middlewarePermissionRepo) Update(ctx context.Context, id uint, roleID, assetGroupID uint, middlewareIDs []uint, middlewareType string, permissions uint) error {
	if err := r.db.WithContext(ctx).Model(&rbac.SysRoleMiddlewarePermission{}).
		Where("id = ?", id).Unscoped().Delete(&rbac.SysRoleMiddlewarePermission{}).Error; err != nil {
		return err
	}

	if permissions == 0 {
		permissions = rbac.MWPermView
	}

	permission := &rbac.SysRoleMiddlewarePermission{
		RoleID:         roleID,
		AssetGroupID:   assetGroupID,
		MiddlewareIDs:  rbac.UintArray(middlewareIDs),
		MiddlewareType: middlewareType,
		Permissions:    permissions,
	}
	return r.db.WithContext(ctx).Create(permission).Error
}

// GetByID 根据ID获取权限
func (r *middlewarePermissionRepo) GetByID(ctx context.Context, id uint) (*rbac.SysRoleMiddlewarePermission, error) {
	var permission rbac.SysRoleMiddlewarePermission
	err := r.db.WithContext(ctx).First(&permission, id).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// List 分页查询权限列表
func (r *middlewarePermissionRepo) List(ctx context.Context, page, pageSize int, roleID, assetGroupID *uint) ([]*rbac.MiddlewarePermissionInfo, int64, error) {
	var permissions []*rbac.MiddlewarePermissionInfo
	var total int64

	query := r.db.WithContext(ctx).
		Table("sys_role_middleware_permission AS p").
		Select(`
			p.id,
			p.role_id,
			r.name AS role_name,
			r.code AS role_code,
			p.asset_group_id,
			g.name AS asset_group_name,
			p.middleware_ids,
			p.middleware_type,
			p.permissions,
			p.created_at
		`).
		Joins("LEFT JOIN sys_role AS r ON p.role_id = r.id").
		Joins("LEFT JOIN asset_group AS g ON p.asset_group_id = g.id").
		Where("p.deleted_at IS NULL")

	if roleID != nil {
		query = query.Where("p.role_id = ?", *roleID)
	}
	if assetGroupID != nil {
		query = query.Where("p.asset_group_id = ?", *assetGroupID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("p.created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&permissions).Error

	return permissions, total, err
}

// CheckMiddlewarePermission 检查用户是否有对指定中间件的特定操作权限
func (r *middlewarePermissionRepo) CheckMiddlewarePermission(ctx context.Context, userID, middlewareID uint, operation uint) (bool, error) {
	// 检查是否是管理员
	var adminCount int64
	err := r.db.WithContext(ctx).
		Table("sys_user_role AS ur").
		Joins("JOIN sys_role AS r ON ur.role_id = r.id").
		Where("ur.user_id = ? AND r.code = ?", userID, "admin").
		Count(&adminCount).Error
	if err != nil {
		return false, err
	}
	if adminCount > 0 {
		return true, nil
	}

	// 获取中间件所属的分组ID
	var groupID uint
	err = r.db.WithContext(ctx).
		Table("middlewares").
		Select("group_id").
		Where("id = ? AND deleted_at IS NULL", middlewareID).
		Scan(&groupID).Error
	if err != nil {
		return false, err
	}

	// 检查权限
	var permCount int64
	err = r.db.WithContext(ctx).
		Table("sys_role_middleware_permission AS p").
		Joins("JOIN sys_user_role AS ur ON p.role_id = ur.role_id").
		Where("ur.user_id = ? AND p.asset_group_id = ? AND p.deleted_at IS NULL", userID, groupID).
		Where("(JSON_LENGTH(COALESCE(p.middleware_ids, JSON_ARRAY())) = 0 OR JSON_CONTAINS(p.middleware_ids, CAST(? AS JSON))) AND (p.permissions & ?) > 0", middlewareID, operation).
		Count(&permCount).Error
	if err != nil {
		return false, err
	}

	return permCount > 0, nil
}

// GetUserAccessibleMiddlewareIDs 获取用户有权限访问的所有中间件ID列表
func (r *middlewarePermissionRepo) GetUserAccessibleMiddlewareIDs(ctx context.Context, userID uint) ([]uint, error) {
	// 检查是否是管理员
	var adminCount int64
	err := r.db.WithContext(ctx).
		Table("sys_user_role AS ur").
		Joins("JOIN sys_role AS r ON ur.role_id = r.id").
		Where("ur.user_id = ? AND r.code = ?", userID, "admin").
		Count(&adminCount).Error
	if err != nil {
		return nil, err
	}

	// 管理员可以访问所有中间件
	if adminCount > 0 {
		var allIDs []uint
		err = r.db.WithContext(ctx).
			Table("middlewares").
			Where("deleted_at IS NULL").
			Pluck("id", &allIDs).Error
		return allIDs, err
	}

	// 获取用户有权限的中间件ID列表
	var middlewareIDs []uint
	err = r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT m.id
		FROM middlewares AS m
		JOIN sys_role_middleware_permission AS p ON p.asset_group_id = m.group_id
		JOIN sys_user_role AS ur ON p.role_id = ur.role_id
		WHERE ur.user_id = ?
		AND m.deleted_at IS NULL
		AND p.deleted_at IS NULL
		AND (
			JSON_LENGTH(COALESCE(p.middleware_ids, JSON_ARRAY())) = 0
			OR JSON_CONTAINS(p.middleware_ids, CAST(m.id AS JSON))
		)
	`, userID).Scan(&middlewareIDs).Error

	return middlewareIDs, err
}
