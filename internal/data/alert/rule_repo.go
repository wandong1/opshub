package alert

import (
	"context"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"gorm.io/gorm"
)

type RuleGroupRepo struct{ db *gorm.DB }

func NewRuleGroupRepo(db *gorm.DB) *RuleGroupRepo {
	return &RuleGroupRepo{db: db}
}

func (r *RuleGroupRepo) Create(ctx context.Context, g *biz.AlertRuleGroup) error {
	return r.db.WithContext(ctx).Create(g).Error
}

func (r *RuleGroupRepo) Update(ctx context.Context, g *biz.AlertRuleGroup) error {
	return r.db.WithContext(ctx).Model(g).Omit("created_at").Select("*").Updates(g).Error
}

func (r *RuleGroupRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&biz.AlertRuleGroup{}, id).Error
}

func (r *RuleGroupRepo) GetByID(ctx context.Context, id uint) (*biz.AlertRuleGroup, error) {
	var g biz.AlertRuleGroup
	if err := r.db.WithContext(ctx).First(&g, id).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *RuleGroupRepo) ListByAssetGroup(ctx context.Context, assetGroupID uint) ([]*biz.AlertRuleGroup, error) {
	var list []*biz.AlertRuleGroup
	q := r.db.WithContext(ctx).Order("sort asc, id asc")
	if assetGroupID > 0 {
		q = q.Where("asset_group_id = ?", assetGroupID)
	}
	return list, q.Find(&list).Error
}

// ----

type RuleRepo struct{ db *gorm.DB }

func NewRuleRepo(db *gorm.DB) *RuleRepo {
	return &RuleRepo{db: db}
}

func (r *RuleRepo) Create(ctx context.Context, rule *biz.AlertRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *RuleRepo) Update(ctx context.Context, rule *biz.AlertRule) error {
	return r.db.WithContext(ctx).Model(rule).Omit("created_at").Select("*").Updates(rule).Error
}

func (r *RuleRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&biz.AlertRule{}, id).Error
}

func (r *RuleRepo) GetByID(ctx context.Context, id uint) (*biz.AlertRule, error) {
	var rule biz.AlertRule
	if err := r.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *RuleRepo) List(ctx context.Context, page, pageSize int, assetGroupID, ruleGroupID uint, keyword string, enabled *bool) ([]*biz.AlertRule, int64, error) {
	var list []*biz.AlertRule
	var total int64
	q := r.db.WithContext(ctx).Model(&biz.AlertRule{})
	if assetGroupID > 0 {
		q = q.Where("asset_group_id = ?", assetGroupID)
	}
	if ruleGroupID > 0 {
		q = q.Where("rule_group_id = ?", ruleGroupID)
	}
	if keyword != "" {
		q = q.Where("name LIKE ?", "%"+keyword+"%")
	}
	if enabled != nil {
		q = q.Where("enabled = ?", *enabled)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	return list, total, q.Order("id desc").Offset(offset).Limit(pageSize).Find(&list).Error
}

func (r *RuleRepo) ListEnabled(ctx context.Context) ([]*biz.AlertRule, error) {
	var list []*biz.AlertRule
	return list, r.db.WithContext(ctx).Where("enabled = true").Find(&list).Error
}

func (r *RuleRepo) UpdateLastEvalAt(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&biz.AlertRule{}).Where("id = ?", id).
		UpdateColumn("last_eval_at", gorm.Expr("NOW()")).Error
}

func (r *RuleRepo) ListByIDs(ctx context.Context, ids []uint) ([]*biz.AlertRule, error) {
	var list []*biz.AlertRule
	return list, r.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error
}
