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
	// group_id = 0 表示对所有分组可见（通用配置）
	// 当传入 groupID > 0 时，查询 group_id = 0 或 group_id = 指定ID
	// 当传入 groupID = 0 时，只查询 group_id = 0（通用配置）
	if groupID > 0 {
		query = query.Where("group_id = 0 OR group_id = ?", groupID)
	} else {
		// 不添加 group_id 过滤，返回所有配置（包括通用配置和专属配置）
		// 或者只返回通用配置：query = query.Where("group_id = 0")
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
