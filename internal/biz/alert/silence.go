package alert

import (
	"time"

	"gorm.io/gorm"
)

// AlertSilenceRule 告警屏蔽规则
type AlertSilenceRule struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 屏蔽维度（三元组）
	Severity string `gorm:"size:20;not null;index" json:"severity"` // 告警等级
	RuleName string `gorm:"size:200;not null;index" json:"ruleName"` // 规则名称
	Labels   string `gorm:"type:text" json:"labels"`                 // JSON 标签（用户可编辑，移除部分标签）

	// 屏蔽时效
	Type string `gorm:"size:20;not null" json:"type"` // fixed(固定时长) / periodic(周期性)

	// 固定时长
	Duration     string     `gorm:"size:20" json:"duration"`         // "2h", "1d"
	SilenceUntil *time.Time `gorm:"index" json:"silenceUntil"`       // 屏蔽截止时间

	// 周期性屏蔽
	TimeRanges string `gorm:"type:text" json:"timeRanges"` // JSON: [{"weekdays":[1,2,3,4,5],"start":"08:00","end":"18:00"}]

	Reason    string `gorm:"size:1000" json:"reason"` // 屏蔽原因
	CreatedBy uint   `gorm:"index" json:"createdBy"`  // 创建人
	Enabled   bool   `gorm:"default:true" json:"enabled"`
}

func (AlertSilenceRule) TableName() string {
	return "alert_silence_rules"
}
