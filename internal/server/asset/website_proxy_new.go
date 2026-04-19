// Copyright (c) 2026 DYCloud J.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED.

package asset

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	auditbiz "github.com/ydcloud-dy/opshub/internal/biz/audit"
	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
	"github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"go.uber.org/zap"
)

type AgentHubInterfaceV2 interface {
	IsOnline(hostID uint) bool
	GetByHostID(hostID uint) (AgentStreamInterface, bool)
	WaitResponse(as AgentStreamInterface, requestID string, timeout time.Duration) (interface{}, error)
}

type AgentStreamInterface interface {
	Send(msg *pb.ServerMessage) error
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Strategy     string   // minimal/standard/hybrid/aggressive
	InjectScript bool     // 是否注入拦截脚本
	RewriteHTML  bool     // 是否重写 HTML
	RewriteCSS   bool     // 是否重写 CSS
	RewriteJS    bool     // 是否重写 JS
	Whitelist    []string // 白名单路径（正则）
	Blacklist    []string // 黑名单路径（正则）
	Debug        bool     // 调试模式
}

type proxyContext struct {
	websiteID      uint
	websiteBaseURL *url.URL
	targetURL      *url.URL
	proxyPath      string
	proxyBasePath  string
	proxyPrefix    string
	token          string
}

type WebsiteProxyHandlerV2 struct {
	websiteUseCase *asset.WebsiteUseCase
	agentHub       AgentHubInterfaceV2
	accessManager  *asset.WebsiteAccessManager
	auditService   AuditServiceInterface
}

type AuditServiceInterface interface {
	EnqueueAuditEvent(ctx context.Context, event *auditbiz.WebsiteProxyAuditLog)
}

func NewWebsiteProxyHandlerV2(websiteUseCase *asset.WebsiteUseCase, agentHub AgentHubInterfaceV2) *WebsiteProxyHandlerV2 {
	return &WebsiteProxyHandlerV2{
		websiteUseCase: websiteUseCase,
		agentHub:       agentHub,
		accessManager:  nil, // 延迟注入
		auditService:   nil, // 延迟注入
	}
}

// SetAccessManager 设置访问管理器（依赖注入）
func (h *WebsiteProxyHandlerV2) SetAccessManager(manager *asset.WebsiteAccessManager) {
	h.accessManager = manager
}

// SetAuditService 设置审计服务（依赖注入）
func (h *WebsiteProxyHandlerV2) SetAuditService(service AuditServiceInterface) {
	h.auditService = service
}

