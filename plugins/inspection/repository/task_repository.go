package repository

import (
	"context"

	"github.com/ydcloud-dy/opshub/plugins/inspection/model"

	"gorm.io/gorm"
)

type TaskRepository interface {
	Create(ctx context.Context, task *model.InspectionTask) error
	Update(ctx context.Context, task *model.InspectionTask) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*model.InspectionTask, error)
	GetByName(ctx context.Context, name string) (*model.InspectionTask, error)
	List(ctx context.Context, page, pageSize int, name string, enabled *bool) ([]*model.InspectionTask, int64, error)
	GetEnabledTasks(ctx context.Context) ([]*model.InspectionTask, error)
}

type taskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(ctx context.Context, task *model.InspectionTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *taskRepository) Update(ctx context.Context, task *model.InspectionTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *taskRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.InspectionTask{}, id).Error
}

func (r *taskRepository) GetByID(ctx context.Context, id uint) (*model.InspectionTask, error) {
	var task model.InspectionTask
	err := r.db.WithContext(ctx).First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *taskRepository) GetByName(ctx context.Context, name string) (*model.InspectionTask, error) {
	var task model.InspectionTask
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *taskRepository) List(ctx context.Context, page, pageSize int, name string, enabled *bool) ([]*model.InspectionTask, int64, error) {
	var tasks []*model.InspectionTask
	var total int64

	query := r.db.WithContext(ctx).Model(&model.InspectionTask{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&tasks).Error
	return tasks, total, err
}

func (r *taskRepository) GetEnabledTasks(ctx context.Context) ([]*model.InspectionTask, error) {
	var tasks []*model.InspectionTask
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&tasks).Error
	return tasks, err
}
