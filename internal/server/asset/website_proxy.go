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
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// AgentHubInterface 定义 AgentHub 接口，避免循环导入
type AgentHubInterface interface {
	IsOnline(hostID uint) bool
	SendProbeRequest(hostID uint, req *pb.ProbeRequest) (*pb.ProbeResult, error)
}

type WebsiteProxyHandler struct {
	websiteUseCase *asset.WebsiteUseCase
	agentHub       AgentHubInterface
}

func NewWebsiteProxyHandler(websiteUseCase *asset.WebsiteUseCase, agentHub AgentHubInterface) *WebsiteProxyHandler {
	return &WebsiteProxyHandler{
		websiteUseCase: websiteUseCase,
		agentHub:       agentHub,
	}
}

// AccessWebsite 访问站点（内部站点通过Agent代理）
func (h *WebsiteProxyHandler) AccessWebsite(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ErrorCode(c, 400, "无效的站点ID")
		return
	}

	website, err := h.websiteUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, 500, "获取站点信息失败: "+err.Error())
		return
	}

	// 外部站点直接返回URL
	if website.Type == "external" {
		response.Success(c, gin.H{
			"type": "external",
			"url":  website.URL,
		})
		return
	}

	// 内部站点需要通过Agent代理
	if len(website.AgentHostIDs) == 0 {
		response.ErrorCode(c, 400, "内部站点未绑定Agent主机")
		return
	}

	// 查找在线的Agent
	var onlineHostID uint
	for _, hostID := range website.AgentHostIDs {
		if h.agentHub.IsOnline(hostID) {
			onlineHostID = hostID
			break
		}
	}

	if onlineHostID == 0 {
		response.ErrorCode(c, 503, "Agent主机离线，无法访问")
		return
	}

	// 返回代理访问信息（包含站点专用 token）
	response.Success(c, gin.H{
		"type":     "internal",
		"proxyUrl": fmt.Sprintf("/api/v1/websites/%d/proxy/?token=%s", id, website.ProxyToken),
		"hostId":   onlineHostID,
	})
}

// ProxyRequest 代理请求到内部站点
func (h *WebsiteProxyHandler) ProxyRequest(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的站点ID"})
		return
	}

	website, err := h.websiteUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取站点信息失败"})
		return
	}

	if website.Type != "internal" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "仅支持内部站点代理"})
		return
	}

	// 查找在线的Agent
	var onlineHostID uint
	for _, hostID := range website.AgentHostIDs {
		if h.agentHub.IsOnline(hostID) {
			onlineHostID = hostID
			break
		}
	}

	if onlineHostID == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "Agent主机离线"})
		return
	}

	// 通过Agent代理请求
	h.proxyViaAgent(c, onlineHostID, website.URL)
}

// proxyViaAgent 通过Agent拨测功能代理HTTP请求
func (h *WebsiteProxyHandler) proxyViaAgent(c *gin.Context, hostID uint, baseURL string) {
	// 读取请求体
	body, _ := io.ReadAll(c.Request.Body)

	// 构建完整URL
	fullURL := baseURL
	if c.Request.URL.Path != "" {
		// 移除 /api/v1/websites/:id/proxy 前缀
		path := c.Param("path")
		if path != "" && path != "/" {
			fullURL = strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(path, "/")
		}
	}

	// 添加查询参数（排除 token 参数）
	if c.Request.URL.RawQuery != "" {
		query := c.Request.URL.Query()
		query.Del("token") // 移除 token 参数，不传递给目标站点
		if len(query) > 0 {
			fullURL += "?" + query.Encode()
		}
	}

	// 构建拨测请求
	probeReq := &pb.ProbeRequest{
		RequestId:           uuid.New().String(),
		ProbeType:           "http",
		Method:              c.Request.Method,
		Url:                 fullURL,
		Headers:             make(map[string]string),
		Body:                string(body),
		Timeout:             30,
		MaxResponseBodySize: 0, // 0 表示不限制响应体大小（代理场景）
	}

	// 复制请求头
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			// 跳过一些不需要转发的头
			if key == "Host" || key == "Connection" || strings.HasPrefix(key, "X-Forwarded") || key == "Authorization" {
				continue
			}
			probeReq.Headers[key] = values[0]
		}
	}

	// 通过Agent发送请求
	result, err := h.agentHub.SendProbeRequest(hostID, probeReq)
	if err != nil {
		c.String(http.StatusBadGateway, "代理请求失败: "+err.Error())
		return
	}

	if !result.Success {
		c.String(http.StatusBadGateway, "代理请求失败: "+result.Error)
		return
	}

	// 获取站点ID用于构建代理路径
	siteID := c.Param("id")
	proxyBasePath := fmt.Sprintf("/api/v1/websites/%s/proxy/", siteID)

	// 重写响应头
	rewrittenHeaders := h.rewriteResponseHeaders(result.ResponseHeaders, siteID, baseURL)

	// 检测HTML响应并注入<base>标签
	contentType := rewrittenHeaders["Content-Type"]
	responseBody := result.ResponseBody
	if strings.Contains(strings.ToLower(contentType), "text/html") {
		responseBody = injectBaseTag(responseBody, proxyBasePath)
	}

	// 设置响应头
	for key, value := range rewrittenHeaders {
		c.Header(key, value)
	}

	// 返回响应体
	c.Data(int(result.HttpStatusCode), contentType, responseBody)
}

