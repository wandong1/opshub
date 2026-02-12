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

package asset

import (
	"context"
	"fmt"
	"strings"
)

// MiddlewareUseCase 中间件用例
type MiddlewareUseCase struct {
	repo      MiddlewareRepo
	groupRepo AssetGroupRepo
	hostRepo  HostRepo
}

// NewMiddlewareUseCase 创建中间件用例
func NewMiddlewareUseCase(repo MiddlewareRepo, groupRepo AssetGroupRepo, hostRepo HostRepo) *MiddlewareUseCase {
	return &MiddlewareUseCase{repo: repo, groupRepo: groupRepo, hostRepo: hostRepo}
}

// Create 创建中间件
func (uc *MiddlewareUseCase) Create(ctx context.Context, req *MiddlewareRequest) (*Middleware, error) {
	// 验证端口范围
	if req.Port < 1 || req.Port > 65535 {
		return nil, fmt.Errorf("端口范围必须在 1-65535 之间")
	}

	// 验证关联主机是否存在
	if req.HostID > 0 {
		if _, err := uc.hostRepo.GetByID(ctx, req.HostID); err != nil {
			return nil, fmt.Errorf("关联主机不存在")
		}
	}

	mw := req.ToModel()
	if err := uc.repo.Create(ctx, mw); err != nil {
		return nil, err
	}
	return mw, nil
}

// Update 更新中间件
func (uc *MiddlewareUseCase) Update(ctx context.Context, req *MiddlewareRequest) error {
	mw, err := uc.repo.GetByID(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("中间件不存在")
	}

	if req.HostID > 0 {
		if _, err := uc.hostRepo.GetByID(ctx, req.HostID); err != nil {
			return fmt.Errorf("关联主机不存在")
		}
	}

	mw.Name = req.Name
	mw.Type = req.Type
	mw.GroupID = req.GroupID
	mw.HostID = req.HostID
	mw.Host = req.Host
	mw.Port = req.Port
	mw.Username = req.Username
	if req.Password != "" {
		mw.Password = req.Password
	}
	mw.DatabaseName = req.DatabaseName
	mw.ConnectionParams = req.ConnectionParams
	mw.Tags = req.Tags
	mw.Description = req.Description

	return uc.repo.Update(ctx, mw)
}

// Delete 删除中间件
func (uc *MiddlewareUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

// GetByID 根据ID获取中间件详情
func (uc *MiddlewareUseCase) GetByID(ctx context.Context, id uint) (*MiddlewareInfoVO, error) {
	mw, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return uc.toInfoVO(ctx, mw), nil
}

// GetByIDDecrypted 获取解密后的中间件（用于连接）
func (uc *MiddlewareUseCase) GetByIDDecrypted(ctx context.Context, id uint) (*Middleware, error) {
	return uc.repo.GetByIDDecrypted(ctx, id)
}

// List 分页查询中间件列表
func (uc *MiddlewareUseCase) List(ctx context.Context, page, pageSize int, keyword, mwType string, groupID *uint, status *int, accessibleIDs []uint) ([]*MiddlewareInfoVO, int64, error) {
	var groupIDs []uint
	if groupID != nil && *groupID > 0 {
		groupIDs = append(groupIDs, *groupID)
		descendantIDs, err := uc.groupRepo.GetDescendantIDs(ctx, *groupID)
		if err == nil {
			groupIDs = append(groupIDs, descendantIDs...)
		}
	}

	middlewares, total, err := uc.repo.List(ctx, page, pageSize, keyword, mwType, groupIDs, status, accessibleIDs)
	if err != nil {
		return nil, 0, err
	}

	var vos []*MiddlewareInfoVO
	for _, mw := range middlewares {
		vos = append(vos, uc.toInfoVO(ctx, mw))
	}
	return vos, total, nil
}

// BatchDelete 批量删除
func (uc *MiddlewareUseCase) BatchDelete(ctx context.Context, ids []uint) error {
	return uc.repo.BatchDelete(ctx, ids)
}

// UpdateStatus 更新状态
func (uc *MiddlewareUseCase) UpdateStatus(ctx context.Context, id uint, status int, version string) error {
	return uc.repo.UpdateStatus(ctx, id, status, version)
}

// toInfoVO 转换为InfoVO
func (uc *MiddlewareUseCase) toInfoVO(ctx context.Context, mw *Middleware) *MiddlewareInfoVO {
	statusText := "未知"
	if mw.Status == 1 {
		statusText = "在线"
	} else if mw.Status == 0 {
		statusText = "离线"
	}

	var tags []string
	if mw.Tags != "" {
		tags = strings.Split(mw.Tags, ",")
	}

	var lastChecked string
	if mw.LastChecked != nil {
		lastChecked = mw.LastChecked.Format("2006-01-02 15:04:05")
	}

	vo := &MiddlewareInfoVO{
		ID:               mw.ID,
		Name:             mw.Name,
		Type:             mw.Type,
		TypeText:         GetMiddlewareTypeText(mw.Type),
		GroupID:          mw.GroupID,
		HostID:           mw.HostID,
		Host:             mw.Host,
		Port:             mw.Port,
		Username:         mw.Username,
		DatabaseName:     mw.DatabaseName,
		ConnectionParams: mw.ConnectionParams,
		Tags:             tags,
		Description:      mw.Description,
		Status:           mw.Status,
		StatusText:       statusText,
		Version:          mw.Version,
		LastChecked:      lastChecked,
		CreateTime:       mw.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdateTime:       mw.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	// 加载分组名称
	if mw.GroupID > 0 {
		group, err := uc.groupRepo.GetByID(ctx, mw.GroupID)
		if err == nil && group != nil {
			vo.GroupName = group.Name
		}
	}

	// 加载主机名称
	if mw.HostID > 0 {
		host, err := uc.hostRepo.GetByID(ctx, mw.HostID)
		if err == nil && host != nil {
			vo.HostName = host.Name
		}
	}

	return vo
}
