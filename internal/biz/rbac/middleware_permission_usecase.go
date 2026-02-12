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

import "context"

// MiddlewarePermissionUseCase 中间件权限用例
type MiddlewarePermissionUseCase struct {
	repo MiddlewarePermissionRepo
}

// NewMiddlewarePermissionUseCase 创建中间件权限用例
func NewMiddlewarePermissionUseCase(repo MiddlewarePermissionRepo) *MiddlewarePermissionUseCase {
	return &MiddlewarePermissionUseCase{repo: repo}
}

// Create 创建中间件权限
func (uc *MiddlewarePermissionUseCase) Create(ctx context.Context, roleID, assetGroupID uint, middlewareIDs []uint, middlewareType string, permissions uint) error {
	return uc.repo.Create(ctx, roleID, assetGroupID, middlewareIDs, middlewareType, permissions)
}

// Delete 删除中间件权限
func (uc *MiddlewarePermissionUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

// Update 更新中间件权限
func (uc *MiddlewarePermissionUseCase) Update(ctx context.Context, id uint, roleID, assetGroupID uint, middlewareIDs []uint, middlewareType string, permissions uint) error {
	return uc.repo.Update(ctx, id, roleID, assetGroupID, middlewareIDs, middlewareType, permissions)
}

// List 分页查询权限列表
func (uc *MiddlewarePermissionUseCase) List(ctx context.Context, page, pageSize int, roleID, assetGroupID *uint) ([]*MiddlewarePermissionInfo, int64, error) {
	return uc.repo.List(ctx, page, pageSize, roleID, assetGroupID)
}

// CheckMiddlewarePermission 检查用户中间件权限
func (uc *MiddlewarePermissionUseCase) CheckMiddlewarePermission(ctx context.Context, userID, middlewareID uint, operation uint) (bool, error) {
	return uc.repo.CheckMiddlewarePermission(ctx, userID, middlewareID, operation)
}

// GetUserAccessibleMiddlewareIDs 获取用户可访问的中间件ID列表
func (uc *MiddlewarePermissionUseCase) GetUserAccessibleMiddlewareIDs(ctx context.Context, userID uint) ([]uint, error) {
	return uc.repo.GetUserAccessibleMiddlewareIDs(ctx, userID)
}
