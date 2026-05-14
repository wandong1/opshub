package alert

import (
	"context"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"gorm.io/gorm"
)

type SubscriptionRepo struct{ db *gorm.DB }

func NewSubscriptionRepo(db *gorm.DB) *SubscriptionRepo {
	return &SubscriptionRepo{db: db}
}

func (r *SubscriptionRepo) Create(ctx context.Context, s *biz.AlertSubscription) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *SubscriptionRepo) Update(ctx context.Context, s *biz.AlertSubscription) error {
	return r.db.WithContext(ctx).Model(s).Omit("created_at").Select("*").Updates(s).Error
}

func (r *SubscriptionRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&biz.AlertSubscription{}, id).Error
}

func (r *SubscriptionRepo) GetByID(ctx context.Context, id uint) (*biz.AlertSubscription, error) {
	var s biz.AlertSubscription
	if err := r.db.WithContext(ctx).First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SubscriptionRepo) List(ctx context.Context, assetGroupID uint) ([]*biz.AlertSubscription, error) {
	var list []*biz.AlertSubscription
	q := r.db.WithContext(ctx).Order("id desc")
	if assetGroupID > 0 {
		q = q.Where("asset_group_id = ?", assetGroupID)
	}
	return list, q.Find(&list).Error
}

func (r *SubscriptionRepo) ListEnabled(ctx context.Context) ([]*biz.AlertSubscription, error) {
	var list []*biz.AlertSubscription
	return list, r.db.WithContext(ctx).Where("enabled = true").Find(&list).Error
}

// --- SubscriptionRule ---

type SubscriptionRuleRepo struct{ db *gorm.DB }

func NewSubscriptionRuleRepo(db *gorm.DB) *SubscriptionRuleRepo {
	return &SubscriptionRuleRepo{db: db}
}

func (r *SubscriptionRuleRepo) SetRules(ctx context.Context, subscriptionID uint, rules []*biz.AlertSubscriptionRule) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 查询现有规则
		var existing []*biz.AlertSubscriptionRule
		if err := tx.Where("subscription_id = ?", subscriptionID).Find(&existing).Error; err != nil {
			return err
		}

		// 2. 构建映射：通过 ruleId + 配置特征匹配现有规则
		// 使用 ruleId 作为主键，因为一个订阅中同一个 ruleId 可能有多条规则（不同时间段）
		existingMap := make(map[uint]*biz.AlertSubscriptionRule)
		for _, e := range existing {
			existingMap[e.ID] = e
		}

		// 3. 构建新规则的指纹映射（用于匹配）
		type ruleFingerprint struct {
			RuleID        uint
			TimeRanges    string
			Severities    string
			DataSourceIDs string
			LabelMatchers string
		}

		// 为现有规则建立指纹索引
		fingerprintToID := make(map[ruleFingerprint]uint)
		for _, e := range existing {
			fp := ruleFingerprint{
				RuleID:        e.RuleID,
				TimeRanges:    e.TimeRanges,
				Severities:    e.Severities,
				DataSourceIDs: e.DataSourceIDs,
				LabelMatchers: e.LabelMatchers,
			}
			fingerprintToID[fp] = e.ID
		}

		// 4. 处理新规则：更新或创建
		processedIDs := make(map[uint]bool)
		for _, newRule := range rules {
			fp := ruleFingerprint{
				RuleID:        newRule.RuleID,
				TimeRanges:    newRule.TimeRanges,
				Severities:    newRule.Severities,
				DataSourceIDs: newRule.DataSourceIDs,
				LabelMatchers: newRule.LabelMatchers,
			}

			if existingID, exists := fingerprintToID[fp]; exists {
				// 找到匹配的现有规则，更新（保留 ID 和 CreatedAt）
				oldRule := existingMap[existingID]
				newRule.ID = oldRule.ID
				newRule.CreatedAt = oldRule.CreatedAt
				if err := tx.Save(newRule).Error; err != nil {
					return err
				}
				processedIDs[existingID] = true
			} else {
				// 没有匹配的规则，创建新规则
				if err := tx.Create(newRule).Error; err != nil {
					return err
				}
			}
		}

		// 5. 删除不再需要的规则
		for id := range existingMap {
			if !processedIDs[id] {
				if err := tx.Delete(&biz.AlertSubscriptionRule{}, id).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (r *SubscriptionRuleRepo) ListBySubscription(ctx context.Context, subscriptionID uint) ([]*biz.AlertSubscriptionRule, error) {
	var list []*biz.AlertSubscriptionRule
	return list, r.db.WithContext(ctx).Where("subscription_id = ?", subscriptionID).Find(&list).Error
}

func (r *SubscriptionRuleRepo) GetByID(ctx context.Context, id uint) (*biz.AlertSubscriptionRule, error) {
	var rule biz.AlertSubscriptionRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *SubscriptionRuleRepo) ListByRuleID(ctx context.Context, ruleID uint) ([]*biz.AlertSubscriptionRule, error) {
	var list []*biz.AlertSubscriptionRule
	// rule_id=0 表示全部规则，同时查精确匹配和全部规则
	return list, r.db.WithContext(ctx).Where("rule_id = ? OR rule_id = 0", ruleID).Find(&list).Error
}

// --- SubscriptionChannel ---

type SubscriptionChannelRepo struct{ db *gorm.DB }

func NewSubscriptionChannelRepo(db *gorm.DB) *SubscriptionChannelRepo {
	return &SubscriptionChannelRepo{db: db}
}

func (r *SubscriptionChannelRepo) SetChannels(ctx context.Context, subscriptionID uint, channelIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("subscription_id = ?", subscriptionID).Delete(&biz.AlertSubscriptionChannel{}).Error; err != nil {
			return err
		}
		if len(channelIDs) > 0 {
			var rows []biz.AlertSubscriptionChannel
			for _, cid := range channelIDs {
				rows = append(rows, biz.AlertSubscriptionChannel{SubscriptionID: subscriptionID, ChannelID: cid})
			}
			return tx.Create(&rows).Error
		}
		return nil
	})
}

