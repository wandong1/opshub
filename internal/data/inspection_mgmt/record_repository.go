package inspection_mgmt

import (
	"context"
	"time"


	"gorm.io/gorm"
)

type RecordRepository interface {
	Create(ctx context.Context, record *InspectionRecord) error
	GetByID(ctx context.Context, id uint) (*InspectionRecord, error)
	List(ctx context.Context, page, pageSize int, taskID, groupID, itemID, hostID uint, status string, startTime, endTime *time.Time) ([]*InspectionRecord, int64, error)
	GetByTaskID(ctx context.Context, taskID uint) ([]*InspectionRecord, error)
	DeleteOldRecords(ctx context.Context, days int) error
	CleanupExcessRecords(ctx context.Context, keepCount int) error // 保留最新的 keepCount 条记录，删除其余
	GetTotalCount(ctx context.Context) (int64, error)
}

type recordRepository struct {
	db *gorm.DB
}

func NewRecordRepository(db *gorm.DB) RecordRepository {
	return &recordRepository{db: db}
}

func (r *recordRepository) Create(ctx context.Context, record *InspectionRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *recordRepository) GetByID(ctx context.Context, id uint) (*InspectionRecord, error) {
	var record InspectionRecord
	err := r.db.WithContext(ctx).First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *recordRepository) List(ctx context.Context, page, pageSize int, taskID, groupID, itemID, hostID uint, status string, startTime, endTime *time.Time) ([]*InspectionRecord, int64, error) {
	var records []*InspectionRecord
	var total int64

	query := r.db.WithContext(ctx).Model(&InspectionRecord{})

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

func (r *recordRepository) GetByTaskID(ctx context.Context, taskID uint) ([]*InspectionRecord, error) {
	var records []*InspectionRecord
	err := r.db.WithContext(ctx).Where("task_id = ?", taskID).Order("executed_at DESC").Find(&records).Error
	return records, err
}

func (r *recordRepository) DeleteOldRecords(ctx context.Context, days int) error {
	cutoffTime := time.Now().AddDate(0, 0, -days)
	return r.db.WithContext(ctx).Where("executed_at < ?", cutoffTime).Delete(&InspectionRecord{}).Error
}

// CleanupExcessRecords 清理超出保留数量的记录，保留最新的 keepCount 条
func (r *recordRepository) CleanupExcessRecords(ctx context.Context, keepCount int) error {
	// 先获取总记录数
	var total int64
	if err := r.db.WithContext(ctx).Model(&InspectionRecord{}).Count(&total).Error; err != nil {
		return err
	}

	// 如果记录数未超过保留数量，无需清理
	if total <= int64(keepCount) {
		return nil
	}

	// 使用子查询删除最旧的记录
	// 找到第 keepCount 条记录的 ID，删除 ID 小于等于该值的记录
	var cutoffID uint
	if err := r.db.WithContext(ctx).Model(&InspectionRecord{}).
		Select("id").
		Order("id DESC").
		Offset(keepCount).
		Limit(1).
		Pluck("id", &cutoffID).Error; err != nil {
		return err
	}

	// 删除 ID 小于等于 cutoffID 的记录
	return r.db.WithContext(ctx).Where("id <= ?", cutoffID).Delete(&InspectionRecord{}).Error
}

// GetTotalCount 获取总记录数
func (r *recordRepository) GetTotalCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&InspectionRecord{}).Count(&count).Error
	return count, err
}
