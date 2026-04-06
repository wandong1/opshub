package alert

import (
	"context"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"gorm.io/gorm"
)

type SilenceRuleRepo struct {
	db *gorm.DB
}

func NewSilenceRuleRepo(db *gorm.DB) *SilenceRuleRepo {
	return &SilenceRuleRepo{db: db}
}

func (r *SilenceRuleRepo) Create(ctx context.Context, rule *biz.AlertSilenceRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *SilenceRuleRepo) Update(ctx context.Context, rule *biz.AlertSilenceRule) error {
	return r.db.WithContext(ctx).Model(rule).Omit("created_at").Select("*").Updates(rule).Error
}

func (r *SilenceRuleRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&biz.AlertSilenceRule{}, id).Error
}

func (r *SilenceRuleRepo) GetByID(ctx context.Context, id uint) (*biz.AlertSilenceRule, error) {
	var rule biz.AlertSilenceRule
	if err := r.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *SilenceRuleRepo) List(ctx context.Context, page, pageSize int) ([]*biz.AlertSilenceRule, int64, error) {
	var list []*biz.AlertSilenceRule
	var total int64

	q := r.db.WithContext(ctx).Model(&biz.AlertSilenceRule{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := q.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *SilenceRuleRepo) ListActiveRules(ctx context.Context) ([]*biz.AlertSilenceRule, error) {
	var rules []*biz.AlertSilenceRule
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&rules).Error
	return rules, err
}

func (r *SilenceRuleRepo) FindMatchingRule(ctx context.Context, severity, ruleName, labels string) (*biz.AlertSilenceRule, error) {
	var rule biz.AlertSilenceRule
	err := r.db.WithContext(ctx).
		Where("enabled = ? AND severity = ? AND rule_name = ? AND labels = ?", true, severity, ruleName, labels).
		First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}
