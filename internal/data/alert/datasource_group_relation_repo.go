package alert

import (
	"context"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"gorm.io/gorm"
)

type DataSourceGroupRelationRepo struct {
	db *gorm.DB
}

func NewDataSourceGroupRelationRepo(db *gorm.DB) *DataSourceGroupRelationRepo {
	return &DataSourceGroupRelationRepo{db: db}
}

// Create 创建关联
func (r *DataSourceGroupRelationRepo) Create(ctx context.Context, rel *biz.DataSourceGroupRelation) error {
	return r.db.WithContext(ctx).Create(rel).Error
}

// Delete 删除关联
func (r *DataSourceGroupRelationRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&biz.DataSourceGroupRelation{}, id).Error
}

// DeleteByDataSourceID 删除数据源的所有关联
func (r *DataSourceGroupRelationRepo) DeleteByDataSourceID(ctx context.Context, dataSourceID uint) error {
	return r.db.WithContext(ctx).Where("data_source_id = ?", dataSourceID).Delete(&biz.DataSourceGroupRelation{}).Error
}

// ListByDataSourceID 查询数据源的所有关联
func (r *DataSourceGroupRelationRepo) ListByDataSourceID(ctx context.Context, dataSourceID uint) ([]*biz.DataSourceGroupRelation, error) {
	var list []*biz.DataSourceGroupRelation
	err := r.db.WithContext(ctx).Where("data_source_id = ?", dataSourceID).Find(&list).Error
	return list, err
}

// ListByAssetGroupID 查询业务分组关联的所有数据源
func (r *DataSourceGroupRelationRepo) ListByAssetGroupID(ctx context.Context, assetGroupID uint) ([]*biz.DataSourceGroupRelation, error) {
	var list []*biz.DataSourceGroupRelation
	err := r.db.WithContext(ctx).Where("asset_group_id = ?", assetGroupID).Find(&list).Error
	return list, err
}

// BatchCreate 批量创建关联
func (r *DataSourceGroupRelationRepo) BatchCreate(ctx context.Context, rels []*biz.DataSourceGroupRelation) error {
	if len(rels) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&rels).Error
}
