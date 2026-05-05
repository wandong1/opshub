package alert

import (
	"context"
	"encoding/json"
	"fmt"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	alertdata "github.com/ydcloud-dy/opshub/internal/data/alert"
	"gorm.io/gorm"
)

// InhibitService 抑制服务
type InhibitService struct {
	db          *gorm.DB
	inhibitRepo *alertdata.InhibitRuleRepo
	eventRepo   *alertdata.EventRepo
}

// NewInhibitService 创建抑制服务
func NewInhibitService(db *gorm.DB) *InhibitService {
	return &InhibitService{
		db:          db,
		inhibitRepo: alertdata.NewInhibitRuleRepo(db),
		eventRepo:   alertdata.NewEventRepo(db),
	}
}

// ShouldInhibit 检查是否应该抑制
func (s *InhibitService) ShouldInhibit(ctx context.Context, event *biz.AlertEvent, subscriptionID uint) (bool, string, error) {
	// 1. 查询该订阅的抑制规则
	rules, err := s.inhibitRepo.ListBySubscription(ctx, subscriptionID)
	if err != nil || len(rules) == 0 {
		return false, "", nil
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// 2. 检查当前告警是否匹配目标条件
		if !s.matchTarget(event, rule.TargetMatchers) {
			continue
		}

		// 3. 查询是否存在匹配的源告警（firing状态）
		sourceEvents, err := s.eventRepo.ListFiringByAssetGroup(ctx, event.AssetGroupID)
		if err != nil {
			continue
		}

		for _, sourceEvent := range sourceEvents {
			// 检查源告警是否匹配
			if !s.matchSource(sourceEvent, rule.SourceMatchers) {
				continue
			}

			// 检查 equal_labels 条件
			if s.checkEqualLabels(event, sourceEvent, rule.EqualLabels) {
				reason := fmt.Sprintf("被源告警抑制: %s (ID:%d)", sourceEvent.RuleName, sourceEvent.ID)
				return true, reason, nil
			}
		}
	}

	return false, "", nil
}

// matchTarget 检查告警是否匹配目标条件
func (s *InhibitService) matchTarget(event *biz.AlertEvent, matchersJSON string) bool {
	var matchers map[string]string
	if err := json.Unmarshal([]byte(matchersJSON), &matchers); err != nil {
		return false
	}

	eventLabels := make(map[string]string)
	json.Unmarshal([]byte(event.Labels), &eventLabels)

	for key, value := range matchers {
		switch key {
		case "severity":
			if event.Severity != value {
				return false
			}
		case "ruleName":
			if event.RuleName != value {
				return false
			}
		default:
			if eventLabels[key] != value {
				return false
			}
		}
	}

	return true
}

// matchSource 检查告警是否匹配源条件
func (s *InhibitService) matchSource(event *biz.AlertEvent, matchersJSON string) bool {
	return s.matchTarget(event, matchersJSON)
}

// checkEqualLabels 检查两个告警的指定标签是否相等
func (s *InhibitService) checkEqualLabels(event1, event2 *biz.AlertEvent, equalLabelsJSON string) bool {
	var equalLabels []string
	if err := json.Unmarshal([]byte(equalLabelsJSON), &equalLabels); err != nil || len(equalLabels) == 0 {
		return true // 无相等标签要求，直接匹配
	}

	labels1 := make(map[string]string)
	labels2 := make(map[string]string)
	json.Unmarshal([]byte(event1.Labels), &labels1)
	json.Unmarshal([]byte(event2.Labels), &labels2)

	for _, label := range equalLabels {
		if labels1[label] != labels2[label] {
			return false
		}
	}

	return true
}