func (h *WebsiteProxyHandlerV2) ProxyWebsiteRequest(c *gin.Context) {
	// 从路径参数中提取 token
	token := c.Param("token")
	if token == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少代理访问Token")
		return
	}

	var website *asset.WebsiteVO
	var err error

	// 优先尝试从 accessManager 解析短期凭证
	if h.accessManager != nil {
		session, sessionErr := h.accessManager.ValidateAccessKey(c.Request.Context(), token)
		if sessionErr == nil {
			// 短期凭证有效，通过 websiteID 查找站点
			website, err = h.websiteUseCase.GetByID(c.Request.Context(), session.WebsiteID)
			if err != nil {
				response.ErrorCode(c, http.StatusNotFound, "站点不存在")
				return
			}

			// 恢复可信用户身份到上下文（供审计使用）
			c.Set("access_user_id", session.UserID)
			c.Set("access_username", session.Username)
			c.Set("access_website_name", session.WebsiteName)
		} else {
			// 短期凭证无效，降级尝试长期 proxy_token
			website, err = h.websiteUseCase.GetByProxyToken(c.Request.Context(), token)
			if err != nil {
				response.ErrorCode(c, http.StatusNotFound, "站点不存在或Token无效")
				return
			}
		}
	} else {
		// accessManager 未注入，直接使用长期 token
		website, err = h.websiteUseCase.GetByProxyToken(c.Request.Context(), token)
		if err != nil {
			response.ErrorCode(c, http.StatusNotFound, "站点不存在或Token无效")
			return
		}
	}

	if website.Type != "internal" {
		response.ErrorCode(c, http.StatusBadRequest, "仅支持内部站点代理")
		return
	}

	if len(website.AgentHostIDs) == 0 {
		response.ErrorCode(c, http.StatusServiceUnavailable, "站点未绑定Agent主机")
		return
	}

	var onlineHostID uint
	for _, hostID := range website.AgentHostIDs {
		if h.agentHub != nil && h.agentHub.IsOnline(hostID) {
			onlineHostID = hostID
			break
		}
	}

	if onlineHostID == 0 {
		response.ErrorCode(c, http.StatusServiceUnavailable, "Agent主机离线，无法访问")
		return
	}

	proxyCtx, err := h.buildProxyContext(c, website, token)
	if err != nil {
		logger.Warn("构建站点代理上下文失败",
			zap.Uint("website_id", uint(website.ID)),
			zap.String("website_name", website.Name),
			zap.Error(err),
		)
		response.ErrorCode(c, http.StatusBadRequest, "代理请求路径无效")
		return
	}

	logger.Info("站点代理转发请求",
		zap.Uint("website_id", uint(website.ID)),
		zap.String("website_name", website.Name),
		zap.String("target_url", proxyCtx.targetURL.String()),
		zap.String("method", c.Request.Method),
	)

	if err := h.forwardToAgent(c, onlineHostID, proxyCtx, website); err != nil {
		logger.Error("站点代理转发失败", zap.String("target_url", proxyCtx.targetURL.String()), zap.Error(err))

		// 记录失败的审计事件
		h.enqueueAuditEvent(c, website, proxyCtx, onlineHostID, "failed", 0, err.Error())

		response.ErrorCode(c, http.StatusBadGateway, "转发请求失败: "+err.Error())
		return
	}

	// 记录成功的审计事件（从响应上下文中获取状态码）
	statusCode := c.Writer.Status()
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	h.enqueueAuditEvent(c, website, proxyCtx, onlineHostID, "success", statusCode, "")
}

func (h *WebsiteProxyHandlerV2) buildProxyContext(c *gin.Context, website *asset.WebsiteVO, token string) (*proxyContext, error) {
	baseURL, err := url.Parse(strings.TrimRight(website.URL, "/"))
	if err != nil {
		return nil, fmt.Errorf("解析站点地址失败: %w", err)
	}

	proxyBasePath := fmt.Sprintf("/api/v1/websites/proxy/t/%s", token)
	proxyPath := c.Request.URL.Path
	targetPath := strings.TrimPrefix(proxyPath, proxyBasePath)
	if targetPath == "" {
		targetPath = "/"
	}
	if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}

	targetURL := baseURL.ResolveReference(&url.URL{Path: targetPath})
	targetURL.RawQuery = c.Request.URL.RawQuery

	return &proxyContext{
		websiteID:      website.ID,
		websiteBaseURL: baseURL,
		targetURL:      targetURL,
		proxyPath:      proxyPath,
		proxyBasePath:  proxyBasePath,
		proxyPrefix:    proxyBasePath + "/",
		token:          token,
	}, nil
}

func (h *WebsiteProxyHandlerV2) forwardToAgent(c *gin.Context, agentHostID uint, proxyCtx *proxyContext, website *asset.WebsiteVO) error {
	as, ok := h.agentHub.GetByHostID(agentHostID)
	if !ok {
		return fmt.Errorf("Agent 未连接")
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return fmt.Errorf("读取请求体失败: %w", err)
	}

	headers := h.buildForwardHeaders(c)

	requestID := uuid.New().String()
	proxyReq := &pb.HttpProxyRequest{
		RequestId: requestID,
		Method:    c.Request.Method,
		Url:       proxyCtx.targetURL.String(),
		Headers:   headers,
		Body:      body,
		Timeout:   30,
	}

	msg := &pb.ServerMessage{
		Payload: &pb.ServerMessage_HttpProxyRequest{HttpProxyRequest: proxyReq},
	}

	if err := as.Send(msg); err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}

	result, err := h.agentHub.WaitResponse(as, requestID, 35*time.Second)
	if err != nil {
		return fmt.Errorf("等待响应超时: %w", err)
	}

	proxyResp, ok := result.(*pb.HttpProxyResponse)
	if !ok {
		return fmt.Errorf("响应类型错误")
	}

	if proxyResp.Error != "" && proxyResp.StatusCode == 0 {
		return fmt.Errorf("Agent 执行失败: %s", proxyResp.Error)
	}

	statusCode := int(proxyResp.StatusCode)
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	responseBody, responseHeaders, _ := h.normalizeProxyResponse(proxyResp, proxyCtx, website)

	for key, value := range responseHeaders {
		c.Header(key, value)
	}

	c.Status(statusCode)
	if len(responseBody) > 0 {
		_, err = c.Writer.Write(responseBody)
		if err != nil {
			logger.Error("写入响应体失败", zap.String("request_id", requestID), zap.Error(err))
			return err
		}
	}

	return nil
}

