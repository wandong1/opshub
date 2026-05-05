package alert

import (
	"time"

	"gorm.io/gorm"
)

// AlertDedupRule 告警去重规则
type AlertDedupRule struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	SubscriptionID uint           `gorm:"index;not null" json:"subscriptionId"`
	Name           string         `gorm:"size:100;not null" json:"name"`
	Enabled        bool           `gorm:"default:true" json:"enabled"`
	FingerprintKeys string        `gorm:"type:text" json:"fingerprintKeys"` // JSON: ["severity", "ruleName", "instance"]
	DedupWindow    int            `gorm:"default:600" json:"dedupWindow"`    // 去重时间窗口(秒)
}

func (AlertDedupRule) TableName() string {
	return "alert_dedup_rules"
}