func (r *SubscriptionChannelRepo) ListBySubscription(ctx context.Context, subscriptionID uint) ([]*biz.AlertSubscriptionChannel, error) {
	var list []*biz.AlertSubscriptionChannel
	return list, r.db.WithContext(ctx).Where("subscription_id = ?", subscriptionID).Find(&list).Error
}

// --- SubscriptionUser ---

type SubscriptionUserRepo struct{ db *gorm.DB }

func NewSubscriptionUserRepo(db *gorm.DB) *SubscriptionUserRepo {
	return &SubscriptionUserRepo{db: db}
}

func (r *SubscriptionUserRepo) SetUsers(ctx context.Context, subscriptionID uint, userIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("subscription_id = ?", subscriptionID).Delete(&biz.AlertSubscriptionUser{}).Error; err != nil {
			return err
		}
		if len(userIDs) > 0 {
			var rows []biz.AlertSubscriptionUser
			for _, uid := range userIDs {
				rows = append(rows, biz.AlertSubscriptionUser{SubscriptionID: subscriptionID, UserID: uid})
			}
			return tx.Create(&rows).Error
		}
		return nil
	})
}

func (r *SubscriptionUserRepo) ListBySubscription(ctx context.Context, subscriptionID uint) ([]*biz.AlertSubscriptionUser, error) {
	var list []*biz.AlertSubscriptionUser
	return list, r.db.WithContext(ctx).Where("subscription_id = ?", subscriptionID).Find(&list).Error
}

// --- SubscriptionLog ---

type SubscriptionLogRepo struct{ db *gorm.DB }

func NewSubscriptionLogRepo(db *gorm.DB) *SubscriptionLogRepo {
	return &SubscriptionLogRepo{db: db}
}

func (r *SubscriptionLogRepo) Create(ctx context.Context, log *biz.AlertSubscriptionLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *SubscriptionLogRepo) List(ctx context.Context, subscriptionID uint, page, pageSize int) ([]*biz.AlertSubscriptionLog, int64, error) {
	var list []*biz.AlertSubscriptionLog
	var total int64

	q := r.db.WithContext(ctx).Model(&biz.AlertSubscriptionLog{})
	if subscriptionID > 0 {
		q = q.Where("subscription_id = ?", subscriptionID)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := q.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

