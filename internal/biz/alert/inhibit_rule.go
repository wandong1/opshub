package alert

import (
	"time"

	"gorm.io/gorm"
)

// AlertInhibitRule 告警抑制规则
type AlertInhibitRule struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	SubscriptionID uint           `gorm:"index;not null" json:"subscriptionId"`
	Name           string         `gorm:"size:100;not null" json:"name"`
	Enabled        bool           `gorm:"default:true" json:"enabled"`
	SourceMatchers string         `gorm:"type:text" json:"sourceMatchers"` // JSON: {"severity": "critical", "ruleName": "节点宕机"}
	TargetMatchers string         `gorm:"type:text" json:"targetMatchers"` // JSON: {"severity": "warning", "ruleName": "服务不可用"}
	EqualLabels    string         `gorm:"type:text" json:"equalLabels"`    // JSON: ["instance", "cluster"]
}

func (AlertInhibitRule) TableName() string {
	return "alert_inhibit_rules"
}
