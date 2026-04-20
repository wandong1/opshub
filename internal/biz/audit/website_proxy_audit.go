// Copyright (c) 2026 DYCloud J.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED.

package audit

import (
	"context"
	"time"
)

// WebsiteProxyAuditLog 网站代理访问审计日志
type WebsiteProxyAuditLog struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	WebsiteID     uint      `gorm:"column:website_id;not null;index;comment:站点ID" json:"websiteId"`
	WebsiteName   string    `gorm:"column:website_name;type:varchar(100);not null;comment:站点名称" json:"websiteName"`
	UserID        uint      `gorm:"column:user_id;not null;index;comment:用户ID" json:"userId"`
	Username      string    `gorm:"column:username;type:varchar(100);not null;index;comment:用户名" json:"username"`
	AgentHostID   uint      `gorm:"column:agent_host_id;not null;comment:Agent主机ID" json:"agentHostId"`
	RequestMethod string    `gorm:"column:request_method;type:varchar(10);not null;comment:请求方法" json:"requestMethod"`
	RequestURL    string    `gorm:"column:request_url;type:text;not null;comment:请求URL" json:"requestUrl"`
	Status        string    `gorm:"column:status;type:varchar(20);not null;index;comment:状态 success/failed" json:"status"`
	StatusCode    int       `gorm:"column:status_code;comment:HTTP状态码" json:"statusCode"`
	ClientIP      string    `gorm:"column:client_ip;type:varchar(50);comment:客户端IP" json:"clientIp"`
	UserAgent     string    `gorm:"column:user_agent;type:varchar(500);comment:User-Agent" json:"userAgent"`
	RequestType   string    `gorm:"column:request_type;type:varchar(20);comment:请求类型 page/api/xhr" json:"requestType"`
	ErrorMessage  string    `gorm:"column:error_message;type:text;comment:错误信息" json:"errorMessage"`
	AccessTime    time.Time `gorm:"column:access_time;not null;index;comment:访问时间" json:"accessTime"`
}

// TableName 指定表名
func (WebsiteProxyAuditLog) TableName() string {
	return "website_proxy_audit_logs"
}

// WebsiteProxyAuditRepo 网站代理访问审计仓储接口
type WebsiteProxyAuditRepo interface {
	Create(ctx context.Context, log *WebsiteProxyAuditLog) error
	BatchCreate(ctx context.Context, logs []*WebsiteProxyAuditLog) error
	List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]*WebsiteProxyAuditLog, int64, error)
	Delete(ctx context.Context, id uint) error
	BatchDelete(ctx context.Context, ids []uint) error
}

// WebsiteProxyAuditUseCase 网站代理访问审计用例
type WebsiteProxyAuditUseCase struct {
	repo WebsiteProxyAuditRepo
}

// NewWebsiteProxyAuditUseCase 创建网站代理访问审计用例
func NewWebsiteProxyAuditUseCase(repo WebsiteProxyAuditRepo) *WebsiteProxyAuditUseCase {
	return &WebsiteProxyAuditUseCase{repo: repo}
}

// Create 创建审计日志
func (uc *WebsiteProxyAuditUseCase) Create(ctx context.Context, log *WebsiteProxyAuditLog) error {
	return uc.repo.Create(ctx, log)
}

// BatchCreate 批量创建审计日志
func (uc *WebsiteProxyAuditUseCase) BatchCreate(ctx context.Context, logs []*WebsiteProxyAuditLog) error {
	return uc.repo.BatchCreate(ctx, logs)
}

// List 查询审计日志列表
func (uc *WebsiteProxyAuditUseCase) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]*WebsiteProxyAuditLog, int64, error) {
	return uc.repo.List(ctx, page, pageSize, filters)
}

// Delete 删除审计日志
func (uc *WebsiteProxyAuditUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

// BatchDelete 批量删除审计日志
func (uc *WebsiteProxyAuditUseCase) BatchDelete(ctx context.Context, ids []uint) error {
	return uc.repo.BatchDelete(ctx, ids)
}