func (h *WebsiteProxyHandlerV2) buildForwardHeaders(c *gin.Context) map[string]string {
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) == 0 {
			continue
		}

		if strings.EqualFold(key, "Host") ||
			strings.EqualFold(key, "Content-Length") ||
			strings.EqualFold(key, "Connection") ||
			strings.HasPrefix(strings.ToLower(key), "x-forwarded-") {
			continue
		}

		headers[key] = values[0]
	}
	return headers
}

func (h *WebsiteProxyHandlerV2) normalizeProxyResponse(proxyResp *pb.HttpProxyResponse, proxyCtx *proxyContext, website *asset.WebsiteVO) ([]byte, map[string]string, string) {
	responseHeaders := cloneHeaderMap(proxyResp.Headers)
	responseBody := proxyResp.Body
	contentType := firstHeaderValue(responseHeaders, "Content-Type")
	contentEncoding := firstHeaderValue(responseHeaders, "Content-Encoding")

	h.rewriteResponseHeaders(responseHeaders, proxyCtx)

	config := &ProxyConfig{
		Strategy:     website.ProxyStrategy,
		InjectScript: website.InjectScript,
		RewriteHTML:  website.RewriteHTML,
		RewriteCSS:   website.RewriteCSS,
		RewriteJS:    website.RewriteJS,
		Whitelist:    parseJSONArray(website.ProxyWhitelist),
		Blacklist:    parseJSONArray(website.ProxyBlacklist),
		Debug:        false,
	}

	if h.shouldRewriteBody(contentType) && len(responseBody) > 0 {
		if strings.Contains(strings.ToLower(contentEncoding), "gzip") {
			decompressed, err := h.decompressGzip(responseBody)
			if err == nil {
				responseBody = decompressed
				deleteHeader(responseHeaders, "Content-Encoding")
			}
		}

		responseBody = h.rewriteResourcePaths(responseBody, proxyCtx.proxyPrefix, proxyCtx.proxyBasePath, proxyCtx.token, contentType, config)
		setHeader(responseHeaders, "Content-Length", fmt.Sprintf("%d", len(responseBody)))
	}

	return responseBody, responseHeaders, contentType
}

func (h *WebsiteProxyHandlerV2) shouldRewriteBody(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.Contains(contentType, "text/html") ||
		strings.Contains(contentType, "text/css") ||
		strings.Contains(contentType, "application/javascript") ||
		strings.Contains(contentType, "text/javascript")
}

func (h *WebsiteProxyHandlerV2) rewriteResponseHeaders(headers map[string]string, proxyCtx *proxyContext) {
	for key, value := range cloneHeaderMap(headers) {
		lowerKey := strings.ToLower(key)
		switch lowerKey {
		case "content-length", "transfer-encoding":
			delete(headers, key)
		case "location":
			headers[key] = h.rewriteLocationHeader(value, proxyCtx)
		case "set-cookie":
			headers[key] = h.rewriteSetCookieHeader(value, proxyCtx.proxyBasePath)
		}
	}
}

