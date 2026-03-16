package inspection_mgmt

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ExecutionRecordRepository 巡检执行记录仓储接口
type ExecutionRecordRepository interface {
	// 主表操作
	CreateRecord(ctx context.Context, record *InspectionExecutionRecord) error
	UpdateRecord(ctx context.Context, record *InspectionExecutionRecord) error
	GetRecordByID(ctx context.Context, id uint) (*InspectionExecutionRecord, error)
	ListRecords(ctx context.Context, page, pageSize int, taskID uint, status string,
		startTime, endTime *time.Time) ([]*InspectionExecutionRecord, int64, error)
	DeleteRecord(ctx context.Context, id uint) error

	// 明细操作
	BatchCreateDetails(ctx context.Context, details []*InspectionExecutionDetail) error
	GetDetailsByExecutionID(ctx context.Context, executionID uint) ([]*InspectionExecutionDetail, error)

	// 统计
	GetRecordStats(ctx context.Context, executionID uint) (map[string]interface{}, error)
}

type executionRecordRepository struct {
	db *gorm.DB
}

// NewExecutionRecordRepository 创建执行记录仓储
func NewExecutionRecordRepository(db *gorm.DB) ExecutionRecordRepository {
	return &executionRecordRepository{db: db}
}

// CreateRecord 创建执行记录
func (r *executionRecordRepository) CreateRecord(ctx context.Context, record *InspectionExecutionRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// UpdateRecord 更新执行记录
func (r *executionRecordRepository) UpdateRecord(ctx context.Context, record *InspectionExecutionRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

// GetRecordByID 根据ID获取执行记录
func (r *executionRecordRepository) GetRecordByID(ctx context.Context, id uint) (*InspectionExecutionRecord, error) {
	var record InspectionExecutionRecord
	err := r.db.WithContext(ctx).First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ListRecords 分页查询执行记录列表
func (r *executionRecordRepository) ListRecords(ctx context.Context, page, pageSize int, taskID uint, status string,
	startTime, endTime *time.Time) ([]*InspectionExecutionRecord, int64, error) {
	var records []*InspectionExecutionRecord
	var total int64

	query := r.db.WithContext(ctx).Model(&InspectionExecutionRecord{})

	// 筛选条件
	if taskID > 0 {
		query = query.Where("task_id = ?", taskID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if startTime != nil {
		query = query.Where("started_at >= ?", startTime)
	}
	if endTime != nil {
		query = query.Where("started_at <= ?", endTime)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("started_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error
	return records, total, err
}

// DeleteRecord 删除执行记录（级联删除明细）
func (r *executionRecordRepository) DeleteRecord(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&InspectionExecutionRecord{}, id).Error
}

// BatchCreateDetails 批量创建执行明细
func (r *executionRecordRepository) BatchCreateDetails(ctx context.Context, details []*InspectionExecutionDetail) error {
	if len(details) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(details, 100).Error
}

// GetDetailsByExecutionID 获取执行记录的所有明细
func (r *executionRecordRepository) GetDetailsByExecutionID(ctx context.Context, executionID uint) ([]*InspectionExecutionDetail, error) {
	var details []*InspectionExecutionDetail
	err := r.db.WithContext(ctx).
		Where("execution_id = ?", executionID).
		Order("group_name ASC, item_name ASC, host_name ASC").
		Find(&details).Error
	return details, err
}

// GetRecordStats 获取执行记录统计信息
func (r *executionRecordRepository) GetRecordStats(ctx context.Context, executionID uint) (map[string]interface{}, error) {
	var stats struct {
		TotalExecutions    int64
		SuccessCount       int64
		FailedCount        int64
		AssertionPassCount int64
		AssertionFailCount int64
		AssertionSkipCount int64
		AvgDuration        float64
	}

	err := r.db.WithContext(ctx).Model(&InspectionExecutionDetail{}).
		Select(`
			COUNT(*) as total_executions,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count,
			SUM(CASE WHEN assertion_result = 'pass' THEN 1 ELSE 0 END) as assertion_pass_count,
			SUM(CASE WHEN assertion_result = 'fail' THEN 1 ELSE 0 END) as assertion_fail_count,
			SUM(CASE WHEN assertion_result = 'skip' THEN 1 ELSE 0 END) as assertion_skip_count,
			AVG(duration) as avg_duration
		`).
		Where("execution_id = ?", executionID).
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"totalExecutions":    stats.TotalExecutions,
		"successCount":       stats.SuccessCount,
		"failedCount":        stats.FailedCount,
		"assertionPassCount": stats.AssertionPassCount,
		"assertionFailCount": stats.AssertionFailCount,
		"assertionSkipCount": stats.AssertionSkipCount,
		"avgDuration":        stats.AvgDuration,
	}, nil
}
