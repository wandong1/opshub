package repository

import (
	"context"

	"github.com/ydcloud-dy/opshub/plugins/inspection/model"

	"gorm.io/gorm"
)

type ItemRepository interface {
	Create(ctx context.Context, item *model.InspectionItem) error
	Update(ctx context.Context, item *model.InspectionItem) error
	Delete(ctx context.Context, id uint) error
	DeleteByGroupID(ctx context.Context, groupID uint) error
	GetByID(ctx context.Context, id uint) (*model.InspectionItem, error)
	List(ctx context.Context, page, pageSize int, groupID uint, name, status string) ([]*model.InspectionItem, int64, error)
	GetByGroupID(ctx context.Context, groupID uint) ([]*model.InspectionItem, error)
	GetByIDs(ctx context.Context, ids []uint) ([]*model.InspectionItem, error)
}

type itemRepository struct {
	db *gorm.DB
}

func NewItemRepository(db *gorm.DB) ItemRepository {
	return &itemRepository{db: db}
}

func (r *itemRepository) Create(ctx context.Context, item *model.InspectionItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *itemRepository) Update(ctx context.Context, item *model.InspectionItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *itemRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.InspectionItem{}, id).Error
}

func (r *itemRepository) DeleteByGroupID(ctx context.Context, groupID uint) error {
	return r.db.WithContext(ctx).Where("group_id = ?", groupID).Delete(&model.InspectionItem{}).Error
}

func (r *itemRepository) GetByID(ctx context.Context, id uint) (*model.InspectionItem, error) {
	var item model.InspectionItem
	err := r.db.WithContext(ctx).First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *itemRepository) List(ctx context.Context, page, pageSize int, groupID uint, name, status string) ([]*model.InspectionItem, int64, error) {
	var items []*model.InspectionItem
	var total int64

	query := r.db.WithContext(ctx).Model(&model.InspectionItem{})

	if groupID > 0 {
		query = query.Where("group_id = ?", groupID)
	}
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
	err := query.Order("sort ASC, id DESC").Offset(offset).Limit(pageSize).Find(&items).Error
	return items, total, err
}

func (r *itemRepository) GetByGroupID(ctx context.Context, groupID uint) ([]*model.InspectionItem, error) {
	var items []*model.InspectionItem
	err := r.db.WithContext(ctx).Where("group_id = ? AND status = ?", groupID, "enabled").
		Order("sort ASC, id DESC").Find(&items).Error
	return items, err
}

func (r *itemRepository) GetByIDs(ctx context.Context, ids []uint) ([]*model.InspectionItem, error) {
	var items []*model.InspectionItem
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Order("sort ASC, id DESC").Find(&items).Error
	return items, err
}
