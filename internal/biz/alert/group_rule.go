package alert

import (
	"time"

	"gorm.io/gorm"
)

// AlertGroupRule 告警分组规则
type AlertGroupRule struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	SubscriptionID uint           `gorm:"index;not null" json:"subscriptionId"`
	Name           string         `gorm:"size:100;not null" json:"name"`
	Enabled        bool           `gorm:"default:true" json:"enabled"`
	GroupBy        string         `gorm:"type:text" json:"groupBy"`        // JSON: ["severity", "ruleName"]
	GroupWait      int            `gorm:"default:30" json:"groupWait"`     // 分组等待时间(秒)
	GroupInterval  int            `gorm:"default:300" json:"groupInterval"` // 分组发送间隔(秒)
	MaxGroupSize   int            `gorm:"default:20" json:"maxGroupSize"`  // 单组最大告警数
}

func (AlertGroupRule) TableName() string {
	return "alert_group_rules"
}
