package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	alertdata "github.com/ydcloud-dy/opshub/internal/data/alert"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GroupService 分组服务
type GroupService struct {
	db         *gorm.DB
	groupRepo  *alertdata.GroupRuleRepo
	cacheRepo  *alertdata.GroupCacheRepo
	eventRepo  *alertdata.EventRepo
	notifySvc  *NotifyService
	subRepo    *alertdata.SubscriptionRepo
	subRuleRepo *alertdata.SubscriptionRuleRepo
	subChannelRepo *alertdata.SubscriptionChannelRepo
	subUserRepo *alertdata.SubscriptionUserRepo
	channelRepo *alertdata.ChannelRepo
}

// NewGroupService 创建分组服务
func NewGroupService(db *gorm.DB, notifySvc *NotifyService) *GroupService {
	return &GroupService{
		db:         db,
		groupRepo:  alertdata.NewGroupRuleRepo(db),
		cacheRepo:  alertdata.NewGroupCacheRepo(db),
		eventRepo:  alertdata.NewEventRepo(db),
		notifySvc:  notifySvc,
		subRepo:    alertdata.NewSubscriptionRepo(db),
		subRuleRepo: alertdata.NewSubscriptionRuleRepo(db),
		subChannelRepo: alertdata.NewSubscriptionChannelRepo(db),
		subUserRepo: alertdata.NewSubscriptionUserRepo(db),
		channelRepo: alertdata.NewChannelRepo(db),
	}
}

// AddToGroup 添加到分组
func (s *GroupService) AddToGroup(ctx context.Context, event *biz.AlertEvent, subscriptionID uint) (bool, error) {
	// 1. 查询该订阅的分组规则
	rules, err := s.groupRepo.ListBySubscription(ctx, subscriptionID)
	if err != nil || len(rules) == 0 {
		return false, nil // 无规则，不分组
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// 2. 生成分组键
		groupKey := s.generateGroupKey(event, rule.GroupBy)

		// 3. 查找或创建分组缓存
		cache, err := s.cacheRepo.GetActiveCache(ctx, rule.ID, groupKey)
		now := time.Now()

		if err != nil {
			// 创建新分组
			alerts := []uint{event.ID}
			alertsJSON, _ := json.Marshal(alerts)

			cache = &biz.AlertGroupCache{
				GroupRuleID:    rule.ID,
				SubscriptionID: subscriptionID,
				GroupKey:       groupKey,
				Alerts:         string(alertsJSON),
				FirstAlertAt:   now,
				LastAlertAt:    now,
				AlertCount:     1,
			}
			s.cacheRepo.Create(ctx, cache)

			// 启动定时器，等待 group_wait 后发送
			time.AfterFunc(time.Duration(rule.GroupWait)*time.Second, func() {
				s.sendGroupCache(context.Background(), cache.ID)
			})

			return true, nil
		}

		// 4. 添加到现有分组
		var alerts []uint
		json.Unmarshal([]byte(cache.Alerts), &alerts)
		alerts = append(alerts, event.ID)

		// 检查是否达到最大分组大小
		if len(alerts) >= rule.MaxGroupSize {
			// 立即发送
			s.sendGroupCache(ctx, cache.ID)
			return true, nil
		}

		alertsJSON, _ := json.Marshal(alerts)
		s.cacheRepo.UpdateAlerts(ctx, cache.ID, string(alertsJSON), now, len(alerts))

		return true, nil
	}

	return false, nil
}

// generateGroupKey 生成分组键
func (s *GroupService) generateGroupKey(event *biz.AlertEvent, groupByJSON string) string {
	var keys []string
	if err := json.Unmarshal([]byte(groupByJSON), &keys); err != nil {
		return "default"
	}

	parts := []string{}
	labels := make(map[string]string)
	json.Unmarshal([]byte(event.Labels), &labels)

	for _, key := range keys {
		switch key {
		case "severity":
			parts = append(parts, fmt.Sprintf("severity=%s", event.Severity))
		case "ruleName":
			parts = append(parts, fmt.Sprintf("ruleName=%s", event.RuleName))
		default:
			if val, ok := labels[key]; ok {
				parts = append(parts, fmt.Sprintf("%s=%s", key, val))
			}
		}
	}

	if len(parts) == 0 {
		return "default"
	}

	return strings.Join(parts, ",")
}

