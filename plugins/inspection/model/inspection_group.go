package model

import (
	"time"

	"gorm.io/gorm"
)

// InspectionGroup 巡检组
type InspectionGroup struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	Status      string         `gorm:"size:20;default:'enabled'" json:"status"` // enabled/disabled
	Sort        int            `gorm:"default:0" json:"sort"`

	// 数据源配置（单个数据源）
	DataSourceID uint `gorm:"default:0;index" json:"datasource_id"` // 关联的数据源ID

	// 执行方式
	ExecutionMode string `gorm:"size:20;default:'auto'" json:"execution_mode"` // ssh/agent/auto

	// 关联分组（JSON 数组）
	GroupIDs string `gorm:"type:text" json:"group_ids"` // [1,2,3]
}

func (InspectionGroup) TableName() string {
	return "inspection_groups"
}
