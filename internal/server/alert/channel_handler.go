package alert

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

func (s *HTTPServer) listChannels(c *gin.Context) {
	list, err := s.channelRepo.List(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}
	response.Success(c, list)
}

func (s *HTTPServer) createChannel(c *gin.Context) {
	var req biz.AlertNotifyChannel
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.AlertTemplate == "" {
		req.AlertTemplate = defaultAlertTemplate(req.Type)
	}
	if req.ResolveTemplate == "" {
		req.ResolveTemplate = defaultResolveTemplate(req.Type)
	}
	if err := s.channelRepo.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败")
		return
	}
	response.Success(c, req)
}

func (s *HTTPServer) getChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	ch, err := s.channelRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "通道不存在")
		return
	}
	response.Success(c, ch)
}

func (s *HTTPServer) updateChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req biz.AlertNotifyChannel
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	req.ID = uint(id)
	if err := s.channelRepo.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败")
		return
	}
	response.Success(c, req)
}

func (s *HTTPServer) deleteChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.channelRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}
	response.Success(c, nil)
}

func (s *HTTPServer) testChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	ch, err := s.channelRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "通道不存在")
		return
	}
	testEvent := &biz.AlertEvent{
		RuleName: "测试告警规则",
		Severity: "warning",
		Value:    99.9,
		Labels:   `{"env":"test"}`,
	}
	s.notifySvc.Send(c.Request.Context(), ch, testEvent, false, []string{})
	response.Success(c, gin.H{"message": "测试通知已发送"})
}

func defaultAlertTemplate(channelType string) string {
	switch channelType {
	case "wechat_work":
		return "## 🔴 SreHub 告警通知\n> **规则**: <font color=\"warning\">{{.RuleName}}</font>\n> **级别**: {{.Severity}}\n> **当前值**: {{.Value}}\n> **触发时间**: {{.FiredAt}}\n> **标签**: {{.Labels}}"
	case "dingtalk":
		return "## 🔴 SreHub 告警通知\n- **规则**: {{.RuleName}}\n- **级别**: {{.Severity}}\n- **当前值**: {{.Value}}\n- **触发时间**: {{.FiredAt}}"
	default:
		return "【OpsHub告警】规则: {{.RuleName}} | 级别: {{.Severity}} | 值: {{.Value}} | 时间: {{.FiredAt}}"
	}
}

func defaultResolveTemplate(channelType string) string {
	return "## ✅ 告警已恢复\n**规则**: {{.RuleName}}\n**级别**: {{.Severity}}\n**触发值**: {{.Value}}\n**恢复时间**: {{.ResolvedAt}}\n**触发时间**: {{.FiredAt}}"
}
