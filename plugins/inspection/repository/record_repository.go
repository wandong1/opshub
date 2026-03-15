package repository

import (
	"context"
	"time"

	"github.com/ydcloud-dy/opshub/plugins/inspection/model"

	"gorm.io/gorm"
)

type RecordRepository interface {
	Create(ctx context.Context, record *model.InspectionRecord) error
	GetByID(ctx context.Context, id uint) (*model.InspectionRecord, error)
	List(ctx context.Context, page, pageSize int, taskID, groupID, itemID, hostID uint, status string, startTime, endTime *time.Time) ([]*model.InspectionRecord, int64, error)
	DeleteOldRecords(ctx context.Context, days int) error
}

type recordRepository struct {
	db *gorm.DB
}

func NewRecordRepository(db *gorm.DB) RecordRepository {
	return &recordRepository{db: db}
}

func (r *recordRepository) Create(ctx context.Context, record *model.InspectionRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *recordRepository) GetByID(ctx context.Context, id uint) (*model.InspectionRecord, error) {
	var record model.InspectionRecord
	err := r.db.WithContext(ctx).First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *recordRepository) List(ctx context.Context, page, pageSize int, taskID, groupID, itemID, hostID uint, status string, startTime, endTime *time.Time) ([]*model.InspectionRecord, int64, error) {
	var records []*model.InspectionRecord
	var total int64

	query := r.db.WithContext(ctx).Model(&model.InspectionRecord{})

	if taskID > 0 {
		query = query.Where("task_id = ?", taskID)
	}
	if groupID > 0 {
		query = query.Where("group_id = ?", groupID)
	}
	if itemID > 0 {
		query = query.Where("item_id = ?", itemID)
	}
	if hostID > 0 {
		query = query.Where("host_id = ?", hostID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if startTime != nil {
		query = query.Where("executed_at >= ?", startTime)
	}
	if endTime != nil {
		query = query.Where("executed_at <= ?", endTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("executed_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error
	return records, total, err
}

func (r *recordRepository) DeleteOldRecords(ctx context.Context, days int) error {
	cutoffTime := time.Now().AddDate(0, 0, -days)
	return r.db.WithContext(ctx).Where("executed_at < ?", cutoffTime).Delete(&model.InspectionRecord{}).Error
}
