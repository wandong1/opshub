// Copyright (c) 2026 DYCloud J.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED.

package asset

import (
	"fmt"
	"io"
	"net/http"
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

// AIModelProxyHandler AI模型代理处理器
type AIModelProxyHandler struct {
	proxyUseCase *asset.AIModelProxyUseCase
	agentHub     AgentHubInterfaceV2
}

func NewAIModelProxyHandler(proxyUseCase *asset.AIModelProxyUseCase, agentHub AgentHubInterfaceV2) *AIModelProxyHandler {
	return &AIModelProxyHandler{
		proxyUseCase: proxyUseCase,
		agentHub:     agentHub,
	}
}

// ProxyStreamRequest 流式代理请求处理（SSE支持）
func (h *AIModelProxyHandler) ProxyStreamRequest(c *gin.Context) {
	// 1. 提取Token
	token := c.Param("token")
	if token == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少代理访问Token")
		return
	}

	// 2. 验证Token并获取配置
	proxyConfig, agentHostIDs, err := h.proxyUseCase.GetByToken(token)
	if err != nil {
		logger.Warn("AI模型代理Token验证失败",
			zap.String("token", token),
			zap.Error(err),
		)
		response.ErrorCode(c, http.StatusNotFound, "无效的代理Token")
		return
	}

	// 3. 检查是否有绑定的Agent
	if len(agentHostIDs) == 0 {
		response.ErrorCode(c, http.StatusServiceUnavailable, "代理未绑定Agent主机")
		return
	}

	// 4. 选择在线的Agent
	var onlineHostID uint
	var agentStream AgentStreamInterface
	for _, hostID := range agentHostIDs {
		if h.agentHub != nil && h.agentHub.IsOnline(hostID) {
			if as, ok := h.agentHub.GetByHostID(hostID); ok {
				onlineHostID = hostID
				agentStream = as
				break
			}
		}
	}

	if onlineHostID == 0 || agentStream == nil {
		response.ErrorCode(c, http.StatusServiceUnavailable, "Agent主机离线，无法访问")
		return
	}

	// 5. 构建目标URL
	proxyPath := c.Param("path")
	if !strings.HasPrefix(proxyPath, "/") {
		proxyPath = "/" + proxyPath
	}
	targetURL := strings.TrimRight(proxyConfig.TargetURL, "/") + proxyPath
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	logger.Info("AI模型代理转发请求",
		zap.Uint("proxy_id", proxyConfig.ID),
		zap.String("proxy_name", proxyConfig.Name),
		zap.String("target_url", targetURL),
		zap.String("method", c.Request.Method),
	)

	// 6. 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("读取请求体失败", zap.Error(err))
		response.ErrorCode(c, http.StatusBadRequest, "读取请求体失败")
		return
	}

	// 7. 构建请求头
	headers := h.buildForwardHeaders(c, proxyConfig)

	// 8. 生成请求ID
	requestID := uuid.New().String()

	// 9. 构建流式代理请求
	streamReq := &pb.StreamProxyRequest{
		RequestId: requestID,
		Method:    c.Request.Method,
		Url:       targetURL,
		Headers:   headers,
		Body:      body,
		Timeout:   int32(proxyConfig.Timeout),
	}

	// 10. 发送请求到Agent
	if err := agentStream.Send(&pb.ServerMessage{
		Payload: &pb.ServerMessage_StreamProxyRequest{
			StreamProxyRequest: streamReq,
		},
	}); err != nil {
		logger.Error("发送请求到Agent失败", zap.Error(err))
		response.ErrorCode(c, http.StatusBadGateway, "发送请求失败")
		return
	}

	// 11. 获取chunk通道
	timeout := time.Duration(proxyConfig.Timeout) * time.Second
	chunkChan, err := h.agentHub.StreamResponse(agentStream, requestID, timeout)
	if err != nil {
		logger.Error("获取流式响应失败", zap.Error(err))
		response.ErrorCode(c, http.StatusInternalServerError, "获取流式响应失败")
		return
	}

	// 12. 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 禁用nginx缓冲
	c.Header("Transfer-Encoding", "chunked")

	// 13. 获取Flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		logger.Error("不支持流式响应")
		response.ErrorCode(c, http.StatusInternalServerError, "不支持流式响应")
		return
	}

	// 14. 流式转发到客户端
	firstChunk := true
	totalBytes := 0

	// 监听客户端断开
	clientGone := c.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			// 客户端断开连接
			logger.Warn("客户端断开连接",
				zap.Uint("proxy_id", proxyConfig.ID),
				zap.Int("total_bytes", totalBytes),
			)
			return

		case chunkInterface, ok := <-chunkChan:
			if !ok {
				// channel已关闭
				logger.Info("AI模型代理请求完成（channel关闭）",
					zap.Uint("proxy_id", proxyConfig.ID),
					zap.Int("total_bytes", totalBytes),
				)
				return
			}

			chunk, ok := chunkInterface.(*pb.StreamProxyChunk)
			if !ok {
				logger.Error("chunk类型错误")
				return
			}

			// 首块：设置状态码和响应头
			if firstChunk {
				if chunk.StatusCode > 0 {
					c.Status(int(chunk.StatusCode))
				}
				for key, value := range chunk.Headers {
					// 跳过某些响应头，避免冲突
					lowerKey := strings.ToLower(key)
					if lowerKey == "content-length" || lowerKey == "transfer-encoding" {
						continue
					}
					c.Header(key, value)
				}
				firstChunk = false
			}

			// 检查错误
			if chunk.Error != "" {
				logger.Error("Agent返回错误", zap.String("error", chunk.Error))
				// 发送错误信息到客户端
				if len(chunk.Data) == 0 {
					errorMsg := []byte(fmt.Sprintf("data: {\"error\": \"%s\"}\n\n", chunk.Error))
					c.Writer.Write(errorMsg)
					flusher.Flush()
				}
				return
			}

			// 写入数据
			if len(chunk.Data) > 0 {
				n, err := c.Writer.Write(chunk.Data)
				if err != nil {
					logger.Error("写入响应失败", zap.Error(err))
					return
				}
				totalBytes += n

				// 立即刷新到客户端
				flusher.Flush()
			}

			// 检查是否结束
			if chunk.IsFinal {
				logger.Info("AI模型代理请求完成",
					zap.Uint("proxy_id", proxyConfig.ID),
					zap.Int("total_bytes", totalBytes),
				)
				return
			}
		}
	}
}

