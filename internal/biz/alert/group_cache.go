package alert

import "time"

// AlertGroupCache 分组告警缓存
type AlertGroupCache struct {
	ID             uint       `gorm:"primarykey" json:"id"`
	GroupRuleID    uint       `gorm:"index;not null" json:"groupRuleId"`
	SubscriptionID uint       `gorm:"index;not null" json:"subscriptionId"`
	GroupKey       string     `gorm:"size:255;not null" json:"groupKey"`
	Alerts         string     `gorm:"type:text" json:"alerts"` // JSON: [123, 124, 125]
	FirstAlertAt   time.Time  `gorm:"not null" json:"firstAlertAt"`
	LastAlertAt    time.Time  `gorm:"not null" json:"lastAlertAt"`
	AlertCount     int        `gorm:"default:1" json:"alertCount"`
	Sent           bool       `gorm:"default:false" json:"sent"`
	SentAt         *time.Time `json:"sentAt"`
}

func (AlertGroupCache) TableName() string {
	return "alert_group_cache"
}
