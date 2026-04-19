// Copyright (c) 2026 DYCloud J.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED.

package audit

import (
	"context"

	auditbiz "github.com/ydcloud-dy/opshub/internal/biz/audit"
	"gorm.io/gorm"
)

type websiteProxyAuditRepo struct {
	db *gorm.DB
}

// NewWebsiteProxyAuditRepo 创建网站代理访问审计仓储
func NewWebsiteProxyAuditRepo(db *gorm.DB) auditbiz.WebsiteProxyAuditRepo {
	return &websiteProxyAuditRepo{db: db}
}

// Create 创建审计日志
func (r *websiteProxyAuditRepo) Create(ctx context.Context, log *auditbiz.WebsiteProxyAuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// BatchCreate 批量创建审计日志
func (r *websiteProxyAuditRepo) BatchCreate(ctx context.Context, logs []*auditbiz.WebsiteProxyAuditLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(logs).Error
}

// List 查询审计日志列表
func (r *websiteProxyAuditRepo) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]*auditbiz.WebsiteProxyAuditLog, int64, error) {
	var logs []*auditbiz.WebsiteProxyAuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&auditbiz.WebsiteProxyAuditLog{})

	// 应用过滤条件
	if username, ok := filters["username"].(string); ok && username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if websiteName, ok := filters["websiteName"].(string); ok && websiteName != "" {
		query = query.Where("website_name LIKE ?", "%"+websiteName+"%")
	}
	if websiteID, ok := filters["websiteId"].(uint); ok && websiteID > 0 {
		query = query.Where("website_id = ?", websiteID)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if method, ok := filters["method"].(string); ok && method != "" {
		query = query.Where("request_method = ?", method)
	}
	if startTime, ok := filters["startTime"].(string); ok && startTime != "" {
		query = query.Where("access_time >= ?", startTime)
	}
	if endTime, ok := filters["endTime"].(string); ok && endTime != "" {
		query = query.Where("access_time <= ?", endTime)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("access_time DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// Delete 删除审计日志
func (r *websiteProxyAuditRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&auditbiz.WebsiteProxyAuditLog{}, id).Error
}

// BatchDelete 批量删除审计日志
func (r *websiteProxyAuditRepo) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Delete(&auditbiz.WebsiteProxyAuditLog{}, ids).Error
}