// rewriteResponseHeaders 重写响应头
func (h *WebsiteProxyHandler) rewriteResponseHeaders(headers map[string]string, siteID string, baseURL string) map[string]string {
	rewritten := make(map[string]string)
	proxyBasePath := fmt.Sprintf("/api/v1/websites/%s/proxy", siteID)

	for key, value := range headers {
		lowerKey := strings.ToLower(key)

		// 跳过不应转发的头
		if lowerKey == "content-encoding" || lowerKey == "content-length" || lowerKey == "transfer-encoding" {
			continue
		}

		// 重写 Location 头
		if lowerKey == "location" {
			value = rewriteLocationHeader(value, baseURL, proxyBasePath)
		}

		// 重写 Set-Cookie 头
		if lowerKey == "set-cookie" {
			value = rewriteSetCookieHeader(value, proxyBasePath)
		}

		// 重写 Content-Security-Policy 头
		if lowerKey == "content-security-policy" {
			value = rewriteCSPHeader(value, proxyBasePath)
		}

		rewritten[key] = value
	}

	return rewritten
}

// rewriteLocationHeader 重写 Location 响应头
func rewriteLocationHeader(location, baseURL, proxyBasePath string) string {
	// 如果是相对路径，转换为代理路径
	if strings.HasPrefix(location, "/") {
		return proxyBasePath + location
	}

	// 如果是完整 URL，检查是否是目标站点的 URL
	if strings.HasPrefix(location, baseURL) {
		relativePath := strings.TrimPrefix(location, baseURL)
		return proxyBasePath + relativePath
	}

	// 其他情况保持不变
	return location
}

// rewriteSetCookieHeader 重写 Set-Cookie 响应头
func rewriteSetCookieHeader(cookie, proxyBasePath string) string {
	// 移除 Domain 属性
	cookie = regexp.MustCompile(`;\s*Domain=[^;]+`).ReplaceAllString(cookie, "")

	// 修改 Path 属性
	if strings.Contains(cookie, "Path=") {
		cookie = regexp.MustCompile(`Path=[^;]+`).ReplaceAllString(cookie, "Path="+proxyBasePath)
	} else {
		cookie += "; Path=" + proxyBasePath
	}

	return cookie
}

// rewriteCSPHeader 重写 Content-Security-Policy 响应头
func rewriteCSPHeader(csp, proxyBasePath string) string {
	// 在 default-src 中添加代理路径
	// 简化实现：直接添加到末尾
	return csp + "; default-src " + proxyBasePath
}

// injectBaseTag 注入 <base> 标签到 HTML
func injectBaseTag(htmlBody []byte, basePath string) []byte {
	html := string(htmlBody)

	// 查找 <head> 标签
	headStart := strings.Index(strings.ToLower(html), "<head")
	if headStart == -1 {
		// 没有 <head> 标签，尝试在 <html> 后插入
		htmlStart := strings.Index(strings.ToLower(html), "<html")
		if htmlStart == -1 {
			return htmlBody // 无法处理，返回原始内容
		}
		htmlEnd := strings.Index(html[htmlStart:], ">")
		if htmlEnd == -1 {
			return htmlBody
		}
		insertPos := htmlStart + htmlEnd + 1
		baseTag := fmt.Sprintf("<head><base href=\"%s\"></head>", basePath)
		return []byte(html[:insertPos] + baseTag + html[insertPos:])
	}

	// 找到 <head> 标签的结束位置
	headEnd := strings.Index(html[headStart:], ">")
	if headEnd == -1 {
		return htmlBody
	}
	insertPos := headStart + headEnd + 1

	// 检查是否已有 <base> 标签
	baseTagPattern := regexp.MustCompile(`<base\s+[^>]*href\s*=\s*["']([^"']+)["'][^>]*>`)
	if baseTagPattern.MatchString(html) {
		// 替换现有的 <base> 标签
		html = baseTagPattern.ReplaceAllString(html, fmt.Sprintf(`<base href="%s">`, basePath))
		return []byte(html)
	}

	// 注入新的 <base> 标签
	baseTag := fmt.Sprintf(`<base href="%s">`, basePath)
	return []byte(html[:insertPos] + baseTag + html[insertPos:])
}