func (h *WebsiteProxyHandlerV2) rewriteLocationHeader(location string, proxyCtx *proxyContext) string {
	if location == "" {
		return location
	}

	parsed, err := url.Parse(location)
	if err != nil {
		return location
	}

	if parsed.IsAbs() {
		if proxyCtx.websiteBaseURL == nil || !sameOriginURL(parsed, proxyCtx.websiteBaseURL) {
			return location
		}
		return proxyURLFromTargetURL(parsed, proxyCtx)
	}

	if strings.HasPrefix(location, "/") {
		return proxyCtx.proxyPrefix + strings.TrimPrefix(location, "/")
	}

	if proxyCtx.targetURL == nil {
		return location
	}

	resolved := proxyCtx.targetURL.ResolveReference(parsed)
	return proxyURLFromTargetURL(resolved, proxyCtx)
}

func (h *WebsiteProxyHandlerV2) rewriteSetCookieHeader(cookie string, proxyBasePath string) string {
	cookie = regexp.MustCompile(`(?i);\s*Domain=[^;]+`).ReplaceAllString(cookie, "")

	pathPattern := regexp.MustCompile(`(?i)Path=[^;]+`)
	if pathPattern.MatchString(cookie) {
		return pathPattern.ReplaceAllString(cookie, "Path="+proxyBasePath)
	}

	return cookie + "; Path=" + proxyBasePath
}

func cloneHeaderMap(src map[string]string) map[string]string {
	cloned := make(map[string]string, len(src))
	for key, value := range src {
		cloned[key] = value
	}
	return cloned
}

func firstHeaderValue(headers map[string]string, key string) string {
	if value, ok := headers[key]; ok {
		return value
	}
	lowerKey := strings.ToLower(key)
	for headerKey, value := range headers {
		if strings.ToLower(headerKey) == lowerKey {
			return value
		}
	}
	return ""
}

func deleteHeader(headers map[string]string, key string) {
	delete(headers, key)
	delete(headers, strings.ToLower(key))
}

func setHeader(headers map[string]string, key string, value string) {
	deleteHeader(headers, key)
	headers[key] = value
}

func sameOriginURL(a *url.URL, b *url.URL) bool {
	if a == nil || b == nil {
		return false
	}
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}

func proxyURLFromTargetURL(target *url.URL, proxyCtx *proxyContext) string {
	if target == nil {
		return proxyCtx.proxyBasePath
	}

	path := target.EscapedPath()
	if path == "" {
		path = "/"
	}

	proxyPath := proxyCtx.proxyPrefix + strings.TrimPrefix(path, "/")
	if strings.HasSuffix(proxyCtx.proxyPrefix, "/") && path == "/" {
		proxyPath = proxyCtx.proxyPrefix
	}

	if target.RawQuery != "" {
		proxyPath += "?" + target.RawQuery
	}
	if target.Fragment != "" {
		proxyPath += "#" + target.Fragment
	}
	return proxyPath
}

// parseJSONArray 解析 JSON 数组字符串
func parseJSONArray(jsonStr string) []string {
	if jsonStr == "" {
		return nil
	}

	var arr []string
	if err := json.Unmarshal([]byte(jsonStr), &arr); err != nil {
		logger.Warn("解析 JSON 数组失败", zap.String("json", jsonStr), zap.Error(err))
		return nil
	}

	return arr
}

var (
	// HTML 中的 src/href 属性（匹配所有路径，不限扩展名）
	reHtmlAttr = regexp.MustCompile(`(src|href)\s*=\s*["']([^"']+)["']`)

	// CSS 中的 url()
	reCssUrl = regexp.MustCompile(`url\(\s*["']?([^"')]+)["']?\s*\)`)

	// JS 中的资源路径（更精确的匹配，只匹配常见的资源扩展名）
	// 匹配 "/xxx.js" 或 "/xxx.css" 等明确的资源路径
	jsResourceExts     = `\.(js|css|json|png|jpg|jpeg|gif|svg|woff|woff2|ttf|eot|ico|webp|mp4|webm|xml|txt|pdf|vue|jsx|tsx)`
	reJsResourceDouble = regexp.MustCompile(`"(/[^"]*` + jsResourceExts + `[^"]*)"`)
	reJsResourceSingle = regexp.MustCompile(`'(/[^']*` + jsResourceExts + `[^']*)'`)

	// base 标签
	reBase = regexp.MustCompile(`<base\s+href=["']([^"']+)["'][^>]*>`)

	// head 标签
	reHead = regexp.MustCompile(`<head[^>]*>`)
)

