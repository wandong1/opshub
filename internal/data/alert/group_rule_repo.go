package alert

import (
	"context"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"gorm.io/gorm"
)

type GroupRuleRepo struct {
	db *gorm.DB
}

func NewGroupRuleRepo(db *gorm.DB) *GroupRuleRepo {
	return &GroupRuleRepo{db: db}
}

func (r *GroupRuleRepo) Create(ctx context.Context, rule *biz.AlertGroupRule) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除该订阅的旧规则
		if err := tx.Where("subscription_id = ?", rule.SubscriptionID).Delete(&biz.AlertGroupRule{}).Error; err != nil {
			return err
		}
		// 创建新规则
		return tx.Create(rule).Error
	})
}

func (r *GroupRuleRepo) Update(ctx context.Context, rule *biz.AlertGroupRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *GroupRuleRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&biz.AlertGroupRule{}, id).Error
}

func (r *GroupRuleRepo) GetByID(ctx context.Context, id uint) (*biz.AlertGroupRule, error) {
	var rule biz.AlertGroupRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	return &rule, err
}

func (r *GroupRuleRepo) ListBySubscription(ctx context.Context, subscriptionID uint) ([]*biz.AlertGroupRule, error) {
	var rules []*biz.AlertGroupRule
	err := r.db.WithContext(ctx).
		Where("subscription_id = ?", subscriptionID).
		Find(&rules).Error
	return rules, err
}

func (r *GroupRuleRepo) List(ctx context.Context) ([]*biz.AlertGroupRule, error) {
	var rules []*biz.AlertGroupRule
	err := r.db.WithContext(ctx).Find(&rules).Error
	return rules, err
}
