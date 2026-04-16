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

package alert

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
	"github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"go.uber.org/zap"
)

// proxyDataSourceRequest 处理数据源代理请求
func (s *HTTPServer) proxyDataSourceRequest(c *gin.Context) {
	proxyToken := c.Param("token")

	// 查询数据源
	ds, err := s.dsRepo.GetByProxyToken(c.Request.Context(), proxyToken)
	if err != nil || ds == nil {
		response.ErrorCode(c, http.StatusNotFound, "数据源不存在")
		return
	}

	if ds.AccessMode != "agent" {
		response.ErrorCode(c, http.StatusBadRequest, "数据源不是Agent代理模式")
		return
	}

	// 获取在线Agent
	rels, err := s.dsAgentRelationRepo.ListByDataSourceID(c.Request.Context(), ds.ID)
	if err != nil || len(rels) == 0 {
		response.ErrorCode(c, http.StatusServiceUnavailable, "没有可用的Agent")
		return
	}

	var selectedRel *biz.DataSourceAgentRelation
	for _, rel := range rels {
		if s.agentHub != nil && s.agentHub.IsOnline(rel.AgentHostID) {
			selectedRel = rel
			break
		}
	}

	if selectedRel == nil {
		response.ErrorCode(c, http.StatusServiceUnavailable, "没有可用的Agent")
		return
	}

	logger.Debug("代理转发数据源，agent 选取成功", zap.Uint("agent", selectedRel.AgentHostID))

	// 构建请求URL
	proxyPath := c.Request.URL.Path
	// 移除 /api/v1/alert/proxy/datasource/{token} 前缀
	prefix := fmt.Sprintf("/api/v1/alert/proxy/datasource/%s", proxyToken)
	targetPath := strings.TrimPrefix(proxyPath, prefix)
	if targetPath == "" {
		targetPath = "/"
	}

	// 使用数据源的 URL 加上目标路径（避免双斜杠问题）
	baseURL := strings.TrimRight(ds.URL, "/")
	targetURL := baseURL + targetPath
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	logger.Info("代理转发请求",
		zap.String("proxy_token", proxyToken),
		zap.String("datasource_name", ds.Name),
		zap.String("datasource_url", ds.URL),
		zap.String("access_mode", ds.AccessMode),
		zap.String("proxy_path", proxyPath),
		zap.String("target_path", targetPath),
		zap.String("target_url", targetURL),
		zap.String("method", c.Request.Method),
		zap.String("query", c.Request.URL.RawQuery),
	)

	// 转发请求到Agent
	if err := s.forwardToAgent(c, selectedRel.AgentHostID, targetURL, ds); err != nil {
		logger.Error("转发请求失败",
			zap.String("target_url", targetURL),
			zap.Error(err),
		)
		response.ErrorCode(c, http.StatusBadGateway, "转发请求失败: "+err.Error())
		return
	}
}

// forwardToAgent 通过Agent转发HTTP请求
func (s *HTTPServer) forwardToAgent(c *gin.Context, agentHostID uint, targetURL string, ds *biz.AlertDataSource) error {
	logger.Info("开始通过Agent转发",
		zap.Uint("agent_host_id", agentHostID),
		zap.String("target_url", targetURL),
	)

	// 获取 Agent 连接
	as, ok := s.agentHub.GetByHostID(agentHostID)
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

	// 添加认证信息
	if ds.Token != "" {
		headers["Authorization"] = "Bearer " + ds.Token
		logger.Debug("添加Bearer Token认证")
	} else if ds.Username != "" {
		auth := ds.Username + ":" + ds.Password
		headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
		logger.Debug("添加Basic Auth认证", zap.String("username", ds.Username))
	}

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
	result, err := s.agentHub.WaitResponse(as, requestID, 35*time.Second)
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

	// 如果 Agent 返回了错误信息，但没有 HTTP 响应，则返回错误
	if proxyResp.Error != "" && proxyResp.StatusCode == 0 {
		logger.Error("Agent执行失败（无HTTP响应）",
			zap.String("request_id", requestID),
			zap.String("error", proxyResp.Error),
		)
		return fmt.Errorf("Agent 执行失败: %s", proxyResp.Error)
	}

	// 返回响应给客户端（包括错误响应，如 500）
	// 设置响应头
	for key, value := range proxyResp.Headers {
		c.Header(key, value)
	}

	// 设置状态码（如果 Agent 没有返回状态码，默认 200）
	statusCode := int(proxyResp.StatusCode)
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	c.Status(statusCode)

	// 写入响应体
	if len(proxyResp.Body) > 0 {
		_, err = c.Writer.Write(proxyResp.Body)
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
		zap.Int("body_len", len(proxyResp.Body)),
	)

	// 成功写入响应，返回 nil
	return nil
}
