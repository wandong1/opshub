package inspection_mgmt

import (
	"context"


	"gorm.io/gorm"
)

type TaskRepository interface {
	Create(ctx context.Context, task *InspectionTask) error
	Update(ctx context.Context, task *InspectionTask) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*InspectionTask, error)
	GetByName(ctx context.Context, name string) (*InspectionTask, error)
	List(ctx context.Context, page, pageSize int, name string, taskType string, enabled *bool) ([]*InspectionTask, int64, error)
	GetEnabledTasks(ctx context.Context) ([]*InspectionTask, error)
}

type taskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(ctx context.Context, task *InspectionTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *taskRepository) Update(ctx context.Context, task *InspectionTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *taskRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&InspectionTask{}, id).Error
}

func (r *taskRepository) GetByID(ctx context.Context, id uint) (*InspectionTask, error) {
	var task InspectionTask
	err := r.db.WithContext(ctx).First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *taskRepository) GetByName(ctx context.Context, name string) (*InspectionTask, error) {
	var task InspectionTask
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *taskRepository) List(ctx context.Context, page, pageSize int, name string, taskType string, enabled *bool) ([]*InspectionTask, int64, error) {
	var tasks []*InspectionTask
	var total int64

	query := r.db.WithContext(ctx).Model(&InspectionTask{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if taskType != "" {
		query = query.Where("task_type = ?", taskType)
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

func (r *taskRepository) GetEnabledTasks(ctx context.Context) ([]*InspectionTask, error) {
	var tasks []*InspectionTask
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&tasks).Error
	return tasks, err
}
