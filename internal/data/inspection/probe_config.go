package inspection

import (
	"context"

	biz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	"gorm.io/gorm"
)

type probeConfigRepo struct{ db *gorm.DB }

func NewProbeConfigRepo(db *gorm.DB) biz.ProbeConfigRepo {
	return &probeConfigRepo{db: db}
}

func (r *probeConfigRepo) Create(ctx context.Context, config *biz.ProbeConfig) error {
	return r.db.WithContext(ctx).Create(config).Error
}

func (r *probeConfigRepo) Update(ctx context.Context, config *biz.ProbeConfig) error {
	return r.db.WithContext(ctx).Model(config).Omit("created_at").Select("*").Updates(config).Error
}

func (r *probeConfigRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&biz.ProbeConfig{}, id).Error
}

func (r *probeConfigRepo) GetByID(ctx context.Context, id uint) (*biz.ProbeConfig, error) {
	var config biz.ProbeConfig
	if err := r.db.WithContext(ctx).First(&config, id).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *probeConfigRepo) List(ctx context.Context, page, pageSize int, keyword, probeType, category string, groupID uint, status *int8) ([]*biz.ProbeConfig, int64, error) {
	var configs []*biz.ProbeConfig
	var total int64

	query := r.db.WithContext(ctx).Model(&biz.ProbeConfig{})
	if keyword != "" {
		query = query.Where("name LIKE ? OR target LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if probeType != "" {
		query = query.Where("type = ?", probeType)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if groupID > 0 {
		query = query.Where("group_id = ?", groupID)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&configs).Error; err != nil {
		return nil, 0, err
	}
	return configs, total, nil
}

func (r *probeConfigRepo) ListAll(ctx context.Context) ([]*biz.ProbeConfig, error) {
	var configs []*biz.ProbeConfig
	if err := r.db.WithContext(ctx).Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *probeConfigRepo) BatchCreate(ctx context.Context, configs []*biz.ProbeConfig) error {
	return r.db.WithContext(ctx).Create(&configs).Error
}