func (h *WebsiteProxyHandlerV2) rewriteResourcePaths(body []byte, proxyPrefix string, proxyBasePath string, token string, contentType string, config *ProxyConfig) []byte {
	content := string(body)

	switch {
	case strings.Contains(contentType, "text/html"):
		// 注入拦截脚本（如果启用）
		if config.InjectScript {
			content = h.injectInterceptScript(content, proxyPrefix, token, config.Debug)
		}

		// 重写 HTML（如果启用）
		if config.RewriteHTML {
			// 注入 base 标签，使用带尾部斜杠的 proxyPrefix
			content = h.injectBaseTag(content, proxyPrefix)
			content = reHtmlAttr.ReplaceAllStringFunc(content, func(match string) string {
				return h.rewriteHtmlAttr(match, proxyPrefix, token, config)
			})
		}

	case strings.Contains(contentType, "text/css"):
		if config.RewriteCSS {
			content = reCssUrl.ReplaceAllStringFunc(content, func(match string) string {
				return h.rewriteCssUrl(match, proxyPrefix, token, config)
			})
		}

	case strings.Contains(contentType, "javascript"):
		// 只在启用 JS 重写时才处理（默认禁用）
		if config.RewriteJS {
			content = reJsResourceDouble.ReplaceAllStringFunc(content, func(match string) string {
				return h.rewriteJsPath(match, proxyPrefix, token, `"`, config)
			})
			content = reJsResourceSingle.ReplaceAllStringFunc(content, func(match string) string {
				return h.rewriteJsPath(match, proxyPrefix, token, `'`, config)
			})
		}
	}

	return []byte(content)
}

func appendProxyToken(rawURL string, token string) string {
	if rawURL == "" || token == "" {
		return rawURL
	}

	if strings.Contains(rawURL, "token=") {
		return rawURL
	}

	if strings.Contains(rawURL, "#") {
		parts := strings.SplitN(rawURL, "#", 2)
		return appendProxyToken(parts[0], token) + "#" + parts[1]
	}

	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + "token=" + token
}

// injectInterceptScript 注入拦截脚本到 HTML
func (h *WebsiteProxyHandlerV2) injectInterceptScript(content string, proxyPrefix string, token string, debug bool) string {
	script := GenerateProxyInterceptScript(proxyPrefix, token, debug)

	// 插入到 <head> 标签之后（优先级最高）
	content = reHead.ReplaceAllStringFunc(content, func(match string) string {
		return match + fmt.Sprintf("\n<script>%s</script>", script)
	})

	return content
}

// shouldRewritePath 判断是否应该重写路径（根据白名单/黑名单）
func (h *WebsiteProxyHandlerV2) shouldRewritePath(path string, config *ProxyConfig) bool {
	// 检查黑名单
	for _, pattern := range config.Blacklist {
		if matched, _ := regexp.MatchString(pattern, path); matched {
			return false
		}
	}

	// 检查白名单
	if len(config.Whitelist) > 0 {
		for _, pattern := range config.Whitelist {
			if matched, _ := regexp.MatchString(pattern, path); matched {
				return true
			}
		}
		return false
	}

	return true
}

// injectBaseTag 注入 base 标签到 HTML head 中
func (h *WebsiteProxyHandlerV2) injectBaseTag(content string, baseHref string) string {
	// 如果已有 base 标签，先移除（避免冲突）
	content = reBase.ReplaceAllString(content, "")

	// 注入 base 标签，使用完整的代理前缀
	baseTag := fmt.Sprintf(`<base href="%s">`, baseHref)

	// 插入到 <head> 标签之后
	content = reHead.ReplaceAllStringFunc(content, func(match string) string {
		return match + baseTag
	})

	return content
}

