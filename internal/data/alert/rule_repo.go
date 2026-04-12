package alert

import (
	"context"
	"fmt"
	"time"

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

// fillRuleGroupNames 批量回填规则分组名称
func (r *RuleRepo) fillRuleGroupNames(ctx context.Context, list []*biz.AlertRule) {
	if len(list) == 0 {
		return
	}

	// 收集所有规则分组 ID
	ruleGroupIDs := make(map[uint]bool)
	for _, rule := range list {
		if rule.RuleGroupID > 0 {
			ruleGroupIDs[rule.RuleGroupID] = true
		}
	}

	if len(ruleGroupIDs) == 0 {
		return
	}

	// 查询规则分组名称
	var ids []uint
	for id := range ruleGroupIDs {
		ids = append(ids, id)
	}

	var ruleGroups []struct {
		ID   uint   `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	r.db.WithContext(ctx).Table("alert_rule_groups").
		Select("id, name").
		Where("id IN ?", ids).
		Scan(&ruleGroups)

	// 构建 ID -> Name 映射
	nameMap := make(map[uint]string)
	for _, rg := range ruleGroups {
		nameMap[rg.ID] = rg.Name
	}

	// 回填到规则对象
	for _, rule := range list {
		if name, ok := nameMap[rule.RuleGroupID]; ok {
			rule.RuleGroupName = name
		}
	}
}

func (r *RuleRepo) Create(ctx context.Context, rule *biz.AlertRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *RuleRepo) Update(ctx context.Context, rule *biz.AlertRule) error {
	return r.db.WithContext(ctx).Model(rule).Omit("created_at").Select("*").Updates(rule).Error
}

func (r *RuleRepo) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	// 过滤掉虚拟字段和不应该更新的字段
	filteredFields := make(map[string]interface{})
	for k, v := range fields {
		// 跳过虚拟字段和系统字段
		if k == "ruleGroupName" || k == "id" || k == "createdAt" || k == "updatedAt" || k == "deletedAt" {
			continue
		}
		filteredFields[k] = v
	}

	if len(filteredFields) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Model(&biz.AlertRule{}).Where("id = ?", id).Updates(filteredFields).Error
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
	if err := q.Order("id desc").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}

	// 关联查询规则分组名称
	r.fillRuleGroupNames(ctx, list)

	return list, total, nil
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

// FillEvalTimesFromCache 从缓存回填评估时间
func (r *RuleRepo) FillEvalTimesFromCache(ctx context.Context, list []*biz.AlertRule, evalCache interface{}) error {
	if len(list) == 0 {
		return nil
	}

	// 类型断言，获取 EvalCache 接口
	type evalCacheGetter interface {
		GetEvalTimes(ctx context.Context, ruleIDs []uint) (map[uint]*time.Time, error)
	}

	cache, ok := evalCache.(evalCacheGetter)
	if !ok {
		return fmt.Errorf("evalCache 类型断言失败")
	}

	// 收集规则 ID
	ruleIDs := make([]uint, len(list))
	for i, rule := range list {
		ruleIDs[i] = rule.ID
	}

	// 批量获取评估时间（优先 Redis）
	evalTimes, err := cache.GetEvalTimes(ctx, ruleIDs)
	if err != nil {
		return err
	}

	// 回填到规则对象
	for _, rule := range list {
		if evalTime, ok := evalTimes[rule.ID]; ok {
			rule.LastEvalAt = evalTime
		}
	}

	return nil
}
