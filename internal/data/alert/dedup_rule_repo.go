package alert

import (
	"context"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"gorm.io/gorm"
)

type DedupRuleRepo struct {
	db *gorm.DB
}

func NewDedupRuleRepo(db *gorm.DB) *DedupRuleRepo {
	return &DedupRuleRepo{db: db}
}

func (r *DedupRuleRepo) Create(ctx context.Context, rule *biz.AlertDedupRule) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除该订阅的旧规则
		if err := tx.Where("subscription_id = ?", rule.SubscriptionID).Delete(&biz.AlertDedupRule{}).Error; err != nil {
			return err
		}
		// 创建新规则
		return tx.Create(rule).Error
	})
}

func (r *DedupRuleRepo) Update(ctx context.Context, rule *biz.AlertDedupRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *DedupRuleRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&biz.AlertDedupRule{}, id).Error
}

func (r *DedupRuleRepo) GetByID(ctx context.Context, id uint) (*biz.AlertDedupRule, error) {
	var rule biz.AlertDedupRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	return &rule, err
}

func (r *DedupRuleRepo) ListBySubscription(ctx context.Context, subscriptionID uint) ([]*biz.AlertDedupRule, error) {
	var rules []*biz.AlertDedupRule
	err := r.db.WithContext(ctx).
		Where("subscription_id = ?", subscriptionID).
		Find(&rules).Error
	return rules, err
}

func (r *DedupRuleRepo) List(ctx context.Context) ([]*biz.AlertDedupRule, error) {
	var rules []*biz.AlertDedupRule
	err := r.db.WithContext(ctx).Find(&rules).Error
	return rules, err
}
