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

package system

import (
	"context"

	"github.com/ydcloud-dy/opshub/internal/biz/system"
	"gorm.io/gorm"
)

// APIKeyRepo API Key 仓库
type APIKeyRepo struct {
	db *gorm.DB
}

// NewAPIKeyRepo 创建 API Key 仓库
func NewAPIKeyRepo(db *gorm.DB) *APIKeyRepo {
	return &APIKeyRepo{db: db}
}

// Create 创建 API Key
func (r *APIKeyRepo) Create(ctx context.Context, apiKey *system.SysAPIKey) error {
	return r.db.WithContext(ctx).Create(apiKey).Error
}

// GetByID 根据 ID 获取 API Key
func (r *APIKeyRepo) GetByID(ctx context.Context, id uint) (*system.SysAPIKey, error) {
	var apiKey system.SysAPIKey
	err := r.db.WithContext(ctx).First(&apiKey, id).Error
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

// GetByKeyHash 根据 KeyHash 获取 API Key
func (r *APIKeyRepo) GetByKeyHash(ctx context.Context, keyHash string) (*system.SysAPIKey, error) {
	var apiKey system.SysAPIKey
	err := r.db.WithContext(ctx).Where("key_hash = ?", keyHash).First(&apiKey).Error
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

// List 获取 API Key 列表
func (r *APIKeyRepo) List(ctx context.Context, page, pageSize int) ([]*system.SysAPIKey, int64, error) {
	var apiKeys []*system.SysAPIKey
	var total int64

	// 查询总数
	if err := r.db.WithContext(ctx).Model(&system.SysAPIKey{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&apiKeys).Error

	if err != nil {
		return nil, 0, err
	}

	return apiKeys, total, nil
}

// Delete 删除 API Key
func (r *APIKeyRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&system.SysAPIKey{}, id).Error
}

// GetAll 获取所有 API Key（用于统计）
func (r *APIKeyRepo) GetAll(ctx context.Context) ([]*system.SysAPIKey, error) {
	var apiKeys []*system.SysAPIKey
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&apiKeys).Error
	if err != nil {
		return nil, err
	}
	return apiKeys, nil
}
