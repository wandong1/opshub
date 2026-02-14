package inspection

import (
	"context"

	biz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	"gorm.io/gorm"
)

type probeResultRepo struct{ db *gorm.DB }

func NewProbeResultRepo(db *gorm.DB) biz.ProbeResultRepo {
	return &probeResultRepo{db: db}
}

func (r *probeResultRepo) Create(ctx context.Context, result *biz.ProbeResult) error {
	return r.db.WithContext(ctx).Create(result).Error
}

func (r *probeResultRepo) ListByTaskID(ctx context.Context, taskID uint, page, pageSize int) ([]*biz.ProbeResult, int64, error) {
	var results []*biz.ProbeResult
	var total int64

	query := r.db.WithContext(ctx).Model(&biz.ProbeResult{}).Where("probe_task_id = ?", taskID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&results).Error; err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (r *probeResultRepo) CleanupByTaskID(ctx context.Context, taskID uint, keepCount int) error {
	// Find the ID threshold: keep the latest `keepCount` rows, delete the rest.
	var cutoffID uint
	err := r.db.WithContext(ctx).Model(&biz.ProbeResult{}).
		Where("probe_task_id = ?", taskID).
		Order("id DESC").Offset(keepCount).Limit(1).
		Select("id").Scan(&cutoffID).Error
	if err != nil || cutoffID == 0 {
		return nil // fewer than keepCount rows, nothing to clean
	}
	return r.db.WithContext(ctx).
		Where("probe_task_id = ? AND id <= ?", taskID, cutoffID).
		Delete(&biz.ProbeResult{}).Error
}
