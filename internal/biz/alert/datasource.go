package alert

import (
	"time"

	"gorm.io/gorm"
)

// AlertDataSource 告警数据源
type AlertDataSource struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Type        string         `gorm:"size:30;not null" json:"type"` // prometheus, victoriametrics, influxdb
	URL         string         `gorm:"size:500;not null" json:"url"`
	Username    string         `gorm:"size:100" json:"username"`
	Password    string         `gorm:"size:200" json:"password"`
	Token       string         `gorm:"size:500" json:"token"`
	Description string         `gorm:"size:500" json:"description"`
	Status      int            `gorm:"default:1" json:"status"` // 1=启用 0=禁用
}

func (AlertDataSource) TableName() string {
	return "alert_datasources"
}
