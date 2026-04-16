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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
	"github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"go.uber.org/zap"
)

// AgentHubInterfaceV2 定义 AgentHub 接口（避免循环导入）
type AgentHubInterfaceV2 interface {
	IsOnline(hostID uint) bool
	GetByHostID(hostID uint) (AgentStreamInterface, bool)
	WaitResponse(as AgentStreamInterface, requestID string, timeout time.Duration) (interface{}, error)
}

// AgentStreamInterface 定义 AgentStream 接口
type AgentStreamInterface interface {
	Send(msg *pb.ServerMessage) error
}

// WebsiteProxyHandlerV2 新版站点代理处理器（使用真实的 HTTP 代理）
type WebsiteProxyHandlerV2 struct {
	websiteUseCase *asset.WebsiteUseCase
	agentHub       AgentHubInterfaceV2
}

func NewWebsiteProxyHandlerV2(websiteUseCase *asset.WebsiteUseCase, agentHub AgentHubInterfaceV2) *WebsiteProxyHandlerV2 {
	return &WebsiteProxyHandlerV2{
		websiteUseCase: websiteUseCase,
		agentHub:       agentHub,
	}
}

// ProxyWebsiteRequest 代理站点请求（完全透明的 HTTP 代理）
func (h *WebsiteProxyHandlerV2) ProxyWebsiteRequest(c *gin.Context) {
	// 提取站点 ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的站点ID")
		return
	}

	// 查询站点信息
	website, err := h.websiteUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取站点信息失败: "+err.Error())
		return
	}

	// 验证站点类型
	if website.Type != "internal" {
		response.ErrorCode(c, http.StatusBadRequest, "仅支持内部站点代理")
		return
	}

	// 查找在线的 Agent
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

	logger.Debug("站点代理，Agent 选取成功", zap.Uint("agent", onlineHostID))

	// 构建目标 URL
	proxyPath := c.Request.URL.Path
	// 移除 /api/v1/websites/:id/proxy 前缀
	prefix := fmt.Sprintf("/api/v1/websites/%d/proxy", id)
	targetPath := strings.TrimPrefix(proxyPath, prefix)
	if targetPath == "" {
		targetPath = "/"
	}

	// 使用站点的 URL 加上目标路径（避免双斜杠问题）
	baseURL := strings.TrimRight(website.URL, "/")
	targetURL := baseURL + targetPath
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	logger.Info("站点代理转发请求",
		zap.Uint("website_id", uint(id)),
		zap.String("website_name", website.Name),
		zap.String("website_url", website.URL),
		zap.String("proxy_path", proxyPath),
		zap.String("target_path", targetPath),
		zap.String("target_url", targetURL),
		zap.String("method", c.Request.Method),
		zap.String("query", c.Request.URL.RawQuery),
	)

	// 转发请求到 Agent
	if err := h.forwardToAgent(c, onlineHostID, targetURL, website); err != nil {
		logger.Error("站点代理转发失败",
			zap.String("target_url", targetURL),
			zap.Error(err),
		)
		response.ErrorCode(c, http.StatusBadGateway, "转发请求失败: "+err.Error())
		return
	}
}

