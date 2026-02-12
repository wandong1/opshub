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
	"time"

	"gorm.io/gorm"
)

// 中间件权限位常量
const (
	MWPermView    = 1 << 0 // 1 (查看)
	MWPermEdit    = 1 << 1 // 2 (编辑)
	MWPermDelete  = 1 << 2 // 4 (删除)
	MWPermConnect = 1 << 3 // 8 (测试连接)
	MWPermExecute = 1 << 4 // 16 (执行数据操作 — 保留兼容)
	MWPermQuery   = 1 << 5 // 32 (查询数据)
	MWPermModify  = 1 << 6 // 64 (修改数据：增删改+DDL)
	MWPermAll     = 0x7F   // 127 (所有权限)
)

// SysRoleMiddlewarePermission 角色中间件权限模型
type SysRoleMiddlewarePermission struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	RoleID         uint           `gorm:"not null;index:idx_role_mw" json:"roleId"`
	AssetGroupID   uint           `gorm:"not null;index:idx_role_mw" json:"assetGroupId"`
	MiddlewareIDs  UintArray      `gorm:"type:json" json:"middlewareIds"`
	MiddlewareType string         `gorm:"type:varchar(50);default:'';comment:中间件类型筛选" json:"middlewareType"`
	Permissions    uint           `gorm:"type:int unsigned;default:1;comment:操作权限位掩码：1=查看,2=编辑,4=删除,8=连接,16=执行;index" json:"permissions"`
}

// TableName 指定表名
func (SysRoleMiddlewarePermission) TableName() string {
	return "sys_role_middleware_permission"
}

// HasPermission 检查是否具有指定权限
func (p *SysRoleMiddlewarePermission) HasPermission(perm uint) bool {
	return (p.Permissions & perm) > 0
}

// AddPermission 添加权限
func (p *SysRoleMiddlewarePermission) AddPermission(perm uint) {
	p.Permissions |= perm
}

// RemovePermission 移除权限
func (p *SysRoleMiddlewarePermission) RemovePermission(perm uint) {
	p.Permissions &^= perm
}

// MiddlewarePermissionInfo 中间件权限信息（用于前端展示）
type MiddlewarePermissionInfo struct {
	ID              uint      `json:"id"`
	RoleID          uint      `json:"roleId"`
	RoleName        string    `json:"roleName"`
	RoleCode        string    `json:"roleCode"`
	AssetGroupID    uint      `json:"assetGroupId"`
	AssetGroupName  string    `json:"assetGroupName"`
	MiddlewareIDs   []uint    `json:"middlewareIds"`
	MiddlewareNames []string  `json:"middlewareNames,omitempty"`
	MiddlewareType  string    `json:"middlewareType"`
	IsAllMiddleware bool      `json:"isAllMiddleware"`
	Permissions     uint      `json:"permissions"`
	CreatedAt       time.Time `json:"createdAt"`
}

// MiddlewarePermissionCreateReq 创建中间件权限请求
type MiddlewarePermissionCreateReq struct {
	RoleID         uint   `json:"roleId" binding:"required"`
	AssetGroupID   uint   `json:"assetGroupId"`
	AssetGroupIDs  []uint `json:"assetGroupIds"`
	MiddlewareIDs  []uint `json:"middlewareIds"`
	MiddlewareType string `json:"middlewareType"`
	Permissions    uint   `json:"permissions"`
}

// MiddlewarePermissionUpdateReq 更新中间件权限请求
type MiddlewarePermissionUpdateReq struct {
	RoleID         uint   `json:"roleId" binding:"required"`
	AssetGroupID   uint   `json:"assetGroupId"`
	AssetGroupIDs  []uint `json:"assetGroupIds"`
	MiddlewareIDs  []uint `json:"middlewareIds"`
	MiddlewareType string `json:"middlewareType"`
	Permissions    uint   `json:"permissions"`
}

// GetMWPermissionName 获取中间件权限名称
func GetMWPermissionName(perm uint) string {
	switch perm {
	case MWPermView:
		return "查看"
	case MWPermEdit:
		return "编辑"
	case MWPermDelete:
		return "删除"
	case MWPermConnect:
		return "测试连接"
	case MWPermExecute:
		return "执行操作"
	case MWPermQuery:
		return "查询数据"
	case MWPermModify:
		return "修改数据"
	default:
		return "未知"
	}
}

// MiddlewarePermissionRepo 中间件权限仓储接口
type MiddlewarePermissionRepo interface {
	Create(ctx context.Context, roleID, assetGroupID uint, middlewareIDs []uint, middlewareType string, permissions uint) error
	Delete(ctx context.Context, id uint) error
	Update(ctx context.Context, id uint, roleID, assetGroupID uint, middlewareIDs []uint, middlewareType string, permissions uint) error
	List(ctx context.Context, page, pageSize int, roleID, assetGroupID *uint) ([]*MiddlewarePermissionInfo, int64, error)
	GetByID(ctx context.Context, id uint) (*SysRoleMiddlewarePermission, error)
	CheckMiddlewarePermission(ctx context.Context, userID, middlewareID uint, operation uint) (bool, error)
	GetUserAccessibleMiddlewareIDs(ctx context.Context, userID uint) ([]uint, error)
}