// sendGroupCache 发送分组缓存
func (s *GroupService) sendGroupCache(ctx context.Context, cacheID uint) {
	cache, err := s.cacheRepo.GetByID(ctx, cacheID)
	if err != nil || cache.Sent {
		return
	}

	// 获取所有告警事件
	var alertIDs []uint
	json.Unmarshal([]byte(cache.Alerts), &alertIDs)

	if len(alertIDs) == 0 {
		return
	}

	// 查询告警事件详情
	events, err := s.eventRepo.ListByIDs(ctx, alertIDs)
	if err != nil || len(events) == 0 {
		appLogger.Warn("查询分组告警事件失败", zap.Uint("cacheID", cacheID), zap.Error(err))
		return
	}

	appLogger.Info("发送分组告警",
		zap.Uint("cacheID", cacheID),
		zap.Int("alertCount", len(events)),
		zap.String("groupKey", cache.GroupKey))

	// 获取订阅规则，确定通道和用户
	subRules, err := s.subRuleRepo.ListBySubscription(ctx, cache.SubscriptionID)
	if err != nil || len(subRules) == 0 {
		appLogger.Warn("未找到订阅规则", zap.Uint("subscriptionID", cache.SubscriptionID))
		s.cacheRepo.MarkSent(ctx, cacheID, time.Now())
		return
	}

	// 使用第一个订阅规则的通道和用户配置
	sr := subRules[0]
	var channelIDs []uint
	if perRuleCh := parseUintList(sr.ChannelIDs); len(perRuleCh) > 0 {
		channelIDs = perRuleCh
	} else {
		subChannels, _ := s.subChannelRepo.ListBySubscription(ctx, cache.SubscriptionID)
		for _, sc := range subChannels {
			channelIDs = append(channelIDs, sc.ChannelID)
		}
	}

	var phones []string
	if perRuleUsers := parseUintList(sr.UserIDs); len(perRuleUsers) > 0 {
		phones = getUserPhones(ctx, s.db, perRuleUsers)
	} else {
		phones = getSubPhones(ctx, s.db, s.subUserRepo, cache.SubscriptionID)
	}

	// 发送每个告警（实际应用中可以构建聚合消息）
	channels, _ := s.channelRepo.ListByIDs(ctx, channelIDs)
	for _, event := range events {
		// 根据事件状态判断是告警还是恢复
		isResolve := event.Status == "resolved"
		for _, ch := range channels {
			if !ch.Enabled {
				continue
			}
			go s.notifySvc.Send(ctx, ch, event, isResolve, phones)
		}
	}

	// 标记已发送
	s.cacheRepo.MarkSent(ctx, cacheID, time.Now())
}

// getUserPhones 获取用户手机号
func getUserPhones(ctx context.Context, db *gorm.DB, userIDs []uint) []string {
	for _, uid := range userIDs {
		if uid == 0 {
			return []string{}
		}
	}
	type phoneRow struct{ Phone string }
	var rows []phoneRow
	db.WithContext(ctx).Table("sys_users").Select("phone").Where("id IN ? AND phone != ''", userIDs).Scan(&rows)
	var phones []string
	for _, r := range rows {
		if r.Phone != "" {
			phones = append(phones, r.Phone)
		}
	}
	return phones
}

// getSubPhones 获取订阅用户手机号
func getSubPhones(ctx context.Context, db *gorm.DB, subUserRepo *alertdata.SubscriptionUserRepo, subscriptionID uint) []string {
	subUsers, err := subUserRepo.ListBySubscription(ctx, subscriptionID)
	if err != nil || len(subUsers) == 0 {
		return []string{}
	}
	for _, su := range subUsers {
		if su.UserID == 0 {
			return []string{}
		}
	}
	var userIDs []uint
	for _, su := range subUsers {
		userIDs = append(userIDs, su.UserID)
	}
	return getUserPhones(ctx, db, userIDs)
}

// SendGroupedAlerts 定时任务：发送超时的分组
func (s *GroupService) SendGroupedAlerts(ctx context.Context) error {
	// 查询所有未发送且超过 group_interval 的分组
	// 这里简化为查询5分钟前的分组
	before := time.Now().Add(-5 * time.Minute)
	caches, err := s.cacheRepo.ListPendingCaches(ctx, before)
	if err != nil {
		return err
	}

	for _, cache := range caches {
		s.sendGroupCache(ctx, cache.ID)
	}

	return nil
}

// CleanupSentCaches 清理已发送的分组缓存（定时任务）
func (s *GroupService) CleanupSentCaches(ctx context.Context) error {
	// 清理7天前已发送的分组缓存
	before := time.Now().AddDate(0, 0, -7)
	return s.cacheRepo.CleanupSent(ctx, before)
}