// forwardToAgent 通过 Agent 转发 HTTP 请求
func (h *WebsiteProxyHandlerV2) forwardToAgent(c *gin.Context, agentHostID uint, targetURL string, website *asset.WebsiteVO) error {
	logger.Info("开始通过Agent转发站点请求",
		zap.Uint("agent_host_id", agentHostID),
		zap.String("target_url", targetURL),
	)

	// 获取 Agent 连接
	as, ok := h.agentHub.GetByHostID(agentHostID)
	if !ok {
		logger.Error("Agent未连接", zap.Uint("agent_host_id", agentHostID))
		return fmt.Errorf("Agent 未连接")
	}

	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("读取请求体失败", zap.Error(err))
		return fmt.Errorf("读取请求体失败: %w", err)
	}

	// 构建请求头
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// 如果站点配置了访问用户名和密码，添加 Basic Auth
	// if website.AccessUser != "" && website.AccessPassword != "" {
	// 	auth := website.AccessUser + ":" + website.AccessPassword
	// 	headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
	// 	logger.Debug("添加Basic Auth认证", zap.String("username", website.AccessUser))
	// }

	// 生成请求 ID
	requestID := uuid.New().String()

	logger.Info("构建HttpProxyRequest",
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("url", targetURL),
		zap.Int("body_len", len(body)),
	)

	// 构建 HttpProxyRequest
	proxyReq := &pb.HttpProxyRequest{
		RequestId: requestID,
		Method:    c.Request.Method,
		Url:       targetURL,
		Headers:   headers,
		Body:      body,
		Timeout:   30,
	}

	// 发送给 Agent
	msg := &pb.ServerMessage{
		Payload: &pb.ServerMessage_HttpProxyRequest{
			HttpProxyRequest: proxyReq,
		},
	}

	if err := as.Send(msg); err != nil {
		logger.Error("发送请求到Agent失败",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return fmt.Errorf("发送请求失败: %w", err)
	}

	logger.Info("已发送请求到Agent，等待响应",
		zap.String("request_id", requestID),
	)

	// 等待响应
	result, err := h.agentHub.WaitResponse(as, requestID, 35*time.Second)
	if err != nil {
		logger.Error("等待Agent响应超时",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return fmt.Errorf("等待响应超时: %w", err)
	}

	proxyResp, ok := result.(*pb.HttpProxyResponse)
	if !ok {
		logger.Error("响应类型错误",
			zap.String("request_id", requestID),
		)
		return fmt.Errorf("响应类型错误")
	}

	logger.Info("收到Agent响应",
		zap.String("request_id", requestID),
		zap.Int32("status_code", proxyResp.StatusCode),
		zap.Int("body_len", len(proxyResp.Body)),
		zap.String("error", proxyResp.Error),
	)

	// 如果 Agent 返回了错误信息，但没有 HTTP 响应
	if proxyResp.Error != "" && proxyResp.StatusCode == 0 {
		logger.Error("Agent执行失败（无HTTP响应）",
			zap.String("request_id", requestID),
			zap.String("error", proxyResp.Error),
		)
		return fmt.Errorf("Agent 执行失败: %s", proxyResp.Error)
	}

	// 设置状态码（如果 Agent 没有返回状态码，默认 200）
	statusCode := int(proxyResp.StatusCode)
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	// 处理响应体：如果是 HTML/CSS/JS，需要重写资源路径
	responseBody := proxyResp.Body
	contentType := proxyResp.Headers["Content-Type"]
	if contentType == "" {
		contentType = proxyResp.Headers["content-type"]
	}

	// 构建代理前缀
	proxyPrefix := fmt.Sprintf("/api/v1/websites/%d/proxy", website.ID)

	// 判断是否需要重写内容
	needRewrite := false
	if strings.Contains(contentType, "text/html") ||
		strings.Contains(contentType, "text/css") ||
		strings.Contains(contentType, "application/javascript") ||
		strings.Contains(contentType, "text/javascript") {
		needRewrite = true
	}

	if needRewrite && len(responseBody) > 0 {
		logger.Debug("重写响应内容中的资源路径",
			zap.String("request_id", requestID),
			zap.String("content_type", contentType),
			zap.String("proxy_prefix", proxyPrefix),
		)

		rewrittenBody := h.rewriteResourcePaths(responseBody, proxyPrefix, website.URL)
		responseBody = rewrittenBody

		// 更新 Content-Length
		proxyResp.Headers["Content-Length"] = fmt.Sprintf("%d", len(responseBody))
	}

	// 设置响应头
	for key, value := range proxyResp.Headers {
		c.Header(key, value)
	}

	c.Status(statusCode)

	// 写入响应体
	if len(responseBody) > 0 {
		_, err = c.Writer.Write(responseBody)
		if err != nil {
			logger.Error("写入响应体失败",
				zap.String("request_id", requestID),
				zap.Error(err),
			)
			// 注意：此时响应头已经发送，无法再返回错误
			return err
		}
	}

	logger.Info("成功返回响应给客户端",
		zap.String("request_id", requestID),
		zap.Int("status_code", statusCode),
		zap.Int("body_len", len(responseBody)),
	)

	// 成功写入响应，返回 nil
	return nil
}

// rewriteResourcePaths 重写 HTML/CSS/JS 中的资源路径，添加代理前缀
func (h *WebsiteProxyHandlerV2) rewriteResourcePaths(body []byte, proxyPrefix string, baseURL string) []byte {
	content := string(body)

	// 1. 重写 HTML 中的资源引用
	// <link href="/css/style.css"> -> <link href="/api/v1/websites/1/proxy/css/style.css">
	// <script src="/js/app.js"> -> <script src="/api/v1/websites/1/proxy/js/app.js">
	// <img src="/img/logo.png"> -> <img src="/api/v1/websites/1/proxy/img/logo.png">
	// <a href="/page"> -> <a href="/api/v1/websites/1/proxy/page">

	// 匹配 href="/xxx" 和 src="/xxx" (绝对路径)
	reAbsolutePath := regexp.MustCompile(`(href|src)="(/[^"]*)"`)
	content = reAbsolutePath.ReplaceAllStringFunc(content, func(match string) string {
		// 提取属性名和路径
		parts := reAbsolutePath.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		attr := parts[1]  // href 或 src
		path := parts[2]  // /xxx

		// 跳过已经包含代理前缀的路径
		if strings.HasPrefix(path, proxyPrefix) {
			return match
		}

		// 跳过外部链接（http:// 或 https://）
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			return match
		}

		// 跳过特殊协议（data:, javascript:, mailto:, tel:）
		if strings.HasPrefix(path, "data:") || strings.HasPrefix(path, "javascript:") ||
			strings.HasPrefix(path, "mailto:") || strings.HasPrefix(path, "tel:") ||
			strings.HasPrefix(path, "#") {
			return match
		}

		// 添加代理前缀
		newPath := proxyPrefix + path
		return fmt.Sprintf(`%s="%s"`, attr, newPath)
	})

	// 2. 重写 CSS 中的 url() 引用
	// url(/fonts/font.woff) -> url(/api/v1/websites/1/proxy/fonts/font.woff)
	// url('/images/bg.png') -> url('/api/v1/websites/1/proxy/images/bg.png')
	// url("/images/bg.png") -> url("/api/v1/websites/1/proxy/images/bg.png")

	reCSSUrl := regexp.MustCompile(`url\(['"]?(/[^'")]+)['"]?\)`)
	content = reCSSUrl.ReplaceAllStringFunc(content, func(match string) string {
		parts := reCSSUrl.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		path := parts[1]

		// 跳过已经包含代理前缀的路径
		if strings.HasPrefix(path, proxyPrefix) {
			return match
		}

		// 跳过外部链接和特殊协议
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") ||
			strings.HasPrefix(path, "data:") {
			return match
		}

		// 添加代理前缀
		newPath := proxyPrefix + path
		// 保持原有的引号风格
		if strings.Contains(match, `"`) {
			return fmt.Sprintf(`url("%s")`, newPath)
		} else if strings.Contains(match, `'`) {
			return fmt.Sprintf(`url('%s')`, newPath)
		} else {
			return fmt.Sprintf(`url(%s)`, newPath)
		}
	})

	// 3. 重写 JavaScript 中的 fetch/XMLHttpRequest 等 API 调用
	// fetch('/api/data') -> fetch('/api/v1/websites/1/proxy/api/data')
	// 注意：这个比较复杂，只处理简单的字符串字面量情况

	// 匹配 fetch('xxx') 或 fetch("xxx")
	reFetch := regexp.MustCompile(`(fetch|XMLHttpRequest\.open)\s*\(\s*['"]([^'"]+)['"]`)
	content = reFetch.ReplaceAllStringFunc(content, func(match string) string {
		parts := reFetch.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		method := parts[1]
		url := parts[2]

		// 只处理相对路径（以 / 开头）
		if !strings.HasPrefix(url, "/") {
			return match
		}

		// 跳过已经包含代理前缀的路径
		if strings.HasPrefix(url, proxyPrefix) {
			return match
		}

		// 跳过外部链接
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			return match
		}

		// 添加代理前缀
		newURL := proxyPrefix + url
		// 保持原有的引号风格
		if strings.Contains(match, `"`) {
			return fmt.Sprintf(`%s("%s"`, method, newURL)
		} else {
			return fmt.Sprintf(`%s('%s'`, method, newURL)
		}
	})

	// 4. 重写 <base> 标签（如果存在）
	// <base href="/"> -> <base href="/api/v1/websites/1/proxy/">
	reBase := regexp.MustCompile(`<base\s+href=["']([^"']+)["']`)
	content = reBase.ReplaceAllStringFunc(content, func(match string) string {
		parts := reBase.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		href := parts[1]

		// 如果 base href 是 /，替换为代理前缀
		if href == "/" {
			return fmt.Sprintf(`<base href="%s/"`, proxyPrefix)
		}

		// 如果是相对路径，添加代理前缀
		if strings.HasPrefix(href, "/") && !strings.HasPrefix(href, proxyPrefix) {
			return fmt.Sprintf(`<base href="%s%s"`, proxyPrefix, href)
		}

		return match
	})

	return []byte(content)
}
