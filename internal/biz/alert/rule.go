package alert

import (
	"time"

	"gorm.io/gorm"
)

// AlertRuleGroup 告警规则分类（业务分组下的二级分类）
type AlertRuleGroup struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Name         string         `gorm:"size:100;not null" json:"name"`
	AssetGroupID uint           `gorm:"index;not null" json:"assetGroupId"` // 复用业务分组
	Description  string         `gorm:"size:500" json:"description"`
	Sort         int            `gorm:"default:0" json:"sort"`
}

func (AlertRuleGroup) TableName() string {
	return "alert_rule_groups"
}

// AlertRule 告警规则
type AlertRule struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	Name            string         `gorm:"size:200;not null" json:"name"`
	Description     string         `gorm:"type:text" json:"description"`
	AssetGroupID    uint           `gorm:"index" json:"assetGroupId"`
	RuleGroupID     uint           `gorm:"index" json:"ruleGroupId"`
	DataSourceID    uint           `gorm:"index" json:"dataSourceId"`
	DataSourceIDs   string         `gorm:"type:text" json:"dataSourceIds"` // JSON数组 e.g. "[1,2,3]"，优先使用
	Expr            string         `gorm:"type:text;not null" json:"expr"`            // PromQL / InfluxQL
	EvalInterval    int            `gorm:"default:15" json:"evalInterval"`            // 采集频率(秒)，默认15
	Duration        string         `gorm:"size:20;default:'0s'" json:"duration"`      // 持续触发时长 e.g. "5m"
	Severity        string         `gorm:"size:20;default:'warning'" json:"severity"` // critical, warning, info
	Labels          string         `gorm:"type:text" json:"labels"`                  // JSON 额外标签 {"key":"val"}
	Annotations     string         `gorm:"type:text" json:"annotations"`             // JSON {"title":"...","description":"..."}
	Enabled         bool           `gorm:"default:true" json:"enabled"`
	NotifyOnResolve bool           `gorm:"default:true" json:"notifyOnResolve"`
	LastEvalAt      *time.Time     `json:"lastEvalAt"`

	// 展示用虚拟字段（不存储，查询时回填）
	RuleGroupName string `gorm:"-" json:"ruleGroupName"`
}

func (AlertRule) TableName() string {
	return "alert_rules"
}
