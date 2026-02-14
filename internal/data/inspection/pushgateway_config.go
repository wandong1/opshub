package inspection

import (
	"context"

	biz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	"gorm.io/gorm"
)

type pushgatewayConfigRepo struct{ db *gorm.DB }

func NewPushgatewayConfigRepo(db *gorm.DB) biz.PushgatewayConfigRepo {
	return &pushgatewayConfigRepo{db: db}
}

func (r *pushgatewayConfigRepo) Create(ctx context.Context, config *biz.PushgatewayConfig) error {
	return r.db.WithContext(ctx).Create(config).Error
}

func (r *pushgatewayConfigRepo) Update(ctx context.Context, config *biz.PushgatewayConfig) error {
	return r.db.WithContext(ctx).Model(config).Omit("created_at").Updates(config).Error
}

func (r *pushgatewayConfigRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&biz.PushgatewayConfig{}, id).Error
}

func (r *pushgatewayConfigRepo) GetByID(ctx context.Context, id uint) (*biz.PushgatewayConfig, error) {
	var config biz.PushgatewayConfig
	if err := r.db.WithContext(ctx).First(&config, id).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *pushgatewayConfigRepo) List(ctx context.Context) ([]*biz.PushgatewayConfig, error) {
	var configs []*biz.PushgatewayConfig
	if err := r.db.WithContext(ctx).Order("id DESC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}
