package alert

import "time"

// AlertEvent 告警事件（活跃告警与历史告警统一表）
type AlertEvent struct {
	ID            uint       `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	AlertRuleID   uint       `gorm:"index;not null" json:"alertRuleId"`
	RuleName      string     `gorm:"size:200" json:"ruleName"`       // 冗余，避免联表
	AssetGroupID  uint       `gorm:"index" json:"assetGroupId"`
	Fingerprint   string     `gorm:"size:64;index" json:"fingerprint"` // sha256(ruleID+sortedLabels)
	Severity      string     `gorm:"size:20" json:"severity"`
	Status        string     `gorm:"size:20;default:'firing'" json:"status"` // firing, resolved
	Labels        string     `gorm:"type:text" json:"labels"`
	Annotations   string     `gorm:"type:text" json:"annotations"`
	Value         float64    `json:"value"`                         // 触发时当前值
	ResolveValue  *float64   `json:"resolveValue"`                  // 恢复时当前值
	FiredAt       time.Time  `gorm:"index" json:"firedAt"`
	ResolvedAt    *time.Time `json:"resolvedAt"`
	ResolveType   string     `gorm:"size:20" json:"resolveType"`    // auto, manual
	// 屏蔽
	Silenced      bool       `gorm:"default:false" json:"silenced"`
	SilenceUntil  *time.Time `json:"silenceUntil"`
	SilenceReason string     `gorm:"size:500" json:"silenceReason"`
	// 手动处理
	ManualHandled bool       `gorm:"default:false" json:"manualHandled"`
	HandledBy     *uint      `json:"handledBy"`  // sys_user.id
	HandledAt     *time.Time `json:"handledAt"`
	HandledNote   string     `gorm:"size:1000" json:"handledNote"`

	// 展示用虚拟字段（不存储，查询时回填）
	AssetGroupName string `gorm:"-" json:"assetGroupName"`
	RuleGroupName  string `gorm:"-" json:"ruleGroupName"`
}

func (AlertEvent) TableName() string {
	return "alert_events"
}
