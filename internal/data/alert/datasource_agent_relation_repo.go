// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package alert

import (
	"context"

	alertbiz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	"gorm.io/gorm"
)

type DataSourceAgentRelationRepo struct {
	db *gorm.DB
}

func NewDataSourceAgentRelationRepo(db *gorm.DB) alertbiz.DataSourceAgentRelationRepo {
	return &DataSourceAgentRelationRepo{db: db}
}

func (r *DataSourceAgentRelationRepo) Create(ctx context.Context, rel *alertbiz.DataSourceAgentRelation) error {
	return r.db.WithContext(ctx).Create(rel).Error
}

func (r *DataSourceAgentRelationRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&alertbiz.DataSourceAgentRelation{}, id).Error
}

func (r *DataSourceAgentRelationRepo) DeleteByDataSourceID(ctx context.Context, dsID uint) error {
	return r.db.WithContext(ctx).Where("data_source_id = ?", dsID).Delete(&alertbiz.DataSourceAgentRelation{}).Error
}

func (r *DataSourceAgentRelationRepo) ListByDataSourceID(ctx context.Context, dsID uint) ([]*alertbiz.DataSourceAgentRelation, error) {
	var rels []*alertbiz.DataSourceAgentRelation
	err := r.db.WithContext(ctx).
		Where("data_source_id = ?", dsID).
		Order("priority asc, id asc").
		Find(&rels).Error
	return rels, err
}
