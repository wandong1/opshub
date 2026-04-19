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

package audit

import (
	"github.com/redis/go-redis/v9"
	"github.com/ydcloud-dy/opshub/internal/biz/audit"
	auditdata "github.com/ydcloud-dy/opshub/internal/data/audit"
	auditservice "github.com/ydcloud-dy/opshub/internal/service/audit"
	"gorm.io/gorm"
)

// NewAuditServices 创建审计模块的所有服务
func NewAuditServices(db *gorm.DB, rdb *redis.Client) (
	operationLogService *auditservice.OperationLogService,
	loginLogService *auditservice.LoginLogService,
	dataLogService *auditservice.DataLogService,
	mwAuditLogService *auditservice.MiddlewareAuditLogService,
	websiteProxyAuditService *auditservice.WebsiteProxyAuditService,
	websiteProxyAuditWorker *auditservice.WebsiteProxyAuditWorker,
	websiteProxyAuditQueryService *auditservice.WebsiteProxyAuditQueryService,
) {
	// 初始化Repository
	operationLogRepo := auditdata.NewOperationLogRepo(db)
	loginLogRepo := auditdata.NewLoginLogRepo(db)
	dataLogRepo := auditdata.NewDataLogRepo(db)
	mwAuditLogRepo := auditdata.NewMiddlewareAuditLogRepo(db)
	websiteProxyAuditRepo := auditdata.NewWebsiteProxyAuditRepo(db)

	// 初始化UseCase
	operationLogUseCase := audit.NewOperationLogUseCase(operationLogRepo)
	loginLogUseCase := audit.NewLoginLogUseCase(loginLogRepo)
	dataLogUseCase := audit.NewDataLogUseCase(dataLogRepo)
	mwAuditLogUseCase := audit.NewMiddlewareAuditLogUseCase(mwAuditLogRepo)
	websiteProxyAuditUseCase := audit.NewWebsiteProxyAuditUseCase(websiteProxyAuditRepo)

	// 初始化Service
	operationLogService = auditservice.NewOperationLogService(operationLogUseCase)
	loginLogService = auditservice.NewLoginLogService(loginLogUseCase)
	dataLogService = auditservice.NewDataLogService(dataLogUseCase)
	mwAuditLogService = auditservice.NewMiddlewareAuditLogService(mwAuditLogUseCase)

	// 初始化网站代理访问审计服务（如果 Redis 可用）
	if rdb != nil {
		websiteProxyAuditService = auditservice.NewWebsiteProxyAuditService(websiteProxyAuditUseCase, rdb)
		websiteProxyAuditWorker = auditservice.NewWebsiteProxyAuditWorker(rdb, db, websiteProxyAuditUseCase)
	}

	// 查询服务（总是可用）
	websiteProxyAuditQueryService = auditservice.NewWebsiteProxyAuditQueryService(websiteProxyAuditUseCase)

	return
}
