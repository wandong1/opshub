// Copyright (c) 2026 DYCloud J.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED.

package audit

import (
	"context"

	auditbiz "github.com/ydcloud-dy/opshub/internal/biz/audit"
)

type WebsiteProxyAuditQueryService struct {
	useCase *auditbiz.WebsiteProxyAuditUseCase
}

func NewWebsiteProxyAuditQueryService(useCase *auditbiz.WebsiteProxyAuditUseCase) *WebsiteProxyAuditQueryService {
	return &WebsiteProxyAuditQueryService{useCase: useCase}
}

// List 查询审计日志列表
func (s *WebsiteProxyAuditQueryService) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]*auditbiz.WebsiteProxyAuditLog, int64, error) {
	return s.useCase.List(ctx, page, pageSize, filters)
}

// Delete 删除审计日志
func (s *WebsiteProxyAuditQueryService) Delete(ctx context.Context, id uint) error {
	return s.useCase.Delete(ctx, id)
}

// BatchDelete 批量删除审计日志
func (s *WebsiteProxyAuditQueryService) BatchDelete(ctx context.Context, ids []uint) error {
	return s.useCase.BatchDelete(ctx, ids)
}
