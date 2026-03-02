package asset

import (
	"context"

	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	"gorm.io/gorm"
)

type serviceLabelRepo struct {
	db *gorm.DB
}

// NewServiceLabelRepo 创建服务标签仓库
func NewServiceLabelRepo(db *gorm.DB) asset.ServiceLabelRepo {
	return &serviceLabelRepo{db: db}
}

func (r *serviceLabelRepo) Create(ctx context.Context, label *asset.ServiceLabel) error {
	return r.db.WithContext(ctx).Create(label).Error
}

func (r *serviceLabelRepo) Update(ctx context.Context, label *asset.ServiceLabel) error {
	return r.db.WithContext(ctx).Save(label).Error
}

func (r *serviceLabelRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&asset.ServiceLabel{}, id).Error
}

func (r *serviceLabelRepo) GetByID(ctx context.Context, id uint) (*asset.ServiceLabel, error) {
	var label asset.ServiceLabel
	err := r.db.WithContext(ctx).First(&label, id).Error
	if err != nil {
		return nil, err
	}
	return &label, nil
}

func (r *serviceLabelRepo) List(ctx context.Context, page, pageSize int, keyword string) ([]*asset.ServiceLabel, int64, error) {
	var labels []*asset.ServiceLabel
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.ServiceLabel{})
	if keyword != "" {
		query = query.Where("name LIKE ? OR match_processes LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Order("id DESC").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&labels).Error; err != nil {
		return nil, 0, err
	}
	return labels, total, nil
}

func (r *serviceLabelRepo) GetAllEnabled(ctx context.Context) ([]*asset.ServiceLabel, error) {
	var labels []*asset.ServiceLabel
	err := r.db.WithContext(ctx).Where("status = ?", 1).Find(&labels).Error
	if err != nil {
		return nil, err
	}
	return labels, nil
}
