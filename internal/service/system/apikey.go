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

package system

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/system"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// APIKeyService API Key 服务
type APIKeyService struct {
	apiKeyUseCase *system.APIKeyUseCase
}

// NewAPIKeyService 创建 API Key 服务
func NewAPIKeyService(apiKeyUseCase *system.APIKeyUseCase) *APIKeyService {
	return &APIKeyService{
		apiKeyUseCase: apiKeyUseCase,
	}
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// CreateAPIKey 创建 API Key
// @Summary 创建 API Key
// @Description 生成新的 API Key，仅此接口返回完整明文密钥
// @Tags API Key 管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body CreateAPIKeyRequest true "API Key 信息"
// @Success 200 {object} response.Response{data=system.CreateAPIKeyResponse} "创建成功"
// @Router /api/v1/system/apikeys [post]
func (s *APIKeyService) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	result, err := s.apiKeyUseCase.CreateAPIKey(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建 API Key 失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// ListAPIKeys 获取 API Key 列表
// @Summary 获取 API Key 列表
// @Description 获取 API Key 列表，密钥已脱敏，包含调用统计信息
// @Tags API Key 管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response{data=[]system.APIKeyVO} "获取成功"
// @Router /api/v1/system/apikeys [get]
func (s *APIKeyService) ListAPIKeys(c *gin.Context) {
	page := 1
	pageSize := 20

	if p, ok := c.GetQuery("page"); ok {
		if pInt, err := parseIntParam(p); err == nil && pInt > 0 {
			page = pInt
		}
	}

	if ps, ok := c.GetQuery("page_size"); ok {
		if psInt, err := parseIntParam(ps); err == nil && psInt > 0 && psInt <= 100 {
			pageSize = psInt
		}
	}

	vos, total, err := s.apiKeyUseCase.ListAPIKeys(c.Request.Context(), page, pageSize)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取列表失败: "+err.Error())
		return
	}

	response.Pagination(c, total, page, pageSize, vos)
}

// DeleteAPIKey 删除 API Key
// @Summary 删除 API Key
// @Description 删除 API Key，删除后立即全局失效
// @Tags API Key 管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "API Key ID"
// @Success 200 {object} response.Response "删除成功"
// @Router /api/v1/system/apikeys/:id [delete]
func (s *APIKeyService) DeleteAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := parseIntParam(idStr)
	if err != nil || id <= 0 {
		response.ErrorCode(c, http.StatusBadRequest, "无效的 ID")
		return
	}

	if err := s.apiKeyUseCase.DeleteAPIKey(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// parseIntParam 解析整数参数
func parseIntParam(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// GetAPIKeyUseCase 获取 API Key 用例（供认证中间件使用）
func (s *APIKeyService) GetAPIKeyUseCase() *system.APIKeyUseCase {
	return s.apiKeyUseCase
}
