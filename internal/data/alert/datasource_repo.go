package alert

import (
	"context"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"gorm.io/gorm"
)

type DataSourceRepo struct{ db *gorm.DB }

func NewDataSourceRepo(db *gorm.DB) *DataSourceRepo {
	return &DataSourceRepo{db: db}
}

func (r *DataSourceRepo) Create(ctx context.Context, ds *biz.AlertDataSource) error {
	return r.db.WithContext(ctx).Create(ds).Error
}

func (r *DataSourceRepo) Update(ctx context.Context, ds *biz.AlertDataSource) error {
	return r.db.WithContext(ctx).Model(ds).Omit("created_at").Select("*").Updates(ds).Error
}

func (r *DataSourceRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&biz.AlertDataSource{}, id).Error
}

func (r *DataSourceRepo) GetByID(ctx context.Context, id uint) (*biz.AlertDataSource, error) {
	var ds biz.AlertDataSource
	if err := r.db.WithContext(ctx).First(&ds, id).Error; err != nil {
		return nil, err
	}
	return &ds, nil
}

func (r *DataSourceRepo) List(ctx context.Context) ([]*biz.AlertDataSource, error) {
	var list []*biz.AlertDataSource
	if err := r.db.WithContext(ctx).Order("id asc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *DataSourceRepo) ListEnabled(ctx context.Context) ([]*biz.AlertDataSource, error) {
	var list []*biz.AlertDataSource
	if err := r.db.WithContext(ctx).Where("status = 1").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *DataSourceRepo) GetByProxyToken(ctx context.Context, proxyToken string) (*biz.AlertDataSource, error) {
	var ds biz.AlertDataSource
	if err := r.db.WithContext(ctx).Where("proxy_token = ?", proxyToken).First(&ds).Error; err != nil {
		return nil, err
	}
	return &ds, nil
}

// ListByAssetGroupIDs 根据业务分组ID列表查询数据源
func (r *DataSourceRepo) ListByAssetGroupIDs(ctx context.Context, assetGroupIDs []uint) ([]*biz.AlertDataSource, error) {
	if len(assetGroupIDs) == 0 {
		return r.List(ctx)
	}

	var list []*biz.AlertDataSource
	err := r.db.WithContext(ctx).
		Joins("INNER JOIN alert_datasource_group_relations ON alert_datasource_group_relations.data_source_id = alert_datasources.id").
		Where("alert_datasource_group_relations.asset_group_id IN ?", assetGroupIDs).
		Group("alert_datasources.id").
		Order("alert_datasources.id asc").
		Find(&list).Error
	return list, err
}
