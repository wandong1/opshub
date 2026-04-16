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
	"time"

	"gorm.io/gorm"
)

// DataSourceAgentRelation 告警数据源与Agent主机的关联关系
type DataSourceAgentRelation struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	DataSourceID  uint           `gorm:"index;not null" json:"data_source_id"`
	AgentHostID   uint           `gorm:"index;not null" json:"agent_host_id"`
	Priority      int            `gorm:"default:0" json:"priority"` // 0-10，越小优先级越高
}

func (DataSourceAgentRelation) TableName() string {
	return "alert_datasource_agent_relations"
}

// DataSourceAgentRelationRepo 数据源Agent关联仓储接口
type DataSourceAgentRelationRepo interface {
	Create(ctx context.Context, rel *DataSourceAgentRelation) error
	Delete(ctx context.Context, id uint) error
	DeleteByDataSourceID(ctx context.Context, dsID uint) error
	ListByDataSourceID(ctx context.Context, dsID uint) ([]*DataSourceAgentRelation, error)
}