// rewriteHtmlAttr 重写 HTML 属性中的路径
func (h *WebsiteProxyHandlerV2) rewriteHtmlAttr(match string, proxyPrefix string, token string, config *ProxyConfig) string {
	parts := reHtmlAttr.FindStringSubmatch(match)
	if len(parts) != 3 {
		return match
	}

	attr := parts[1] // src 或 href
	path := parts[2]

	// 检查白名单/黑名单
	if !h.shouldRewritePath(path, config) {
		return match
	}

	// 跳过已经包含代理前缀的路径（检查通用代理路径前缀，避免重复拼接）
	if strings.HasPrefix(path, "/api/v1/websites/proxy/") {
		return match
	}

	// 跳过外部链接
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") ||
		strings.HasPrefix(path, "//") {
		return match
	}

	// 跳过特殊协议
	if strings.HasPrefix(path, "data:") || strings.HasPrefix(path, "javascript:") ||
		strings.HasPrefix(path, "mailto:") || strings.HasPrefix(path, "tel:") ||
		strings.HasPrefix(path, "#") || strings.HasPrefix(path, "blob:") {
		return match
	}

	// 跳过已经是相对路径的
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		return match
	}

	// 将绝对路径转换为带代理前缀的路径
	if strings.HasPrefix(path, "/") {
		newPath := proxyPrefix + strings.TrimPrefix(path, "/")
		return fmt.Sprintf(`%s="%s"`, attr, newPath)
	}

	// 相对路径保持不变（base 标签会处理）
	return match
}

// rewriteCssUrl 重写 CSS url() 中的路径
func (h *WebsiteProxyHandlerV2) rewriteCssUrl(match string, proxyPrefix string, token string, config *ProxyConfig) string {
	parts := reCssUrl.FindStringSubmatch(match)
	if len(parts) != 2 {
		return match
	}

	path := parts[1]

	// 检查白名单/黑名单
	if !h.shouldRewritePath(path, config) {
		return match
	}

	// 跳过已经包含代理前缀的路径（检查通用代理路径前缀，避免重复拼接）
	if strings.HasPrefix(path, "/api/v1/websites/proxy/") {
		return match
	}

	// 跳过外部链接和特殊协议
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") ||
		strings.HasPrefix(path, "//") || strings.HasPrefix(path, "data:") {
		return match
	}

	// 跳过相对路径
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		return match
	}

	// 将绝对路径转换为带代理前缀的路径
	if strings.HasPrefix(path, "/") {
		newPath := proxyPrefix + strings.TrimPrefix(path, "/")
		// 保持原有的引号风格
		if strings.Contains(match, `"`) {
			return fmt.Sprintf(`url("%s")`, newPath)
		} else if strings.Contains(match, `'`) {
			return fmt.Sprintf(`url('%s')`, newPath)
		}
		return fmt.Sprintf(`url(%s)`, newPath)
	}

	// 相对路径保持不变
	return match
}

// rewriteJsPath 重写 JS 中的资源路径
func (h *WebsiteProxyHandlerV2) rewriteJsPath(match string, proxyPrefix string, token string, quote string, config *ProxyConfig) string {
	var re *regexp.Regexp
	if quote == `"` {
		re = reJsResourceDouble
	} else {
		re = reJsResourceSingle
	}

	parts := re.FindStringSubmatch(match)
	if len(parts) != 2 {
		return match
	}

	path := parts[1] // /xxx.js

	// 检查白名单/黑名单
	if !h.shouldRewritePath(path, config) {
		return match
	}

	// 跳过已经包含代理前缀的路径（检查通用代理路径前缀，避免重复拼接）
	if strings.HasPrefix(path, "/api/v1/websites/proxy/") {
		return match
	}

	// 跳过外部链接
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return match
	}

	// 将绝对路径转换为带代理前缀的路径
	newPath := proxyPrefix + strings.TrimPrefix(path, "/")
	return fmt.Sprintf(`%s%s%s`, quote, newPath, quote)
}

func (h *WebsiteProxyHandlerV2) decompressGzip(body []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建gzip reader失败: %w", err)
	}
	defer reader.Close()

	return io.ReadAll(reader)
}
