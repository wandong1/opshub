package alert

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	alertsvc "github.com/ydcloud-dy/opshub/internal/service/alert"
	"github.com/ydcloud-dy/opshub/pkg/response"
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
	var req biz.AlertDataSource
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.dsRepo.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败")
		return
	}
	response.Success(c, req)
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
	var req biz.AlertDataSource
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	req.ID = uint(id)
	if err := s.dsRepo.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败")
		return
	}
	response.Success(c, req)
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
	if err := alertsvc.TestDataSource(ds); err != nil {
		response.ErrorCode(c, http.StatusBadGateway, "连接失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "连接成功"})
}
