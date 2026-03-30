package alert

import (
	"time"

	"gorm.io/gorm"
)

// AlertNotifyChannel 告警通知通道
type AlertNotifyChannel struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	Name            string         `gorm:"size:100;not null" json:"name"`
	Type            string         `gorm:"size:30;not null" json:"type"` // wechat_work, dingtalk, sms, phone, ai_agent
	Config          string         `gorm:"type:text" json:"config"`         // JSON 通道配置
	AlertTemplate   string         `gorm:"type:text" json:"alertTemplate"`  // 告警通知模板
	ResolveTemplate string         `gorm:"type:text" json:"resolveTemplate"` // 恢复通知模板
	Enabled         bool           `gorm:"default:true" json:"enabled"`
	AIHookEnabled   bool           `gorm:"default:false" json:"aiHookEnabled"` // AI智能体总开关
}

func (AlertNotifyChannel) TableName() string {
	return "alert_notify_channels"
}
