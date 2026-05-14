package alert

import (
	"context"
	"time"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"gorm.io/gorm"
)

// PatrolRepo 巡检配置仓储
type PatrolRepo struct {
	db *gorm.DB
}

func NewPatrolRepo(db *gorm.DB) *PatrolRepo {
	return &PatrolRepo{db: db}
}

// GetBySubscriptionRuleID 根据订阅规则ID查询巡检配置
func (r *PatrolRepo) GetBySubscriptionRuleID(ctx context.Context, subscriptionRuleID uint) (*biz.AlertSubscriptionRulePatrol, error) {
	var patrol biz.AlertSubscriptionRulePatrol
	err := r.db.WithContext(ctx).Where("subscription_rule_id = ?", subscriptionRuleID).First(&patrol).Error
	if err != nil {
		return nil, err
	}
	return &patrol, nil
}

// Create 创建巡检配置
func (r *PatrolRepo) Create(ctx context.Context, patrol *biz.AlertSubscriptionRulePatrol) error {
	return r.db.WithContext(ctx).Create(patrol).Error
}

// Update 更新巡检配置
func (r *PatrolRepo) Update(ctx context.Context, patrol *biz.AlertSubscriptionRulePatrol) error {
	return r.db.WithContext(ctx).Save(patrol).Error
}

// Delete 删除巡检配置
func (r *PatrolRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&biz.AlertSubscriptionRulePatrol{}, id).Error
}

// ListEnabledPatrols 查询所有启用的巡检配置
func (r *PatrolRepo) ListEnabledPatrols(ctx context.Context) ([]*biz.AlertSubscriptionRulePatrol, error) {
	var patrols []*biz.AlertSubscriptionRulePatrol
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&patrols).Error
	return patrols, err
}

// UpdateNextPatrolTime 更新下次巡检时间
func (r *PatrolRepo) UpdateNextPatrolTime(ctx context.Context, id uint, nextTime time.Time) error {
	return r.db.WithContext(ctx).Model(&biz.AlertSubscriptionRulePatrol{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_patrol_at": time.Now(),
			"next_patrol_at": nextTime,
		}).Error
}

// PatrolReportRepo 巡检报告仓储
type PatrolReportRepo struct {
	db *gorm.DB
}

func NewPatrolReportRepo(db *gorm.DB) *PatrolReportRepo {
	return &PatrolReportRepo{db: db}
}

// Create 创建巡检报告
func (r *PatrolReportRepo) Create(ctx context.Context, report *biz.AlertPatrolReport) error {
	return r.db.WithContext(ctx).Create(report).Error
}

// Update 更新巡检报告
func (r *PatrolReportRepo) Update(ctx context.Context, report *biz.AlertPatrolReport) error {
	return r.db.WithContext(ctx).Save(report).Error
}

// ListBySubscriptionRule 查询订阅规则的巡检报告列表
func (r *PatrolReportRepo) ListBySubscriptionRule(ctx context.Context, subscriptionRuleID uint, page, pageSize int) ([]*biz.AlertPatrolReport, int64, error) {
	var reports []*biz.AlertPatrolReport
	var total int64

	query := r.db.WithContext(ctx).Model(&biz.AlertPatrolReport{}).Where("subscription_rule_id = ?", subscriptionRuleID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&reports).Error

	return reports, total, err
}

// GetByID 根据ID查询巡检报告
func (r *PatrolReportRepo) GetByID(ctx context.Context, id uint) (*biz.AlertPatrolReport, error) {
	var report biz.AlertPatrolReport
	err := r.db.WithContext(ctx).First(&report, id).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}

// CleanupOldReports 清理旧报告（保留30天）
func (r *PatrolReportRepo) CleanupOldReports(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).Where("created_at < ?", before).Delete(&biz.AlertPatrolReport{}).Error
}
