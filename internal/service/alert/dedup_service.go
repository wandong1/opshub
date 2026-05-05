package alert

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	alertdata "github.com/ydcloud-dy/opshub/internal/data/alert"
	"gorm.io/gorm"
)

// DedupService 去重服务
type DedupService struct {
	db              *gorm.DB
	dedupRepo       *alertdata.DedupRuleRepo
	fingerprintRepo *alertdata.FingerprintRepo
}

// NewDedupService 创建去重服务
func NewDedupService(db *gorm.DB) *DedupService {
	return &DedupService{
		db:              db,
		dedupRepo:       alertdata.NewDedupRuleRepo(db),
		fingerprintRepo: alertdata.NewFingerprintRepo(db),
	}
}

// ShouldDeduplicate 检查是否应该去重
func (s *DedupService) ShouldDeduplicate(ctx context.Context, event *biz.AlertEvent, subscriptionID uint) (bool, error) {
	// 1. 查询该订阅的去重规则
	rules, err := s.dedupRepo.ListBySubscription(ctx, subscriptionID)
	if err != nil || len(rules) == 0 {
		return false, nil // 无规则，不去重
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// 2. 生成指纹
		fingerprint := s.generateFingerprint(event, rule.FingerprintKeys)

		// 3. 查询指纹记录
		fp, err := s.fingerprintRepo.GetByFingerprint(ctx, subscriptionID, fingerprint)
		if err != nil {
			// 首次出现，不去重
			return false, nil
		}

		// 4. 检查是否在去重窗口内
		now := time.Now()
		if fp.LastSentAt != nil && now.Sub(*fp.LastSentAt).Seconds() < float64(rule.DedupWindow) {
			// 在窗口内，去重
			s.fingerprintRepo.UpdateOccurrence(ctx, fp.ID)
			return true, nil
		}
	}

	return false, nil
}

// generateFingerprint 生成告警指纹
func (s *DedupService) generateFingerprint(event *biz.AlertEvent, keysJSON string) string {
	var keyList []string
	if err := json.Unmarshal([]byte(keysJSON), &keyList); err != nil {
		return ""
	}

	data := make(map[string]string)
	labels := make(map[string]string)
	json.Unmarshal([]byte(event.Labels), &labels)

	for _, key := range keyList {
		switch key {
		case "severity":
			data[key] = event.Severity
		case "ruleName":
			data[key] = event.RuleName
		default:
			// 从标签中提取
			if val, ok := labels[key]; ok {
				data[key] = val
			}
		}
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// RecordFingerprint 记录告警指纹
func (s *DedupService) RecordFingerprint(ctx context.Context, event *biz.AlertEvent, subscriptionID uint) error {
	rules, err := s.dedupRepo.ListBySubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		fingerprint := s.generateFingerprint(event, rule.FingerprintKeys)
		now := time.Now()

		fp, err := s.fingerprintRepo.GetByFingerprint(ctx, subscriptionID, fingerprint)
		if err != nil {
			// 创建新记录
			s.fingerprintRepo.Create(ctx, &biz.AlertFingerprint{
				SubscriptionID:  subscriptionID,
				Fingerprint:     fingerprint,
				RuleName:        event.RuleName,
				Severity:        event.Severity,
				Labels:          event.Labels,
				FirstSeenAt:     now,
				LastSeenAt:      now,
				LastSentAt:      &now,
				OccurrenceCount: 1,
			})
		} else {
			// 更新记录
			s.fingerprintRepo.UpdateSent(ctx, fp.ID, now)
		}
	}

	return nil
}

// CleanupExpiredFingerprints 清理过期指纹（定时任务）
func (s *DedupService) CleanupExpiredFingerprints(ctx context.Context) error {
	// 清理30天前的指纹记录
	before := time.Now().AddDate(0, 0, -30)
	return s.fingerprintRepo.CleanupExpired(ctx, before)
}
