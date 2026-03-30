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
		if err := tx.Where("subscription_id = ?", subscriptionID).Delete(&biz.AlertSubscriptionRule{}).Error; err != nil {
			return err
		}
		if len(rules) > 0 {
			return tx.Create(&rules).Error
		}
		return nil
	})
}

func (r *SubscriptionRuleRepo) ListBySubscription(ctx context.Context, subscriptionID uint) ([]*biz.AlertSubscriptionRule, error) {
	var list []*biz.AlertSubscriptionRule
	return list, r.db.WithContext(ctx).Where("subscription_id = ?", subscriptionID).Find(&list).Error
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