// buildForwardHeaders 构建转发请求头
func (h *AIModelProxyHandler) buildForwardHeaders(c *gin.Context, proxyConfig *asset.AIModelProxy) map[string]string {
	headers := make(map[string]string)

	// 复制原始请求头
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			// 跳过某些不应该转发的头
			lowerKey := strings.ToLower(key)
			if lowerKey == "host" || lowerKey == "connection" || lowerKey == "upgrade" {
				continue
			}
			headers[key] = values[0]
		}
	}

	// 添加API密钥（如果配置了）
	if proxyConfig.APIKey != "" {
		// 根据模型类型设置不同的认证头
		switch proxyConfig.ModelType {
		case "openai":
			headers["Authorization"] = "Bearer " + proxyConfig.APIKey
		case "ollama":
			// Ollama通常不需要API密钥，但如果配置了就添加
			if proxyConfig.APIKey != "" {
				headers["Authorization"] = "Bearer " + proxyConfig.APIKey
			}
		default:
			// 自定义类型，使用通用的Authorization头
			headers["Authorization"] = "Bearer " + proxyConfig.APIKey
		}
	}

	// 设置Content-Type（如果原请求没有）
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/json"
	}

	// 添加User-Agent
	if _, ok := headers["User-Agent"]; !ok {
		headers["User-Agent"] = "OpsHub-AI-Proxy/1.0"
	}

	return headers
}
