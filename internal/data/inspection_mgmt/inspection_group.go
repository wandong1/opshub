package inspection_mgmt

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
	Name        string         `gorm:"size:100;not null;uniqueIndex:idx_inspection_groups_name_deleted" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	Status      string         `gorm:"size:20;default:'enabled'" json:"status"` // enabled/disabled
	Sort        int            `gorm:"default:0" json:"sort"`

	// Prometheus 配置
	PrometheusURL      string `gorm:"size:200" json:"prometheus_url"`
	PrometheusUsername string `gorm:"size:100" json:"prometheus_username"`
	PrometheusPassword string `gorm:"size:200" json:"prometheus_password"` // 加密存储

	// 执行方式
	ExecutionMode string `gorm:"size:20;default:'auto'" json:"execution_mode"` // ssh/agent/auto

	// 执行策略
	ExecutionStrategy string `gorm:"size:20;default:'concurrent'" json:"execution_strategy"` // concurrent/sequential
	Concurrency       int    `gorm:"default:50" json:"concurrency"`                          // 并发数量

	// 关联分组（JSON 数组）
	GroupIDs string `gorm:"type:text" json:"group_ids"` // [1,2,3]

	// 自定义变量（JSON 对象）
	CustomVariables string `gorm:"type:text" json:"custom_variables"` // {"key1":"value1","key2":"value2"}

	// 自定义标签（JSON 数组，用于 metric 标签）
	Labels string `gorm:"type:text" json:"labels"` // ["env:prod","team:ops"]
}

func (InspectionGroup) TableName() string {
	return "inspection_groups"
}
