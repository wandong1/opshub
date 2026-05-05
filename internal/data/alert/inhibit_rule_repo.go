package alert

import (
	"context"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"gorm.io/gorm"
)

type InhibitRuleRepo struct {
	db *gorm.DB
}

func NewInhibitRuleRepo(db *gorm.DB) *InhibitRuleRepo {
	return &InhibitRuleRepo{db: db}
}

func (r *InhibitRuleRepo) Create(ctx context.Context, rule *biz.AlertInhibitRule) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除该订阅的旧规则
		if err := tx.Where("subscription_id = ?", rule.SubscriptionID).Delete(&biz.AlertInhibitRule{}).Error; err != nil {
			return err
		}
		// 创建新规则
		return tx.Create(rule).Error
	})
}

func (r *InhibitRuleRepo) Update(ctx context.Context, rule *biz.AlertInhibitRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *InhibitRuleRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&biz.AlertInhibitRule{}, id).Error
}

func (r *InhibitRuleRepo) GetByID(ctx context.Context, id uint) (*biz.AlertInhibitRule, error) {
	var rule biz.AlertInhibitRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	return &rule, err
}

func (r *InhibitRuleRepo) ListBySubscription(ctx context.Context, subscriptionID uint) ([]*biz.AlertInhibitRule, error) {
	var rules []*biz.AlertInhibitRule
	err := r.db.WithContext(ctx).
		Where("subscription_id = ?", subscriptionID).
		Find(&rules).Error
	return rules, err
}

func (r *InhibitRuleRepo) List(ctx context.Context) ([]*biz.AlertInhibitRule, error) {
	var rules []*biz.AlertInhibitRule
	err := r.db.WithContext(ctx).Find(&rules).Error
	return rules, err
}
