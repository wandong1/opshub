package alert

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	alertsvc "github.com/ydcloud-dy/opshub/internal/service/alert"
	"github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"go.uber.org/zap"
)

func (s *HTTPServer) listDataSources(c *gin.Context) {
	list, err := s.dsRepo.List(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}
	response.Success(c, list)
}

func (s *HTTPServer) createDataSource(c *gin.Context) {
	var req biz.CreateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}

	// 业务验证：URL 必填
	if req.URL == "" {
		response.ErrorCode(c, http.StatusBadRequest, "请输入数据源地址（URL）")
		return
	}

	// 构建数据源对象
	ds := &biz.AlertDataSource{
		Name:        req.Name,
		Type:        req.Type,
		AccessMode:  req.AccessMode,
		URL:         req.URL, // 两种模式都使用 URL
		Username:    req.Username,
		Password:    req.Password,
		Token:       req.Token,
		Description: req.Description,
		Status:      req.Status,
	}

	// 只有 Agent 模式才生成代理信息
	if req.AccessMode == "agent" {
		ds.ProxyToken = uuid.New().String()                             // 生成 Token
		ds.ProxyURL = "/api/v1/alert/proxy/datasource/" + ds.ProxyToken // 生成 URL
		ds.ProxyEnabled = true
	}

	if err := s.dsRepo.Create(c.Request.Context(), ds); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}
	response.Success(c, ds)
}

func (s *HTTPServer) getDataSource(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	ds, err := s.dsRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "数据源不存在")
		return
	}
	response.Success(c, ds)
}

func (s *HTTPServer) updateDataSource(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req biz.UpdateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}

	// 获取原有数据源
	existingDS, err := s.dsRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "数据源不存在")
		return
	}

	// 只允许更新指定字段（AccessMode 不可修改）
	if req.Name != "" {
		existingDS.Name = req.Name
	}
	if req.Type != "" {
		existingDS.Type = req.Type
	}
	if req.URL != "" {
		existingDS.URL = req.URL
	}

	// 两种模式都可更新的字段
	if req.Username != "" {
		existingDS.Username = req.Username
	}
	if req.Password != "" {
		existingDS.Password = req.Password
	}
	if req.Token != "" {
		existingDS.Token = req.Token
	}
	if req.Description != "" {
		existingDS.Description = req.Description
	}
	if req.Status > 0 {
		existingDS.Status = req.Status
	}

	if err := s.dsRepo.Update(c.Request.Context(), existingDS); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}
	response.Success(c, existingDS)
}

func (s *HTTPServer) deleteDataSource(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.dsRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}
	response.Success(c, nil)
}

func (s *HTTPServer) testDataSource(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	ds, err := s.dsRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "数据源不存在")
		return
	}

	// Agent代理模式：需要通过关联的Agent转发测试
	if ds.AccessMode == "agent" {
		// 获取关联的Agent
		relations, err := s.dsAgentRelationRepo.ListByDataSourceID(c.Request.Context(), ds.ID)
		if err != nil || len(relations) == 0 {
			response.ErrorCode(c, http.StatusBadRequest, "该数据源未关联任何Agent")
			return
		}

		// 获取首个在线的Agent（这里应该按优先级排序，但暂时取第一个）
		var targetAgent *biz.DataSourceAgentRelation
		for _, rel := range relations {
			// TODO: 检查Agent是否在线
			targetAgent = rel
			break
		}

		if targetAgent == nil {
			response.ErrorCode(c, http.StatusServiceUnavailable, "没有可用的Agent")
			return
		}

		// Agent代理模式：通过代理转发 URL 进行测试
		// 这样才能真正通过 Agent 转发，而不是直接访问

		// 构建测试路径
		var testPath string
		switch ds.Type {
		case "prometheus", "victoriametrics":
			testPath = "/api/v1/query?query=up"
		case "influxdb":
			testPath = "/query?q=SHOW+DATABASES"
		default:
			testPath = "/"
		}

		// 使用代理转发 URL（通过 ProxyToken）
		// 格式：/api/v1/alert/proxy/datasource/{token}{testPath}
		proxyURL := ds.ProxyURL + testPath

		// 构建完整的测试 URL（本地服务器地址）
		// 这样请求会经过代理处理器，再通过 Agent 转发到实际数据源
		baseURL := "http://localhost:9876" // 或从配置读取
		testURL := baseURL + proxyURL
		logger.Info("数据源测试转发地址 %s", zap.String("test_url", testURL))
		// 构建测试请求
		req, err := http.NewRequest("GET", testURL, nil)
		if err != nil {
			response.ErrorCode(c, http.StatusBadRequest, "构建请求失败: "+err.Error())
			return
		}

		// 执行请求（会经过代理处理器转发）
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			response.ErrorCode(c, http.StatusBadGateway, "代理转发失败: "+err.Error())
			return
		}
		defer resp.Body.Close()

		// 检查响应状态
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			response.ErrorCode(c, http.StatusBadGateway, fmt.Sprintf("数据源返回错误: %d %s", resp.StatusCode, string(body)))
			return
		}

		response.Success(c, gin.H{
			"message":     "Agent代理转发测试成功",
			"proxy_url":   proxyURL,
			"status_code": resp.StatusCode,
		})
		return
	}

	// 直连模式：直接测试
	if err := alertsvc.TestDataSource(ds); err != nil {
		response.ErrorCode(c, http.StatusBadGateway, "连接失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "连接成功"})
}

// 数据源Agent关联管理

func (s *HTTPServer) listAgentRelations(c *gin.Context) {
	// 从URL路由参数中获取dataSourceId（路由格式：/:id/agent-relations）
	dsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "缺少或格式错误的datasource_id参数")
		return
	}

	rels, err := s.dsAgentRelationRepo.ListByDataSourceID(c.Request.Context(), uint(dsID))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}
	response.Success(c, rels)
}

func (s *HTTPServer) createAgentRelation(c *gin.Context) {
	// 从URL路由参数中获取dataSourceId
	dsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "缺少或格式错误的datasource_id参数")
		return
	}

	var req biz.DataSourceAgentRelation
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}

	// 确保dataSourceId与URL参数一致
	req.DataSourceID = uint(dsID)

	if err := s.dsAgentRelationRepo.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败")
		return
	}
	response.Success(c, req)
}

func (s *HTTPServer) deleteAgentRelation(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.dsAgentRelationRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}
	response.Success(c, nil)
}
