package audit

import (
	"context"
	"time"

	"github.com/ydcloud-dy/opshub/internal/biz/audit"
	"gorm.io/gorm"
)

type middlewareAuditLogRepo struct {
	db *gorm.DB
}

func NewMiddlewareAuditLogRepo(db *gorm.DB) audit.MiddlewareAuditLogRepo {
	return &middlewareAuditLogRepo{db: db}
}

func (r *middlewareAuditLogRepo) Create(ctx context.Context, log *audit.SysMiddlewareAuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *middlewareAuditLogRepo) List(ctx context.Context, page, pageSize int, username, middlewareType, commandType, status, startTime, endTime string, middlewareID *uint) ([]*audit.SysMiddlewareAuditLog, int64, error) {
	var logs []*audit.SysMiddlewareAuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&audit.SysMiddlewareAuditLog{})

	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if middlewareType != "" {
		query = query.Where("middleware_type = ?", middlewareType)
	}
	if commandType != "" {
		query = query.Where("command_type = ?", commandType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if middlewareID != nil {
		query = query.Where("middleware_id = ?", *middlewareID)
	}
	if startTime != "" {
		t, err := time.Parse("2006-01-02", startTime)
		if err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if endTime != "" {
		t, err := time.Parse("2006-01-02", endTime)
		if err == nil {
			t = t.AddDate(0, 0, 1)
			query = query.Where("created_at < ?", t)
		}
	}
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page - 1) * pageSize).Limit(pageSize).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

func (r *middlewareAuditLogRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&audit.SysMiddlewareAuditLog{}, id).Error
}

func (r *middlewareAuditLogRepo) DeleteBatch(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Delete(&audit.SysMiddlewareAuditLog{}, ids).Error
}
