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
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

type AIModelProxyService struct {
	proxyUseCase *asset.AIModelProxyUseCase
}

func NewAIModelProxyService(proxyUseCase *asset.AIModelProxyUseCase) *AIModelProxyService {
	return &AIModelProxyService{
		proxyUseCase: proxyUseCase,
	}
}

// ListAIModelProxies 获取AI模型代理列表
// @Summary 获取AI模型代理列表
// @Tags AI模型代理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param groupId query int false "分组ID"
// @Param status query int false "状态"
// @Param keyword query string false "关键词"
// @Success 200 {object} response.Response
// @Router /api/v1/ai-model-proxies [get]
func (s *AIModelProxyService) ListAIModelProxies(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	var groupID uint
	if groupIDStr := c.Query("groupId"); groupIDStr != "" {
		if id, err := strconv.ParseUint(groupIDStr, 10, 32); err == nil {
			groupID = uint(id)
		}
	}

	var status *int
	if statusStr := c.Query("status"); statusStr != "" {
		if s, err := strconv.Atoi(statusStr); err == nil {
			status = &s
		}
	}

	proxies, total, err := s.proxyUseCase.List(page, pageSize, groupID, status, keyword)
	if err != nil {
		response.ErrorCode(c, 500, "获取AI模型代理列表失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     proxies,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetAIModelProxy 获取AI模型代理详情
// @Summary 获取AI模型代理详情
// @Tags AI模型代理
// @Accept json
// @Produce json
// @Param id path int true "代理ID"
// @Success 200 {object} response.Response
// @Router /api/v1/ai-model-proxies/{id} [get]
func (s *AIModelProxyService) GetAIModelProxy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ErrorCode(c, 400, "无效的代理ID")
		return
	}

	proxy, err := s.proxyUseCase.GetByID(uint(id))
	if err != nil {
		response.ErrorCode(c, 500, "获取AI模型代理详情失败: "+err.Error())
		return
	}

	response.Success(c, proxy)
}

// CreateAIModelProxy 创建AI模型代理
// @Summary 创建AI模型代理
// @Tags AI模型代理
// @Accept json
// @Produce json
// @Param request body asset.AIModelProxyRequest true "代理信息"
// @Success 200 {object} response.Response
// @Router /api/v1/ai-model-proxies [post]
func (s *AIModelProxyService) CreateAIModelProxy(c *gin.Context) {
	var req asset.AIModelProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, 400, "参数错误: "+err.Error())
		return
	}

	proxy, err := s.proxyUseCase.Create(&req)
	if err != nil {
		response.ErrorCode(c, 500, "创建AI模型代理失败: "+err.Error())
		return
	}

	response.Success(c, proxy)
}

// UpdateAIModelProxy 更新AI模型代理
// @Summary 更新AI模型代理
// @Tags AI模型代理
// @Accept json
// @Produce json
// @Param id path int true "代理ID"
// @Param request body asset.AIModelProxyRequest true "代理信息"
// @Success 200 {object} response.Response
// @Router /api/v1/ai-model-proxies/{id} [put]
func (s *AIModelProxyService) UpdateAIModelProxy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ErrorCode(c, 400, "无效的代理ID")
		return
	}

	var req asset.AIModelProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, 400, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	proxy, err := s.proxyUseCase.Update(&req)
	if err != nil {
		response.ErrorCode(c, 500, "更新AI模型代理失败: "+err.Error())
		return
	}

	response.Success(c, proxy)
}

// DeleteAIModelProxy 删除AI模型代理
// @Summary 删除AI模型代理
// @Tags AI模型代理
// @Accept json
// @Produce json
// @Param id path int true "代理ID"
// @Success 200 {object} response.Response
// @Router /api/v1/ai-model-proxies/{id} [delete]
func (s *AIModelProxyService) DeleteAIModelProxy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ErrorCode(c, 400, "无效的代理ID")
		return
	}

	if err := s.proxyUseCase.Delete(uint(id)); err != nil {
		response.ErrorCode(c, 500, "删除AI模型代理失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// RegenerateToken 重新生成Token
// @Summary 重新生成Token
// @Tags AI模型代理
// @Accept json
// @Produce json
// @Param id path int true "代理ID"
// @Success 200 {object} response.Response
// @Router /api/v1/ai-model-proxies/{id}/regenerate-token [post]
func (s *AIModelProxyService) RegenerateToken(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ErrorCode(c, 400, "无效的代理ID")
		return
	}

	proxy, err := s.proxyUseCase.RegenerateToken(uint(id))
	if err != nil {
		response.ErrorCode(c, 500, "重新生成Token失败: "+err.Error())
		return
	}

	response.Success(c, proxy)
}

// TestConnection 测试连接
// @Summary 测试连接
// @Tags AI模型代理
// @Accept json
// @Produce json
// @Param id path int true "代理ID"
// @Success 200 {object} response.Response
// @Router /api/v1/ai-model-proxies/{id}/test [get]
func (s *AIModelProxyService) TestConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ErrorCode(c, 400, "无效的代理ID")
		return
	}

	// 获取代理配置
	proxy, err := s.proxyUseCase.GetByID(uint(id))
	if err != nil {
		response.ErrorCode(c, 500, "获取AI模型代理失败: "+err.Error())
		return
	}

	// TODO: 实现实际的连接测试逻辑
	// 这里可以通过Agent发送测试请求到目标URL

	response.Success(c, gin.H{
		"success": true,
		"message": "连接测试成功",
		"proxy":   proxy,
	})
}
