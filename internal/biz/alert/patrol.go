package alert

import (
	"time"

	"gorm.io/gorm"
)

// AlertSubscriptionRulePatrol 告警订阅规则巡检配置
type AlertSubscriptionRulePatrol struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	SubscriptionRuleID uint `gorm:"uniqueIndex;not null" json:"subscriptionRuleId"`

	// 巡检配置
	Enabled        bool   `gorm:"default:false" json:"enabled"`
	PatrolMode     string `gorm:"size:20;default:'interval'" json:"patrolMode"` // interval/fixed
	PatrolInterval int    `gorm:"default:3600" json:"patrolInterval"`           // 秒
	PatrolTimes    string `gorm:"type:text" json:"patrolTimes"`                 // JSON: ["09:00","18:00"]

	// 巡检范围
	IncludeResolved    bool `gorm:"default:false" json:"includeResolved"`
	TimeRange          int  `gorm:"default:0" json:"timeRange"`               // 秒，0=所有
	MaxAlertsPerReport int  `gorm:"default:100" json:"maxAlertsPerReport"`

	// 推送配置
	SendMode string `gorm:"size:20;default:'always'" json:"sendMode"` // always/only_firing

	// 报告样式
	ReportStyle string `gorm:"size:20;default:'detailed'" json:"reportStyle"` // detailed/summary
	GroupBy     string `gorm:"size:50;default:'severity'" json:"groupBy"`     // severity/ruleName/assetGroup

	// 巡检状态
	LastPatrolAt *time.Time `json:"lastPatrolAt"`
	NextPatrolAt *time.Time `gorm:"index" json:"nextPatrolAt"`
}

func (AlertSubscriptionRulePatrol) TableName() string {
	return "alert_subscription_rule_patrol"
}

// AlertPatrolReport 巡检报告
type AlertPatrolReport struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`

	SubscriptionID     uint `gorm:"index;not null" json:"subscriptionId"`
	SubscriptionRuleID uint `gorm:"index;not null" json:"subscriptionRuleId"`
	PatrolConfigID     uint `gorm:"index;not null" json:"patrolConfigId"`

	// 巡检结果
	FiringCount   int `gorm:"default:0" json:"firingCount"`
	ResolvedCount int `gorm:"default:0" json:"resolvedCount"`
	CriticalCount int `gorm:"default:0" json:"criticalCount"` // 紧急(P1)
	MajorCount    int `gorm:"default:0" json:"majorCount"`    // 严重(P2)
	MinorCount    int `gorm:"default:0" json:"minorCount"`    // 一般(P3)
	WarningCount  int `gorm:"default:0" json:"warningCount"`  // 提示(P4)

	// 告警详情
	AlertIDs     string `gorm:"type:text" json:"alertIds"`     // JSON: [1,2,3]
	AlertSummary string `gorm:"type:text" json:"alertSummary"` // JSON

	// 推送结果
	Sent      bool       `gorm:"default:false" json:"sent"`
	SentAt    *time.Time `json:"sentAt"`
	Channels  string     `gorm:"type:text" json:"channels"`  // JSON
	Users     string     `gorm:"type:text" json:"users"`     // JSON
	SendError string     `gorm:"type:text" json:"sendError"`
}

func (AlertPatrolReport) TableName() string {
	return "alert_patrol_report"
}

// AlertSummaryItem 告警摘要项
type AlertSummaryItem struct {
	RuleName      string    `json:"ruleName"`
	Severity      string    `json:"severity"`
	Count         int       `json:"count"`
	OldestFiredAt time.Time `json:"oldestFiredAt"` // 最早触发时间
	LatestValue   float64   `json:"latestValue"`   // 最新值
}
