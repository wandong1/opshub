package model

import (
	"time"

	"gorm.io/gorm"
)

// InspectionTask 定时任务
type InspectionTask struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	CronExpr    string         `gorm:"size:50;not null" json:"cron_expr"`
	Status      string         `gorm:"size:20;default:'pending'" json:"status"` // pending/running/success/failed
	Enabled     bool           `gorm:"default:true" json:"enabled"`

	// 执行范围
	GroupIDs string `gorm:"type:text" json:"group_ids"` // JSON 数组 [1,2,3]
	ItemIDs  string `gorm:"type:text" json:"item_ids"`  // JSON 数组 [1,2,3]

	// Pushgateway
	PushgatewayID uint `gorm:"default:0" json:"pushgateway_id"`

	// 执行记录
	LastRunAt     *time.Time `json:"last_run_at"`
	LastRunStatus string     `gorm:"size:20" json:"last_run_status"` // success/failed
	NextRunAt     *time.Time `json:"next_run_at"`
}

func (InspectionTask) TableName() string {
	return "inspection_tasks"
}
