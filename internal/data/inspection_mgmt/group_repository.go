package inspection_mgmt

import (
	"context"

	"gorm.io/gorm"
)

type GroupRepository interface {
	Create(ctx context.Context, group *InspectionGroup) error
	Update(ctx context.Context, group *InspectionGroup) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*InspectionGroup, error)
	GetByName(ctx context.Context, name string) (*InspectionGroup, error)
	List(ctx context.Context, page, pageSize int, name, status string) ([]*InspectionGroup, int64, error)
	GetAll(ctx context.Context) ([]*InspectionGroup, error)
	GetByIDs(ctx context.Context, ids []uint) ([]*InspectionGroup, error)
	GetStats(ctx context.Context) (map[string]interface{}, error)
}

type groupRepository struct {
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &groupRepository{db: db}
}

func (r *groupRepository) Create(ctx context.Context, group *InspectionGroup) error {
	return r.db.WithContext(ctx).Create(group).Error
}

func (r *groupRepository) Update(ctx context.Context, group *InspectionGroup) error {
	return r.db.WithContext(ctx).Save(group).Error
}

func (r *groupRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&InspectionGroup{}, id).Error
}

func (r *groupRepository) GetByID(ctx context.Context, id uint) (*InspectionGroup, error) {
	var group InspectionGroup
	err := r.db.WithContext(ctx).First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *groupRepository) GetByName(ctx context.Context, name string) (*InspectionGroup, error) {
	var group InspectionGroup
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *groupRepository) List(ctx context.Context, page, pageSize int, name, status string) ([]*InspectionGroup, int64, error) {
	var groups []*InspectionGroup
	var total int64

	query := r.db.WithContext(ctx).Model(&InspectionGroup{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("sort ASC, id DESC").Offset(offset).Limit(pageSize).Find(&groups).Error
	return groups, total, err
}

func (r *groupRepository) GetAll(ctx context.Context) ([]*InspectionGroup, error) {
	var groups []*InspectionGroup
	err := r.db.WithContext(ctx).Where("status = ?", "enabled").Order("sort ASC, id DESC").Find(&groups).Error
	return groups, err
}

func (r *groupRepository) GetByIDs(ctx context.Context, ids []uint) ([]*InspectionGroup, error) {
	var groups []*InspectionGroup
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&groups).Error
	return groups, err
}

func (r *groupRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	var total, enabled, disabled int64
	var itemCount int64

	// 统计总数
	if err := r.db.WithContext(ctx).Model(&InspectionGroup{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// 统计启用数
	if err := r.db.WithContext(ctx).Model(&InspectionGroup{}).Where("status = ?", "enabled").Count(&enabled).Error; err != nil {
		return nil, err
	}

	// 统计禁用数
	if err := r.db.WithContext(ctx).Model(&InspectionGroup{}).Where("status = ?", "disabled").Count(&disabled).Error; err != nil {
		return nil, err
	}

	// 统计巡检项总数 - 只统计属于现有巡检组的巡检项
	// 使用子查询确保只统计有效巡检组的巡检项
	if err := r.db.WithContext(ctx).Model(&InspectionItem{}).
		Where("group_id IN (?)", r.db.Model(&InspectionGroup{}).Select("id")).
		Count(&itemCount).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total":    total,
		"enabled":  enabled,
		"disabled": disabled,
		"items":    itemCount,
	}, nil
}
