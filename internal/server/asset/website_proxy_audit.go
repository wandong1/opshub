// Copyright (c) 2026 DYCloud J.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED.

package asset

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	auditbiz "github.com/ydcloud-dy/opshub/internal/biz/audit"
	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
)

// shouldAuditWebsiteRequest 判断是否应该审计该请求
func (h *WebsiteProxyHandlerV2) shouldAuditWebsiteRequest(method string, path string, contentType string) bool {
	// 跳过 OPTIONS 预检请求
	if method == "OPTIONS" {
		return false
	}

	// 提取文件扩展名
	ext := ""
	if idx := strings.LastIndex(path, "."); idx != -1 {
		ext = strings.ToLower(path[idx:])
	}

	// 跳过明确的静态资源
	staticExts := []string{
		".js", ".css", ".map",
		".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".webp",
		".woff", ".woff2", ".ttf", ".eot", ".otf",
		".mp4", ".webm", ".ogg",
	}
	for _, staticExt := range staticExts {
		if ext == staticExt {
			return false
		}
	}

	// 记录以下类型的请求：
	// 1. HTML 页面请求
	if strings.Contains(contentType, "text/html") {
		return true
	}

	// 2. API 请求（路径包含 /api/）
	if strings.Contains(path, "/api/") {
		return true
	}

	// 3. XHR/JSON 请求
	if strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "application/xml") {
		return true
	}

	// 4. 无扩展名的路径（可能是 SPA 路由）
	if ext == "" && path != "/" {
		return true
	}

	return false
}

// enqueueAuditEvent 入队审计事件
func (h *WebsiteProxyHandlerV2) enqueueAuditEvent(
	c *gin.Context,
	website *asset.WebsiteVO,
	proxyCtx *proxyContext,
	agentHostID uint,
	status string,
	statusCode int,
	errorMessage string,
) {
	// 如果审计服务未注入，直接返回
	if h.auditService == nil {
		return
	}

	// 判断是否应该审计
	contentType := c.GetHeader("Accept")
	if !h.shouldAuditWebsiteRequest(c.Request.Method, proxyCtx.targetURL.Path, contentType) {
		return
	}

	// 从上下文中获取可信用户身份（由 accessKey 恢复）
	userID, _ := c.Get("access_user_id")
	username, _ := c.Get("access_username")
	websiteName, _ := c.Get("access_website_name")

	// 如果没有可信身份，跳过审计（降级场景）
	uid, ok := userID.(uint)
	if !ok || uid == 0 {
		return
	}
	uname, ok := username.(string)
	if !ok || uname == "" {
		return
	}

	// 使用快照的站点名称，如果没有则使用当前站点名称
	wsName, ok := websiteName.(string)
	if !ok || wsName == "" {
		wsName = website.Name
	}

	// 脱敏请求 URL（移除敏感 query 参数）
	sanitizedURL := h.sanitizeURL(proxyCtx.targetURL.String())

	// 判断请求类型
	requestType := h.detectRequestType(c.Request.Method, proxyCtx.targetURL.Path, contentType)

	// 构建审计事件
	event := &auditbiz.WebsiteProxyAuditLog{
		WebsiteID:     website.ID,
		WebsiteName:   wsName,
		UserID:        uid,
		Username:      uname,
		AgentHostID:   agentHostID,
		RequestMethod: c.Request.Method,
		RequestURL:    sanitizedURL,
		Status:        status,
		StatusCode:    statusCode,
		ClientIP:      c.ClientIP(),
		UserAgent:     c.Request.UserAgent(),
		RequestType:   requestType,
		ErrorMessage:  errorMessage,
		AccessTime:    time.Now(),
	}

	// 异步入队（fail-open）
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("审计事件入队 panic", zap.Any("recover", r))
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		h.auditService.EnqueueAuditEvent(ctx, event)
	}()
}

// sanitizeURL 脱敏 URL（移除敏感 query 参数）
func (h *WebsiteProxyHandlerV2) sanitizeURL(rawURL string) string {
	// 移除可能包含敏感信息的 query 参数
	sensitiveParams := []string{"token", "password", "secret", "key", "auth", "session"}

	for _, param := range sensitiveParams {
		// 简单替换，避免复杂的 URL 解析
		patterns := []string{
			param + "=",
			param + "%3D", // URL 编码的 =
		}
		for _, pattern := range patterns {
			if idx := strings.Index(rawURL, pattern); idx != -1 {
				// 找到参数位置，替换为 ***
				end := idx + len(pattern)
				nextAmp := strings.Index(rawURL[end:], "&")
				if nextAmp == -1 {
					rawURL = rawURL[:end] + "***"
				} else {
					rawURL = rawURL[:end] + "***" + rawURL[end+nextAmp:]
				}
			}
		}
	}

	return rawURL
}

// detectRequestType 检测请求类型
func (h *WebsiteProxyHandlerV2) detectRequestType(method string, path string, contentType string) string {
	// API 请求
	if strings.Contains(path, "/api/") {
		return "api"
	}

	// XHR 请求
	if strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "application/xml") {
		return "xhr"
	}

	// HTML 页面请求
	if strings.Contains(contentType, "text/html") {
		return "page"
	}

	// 默认为页面请求
	return "page"
}
