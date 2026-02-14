package inspection

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	biz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	"github.com/ydcloud-dy/opshub/pkg/metrics"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// PushgatewayService handles HTTP requests for Pushgateway configs.
type PushgatewayService struct {
	useCase *biz.PushgatewayUseCase
}

func NewPushgatewayService(uc *biz.PushgatewayUseCase) *PushgatewayService {
	return &PushgatewayService{useCase: uc}
}

func (s *PushgatewayService) Create(c *gin.Context) {
	var req biz.PushgatewayConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	model := req.ToModel()
	if err := s.useCase.Create(c.Request.Context(), model); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}
	response.Success(c, model)
}

func (s *PushgatewayService) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req biz.PushgatewayConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	model := req.ToModel()
	model.ID = uint(id)
	if err := s.useCase.Update(c.Request.Context(), model); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}
	response.Success(c, model)
}

func (s *PushgatewayService) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.useCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}
	response.Success(c, nil)
}

func (s *PushgatewayService) List(c *gin.Context) {
	configs, err := s.useCase.List(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}
	response.Success(c, configs)
}

func (s *PushgatewayService) Test(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	config, err := s.useCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "未找到: "+err.Error())
		return
	}

	pusher := metrics.NewPusher(config.URL, config.Username, config.Password)
	if err := pusher.TestConnection(); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "连接失败: "+err.Error())
		return
	}
	response.SuccessWithMessage(c, "连接成功", nil)
}
