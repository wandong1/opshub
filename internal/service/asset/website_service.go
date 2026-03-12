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

type WebsiteService struct {
	websiteUseCase *asset.WebsiteUseCase
}

func NewWebsiteService(websiteUseCase *asset.WebsiteUseCase) *WebsiteService {
	return &WebsiteService{
		websiteUseCase: websiteUseCase,
	}
}

// ListWebsites 获取站点列表
func (s *WebsiteService) ListWebsites(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")
	siteType := c.Query("type")

	// 分组ID列表
	var groupIDs []uint
	if groupIDsStr := c.Query("groupIds"); groupIDsStr != "" {
		if err := c.ShouldBindJSON(&groupIDs); err == nil {
			// 从query参数解析
		}
	}

	websites, total, err := s.websiteUseCase.List(c.Request.Context(), page, pageSize, keyword, groupIDs, siteType)
	if err != nil {
		response.ErrorCode(c, 500, "获取站点列表失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     websites,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetWebsite 获取站点详情
func (s *WebsiteService) GetWebsite(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ErrorCode(c, 400, "无效的站点ID")
		return
	}

	website, err := s.websiteUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, 500, "获取站点详情失败: "+err.Error())
		return
	}

	response.Success(c, website)
}

// CreateWebsite 创建站点
func (s *WebsiteService) CreateWebsite(c *gin.Context) {
	var req asset.WebsiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, 400, "参数错误: "+err.Error())
		return
	}

	if err := s.websiteUseCase.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, 500, "创建站点失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// UpdateWebsite 更新站点
func (s *WebsiteService) UpdateWebsite(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ErrorCode(c, 400, "无效的站点ID")
		return
	}

	var req asset.WebsiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, 400, "参数错误: "+err.Error())
		return
	}

	if err := s.websiteUseCase.Update(c.Request.Context(), uint(id), &req); err != nil {
		response.ErrorCode(c, 500, "更新站点失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// DeleteWebsite 删除站点
func (s *WebsiteService) DeleteWebsite(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ErrorCode(c, 400, "无效的站点ID")
		return
	}

	if err := s.websiteUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, 500, "删除站点失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}
