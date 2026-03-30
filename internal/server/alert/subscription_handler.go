package alert

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// subscriptionRequest 创建/更新订阅的请求体
type subscriptionRequest struct {
	biz.AlertSubscription
	Rules      []subscriptionRuleReq `json:"rules"`
	ChannelIDs []uint                `json:"channelIds"`
	UserIDs    []uint                `json:"userIds"`
}

type subscriptionRuleReq struct {
	RuleID     uint            `json:"ruleId"`
	TimeRanges json.RawMessage `json:"timeRanges"` // JSON array of TimeRange
	Severities json.RawMessage `json:"severities"` // JSON array of severity strings
	ChannelIDs json.RawMessage `json:"channelIds"` // JSON array of channel IDs
	UserIDs    json.RawMessage `json:"userIds"`    // JSON array of user IDs
}

// subscriptionVO 订阅任务详情视图
type subscriptionVO struct {
	biz.AlertSubscription
	Rules      []*biz.AlertSubscriptionRule    `json:"rules"`
	Channels   []*biz.AlertSubscriptionChannel `json:"channels"`
	Users      []*biz.AlertSubscriptionUser    `json:"users"`
	RuleCount  int                             `json:"ruleCount"`
	ChannelCount int                           `json:"channelCount"`
}

func (s *HTTPServer) listSubscriptions(c *gin.Context) {
	assetGroupID, _ := strconv.ParseUint(c.Query("assetGroupId"), 10, 64)
	list, err := s.subRepo.List(c.Request.Context(), uint(assetGroupID))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}
	// 附加规则数和通道数
	var vos []subscriptionVO
	for _, sub := range list {
		rules, _ := s.subRuleRepo.ListBySubscription(c.Request.Context(), sub.ID)
		channels, _ := s.subChannelRepo.ListBySubscription(c.Request.Context(), sub.ID)
		vos = append(vos, subscriptionVO{
			AlertSubscription: *sub,
			RuleCount:         len(rules),
			ChannelCount:      len(channels),
		})
	}
	response.Success(c, vos)
}

func (s *HTTPServer) createSubscription(c *gin.Context) {
	var req subscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.subRepo.Create(c.Request.Context(), &req.AlertSubscription); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败")
		return
	}
	s.saveSubRelations(c, req.AlertSubscription.ID, req.Rules, req.ChannelIDs, req.UserIDs)
	response.Success(c, req.AlertSubscription)
}

func (s *HTTPServer) getSubscription(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	sub, err := s.subRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "订阅任务不存在")
		return
	}
	rules, _ := s.subRuleRepo.ListBySubscription(c.Request.Context(), uint(id))
	channels, _ := s.subChannelRepo.ListBySubscription(c.Request.Context(), uint(id))
	users, _ := s.subUserRepo.ListBySubscription(c.Request.Context(), uint(id))
	vo := subscriptionVO{
		AlertSubscription: *sub,
		Rules:             rules,
		Channels:          channels,
		Users:             users,
		RuleCount:         len(rules),
		ChannelCount:      len(channels),
	}
	response.Success(c, vo)
}

func (s *HTTPServer) updateSubscription(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req subscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	req.AlertSubscription.ID = uint(id)
	if err := s.subRepo.Update(c.Request.Context(), &req.AlertSubscription); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败")
		return
	}
	s.saveSubRelations(c, uint(id), req.Rules, req.ChannelIDs, req.UserIDs)
	response.Success(c, req.AlertSubscription)
}

func (s *HTTPServer) deleteSubscription(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.subRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}
	response.Success(c, nil)
}

func rawJSONStr(raw json.RawMessage, fallback string) string {
	if len(raw) == 0 {
		return fallback
	}
	var check interface{}
	if json.Unmarshal(raw, &check) == nil {
		return string(raw)
	}
	return fallback
}

func (s *HTTPServer) saveSubRelations(c *gin.Context, subID uint, rules []subscriptionRuleReq, channelIDs, userIDs []uint) {
	var subRules []*biz.AlertSubscriptionRule
	for _, r := range rules {
		subRules = append(subRules, &biz.AlertSubscriptionRule{
			SubscriptionID: subID,
			RuleID:         r.RuleID,
			TimeRanges:     rawJSONStr(r.TimeRanges, "[]"),
			Severities:     rawJSONStr(r.Severities, "[]"),
			ChannelIDs:     rawJSONStr(r.ChannelIDs, "[]"),
			UserIDs:        rawJSONStr(r.UserIDs, "[]"),
		})
	}
	_ = s.subRuleRepo.SetRules(c.Request.Context(), subID, subRules)
	// 保留全局 channelIds/userIds 向下兼容（当规则行未配置时使用）
	_ = s.subChannelRepo.SetChannels(c.Request.Context(), subID, channelIDs)
	_ = s.subUserRepo.SetUsers(c.Request.Context(), subID, userIDs)
}
