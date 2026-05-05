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
	ID              uint           `gorm:"primarykey" json:"id" yaml:"-"`
	CreatedAt       time.Time      `json:"createdAt" yaml:"-"`
	UpdatedAt       time.Time      `json:"updatedAt" yaml:"-"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-" yaml:"-"`
	Name            string         `gorm:"size:200;not null" json:"name" yaml:"name"`
	Description     string         `gorm:"type:text" json:"description" yaml:"description"`
	AssetGroupID    uint           `gorm:"index" json:"assetGroupId" yaml:"assetgroupid"`
	RuleGroupID     uint           `gorm:"index" json:"ruleGroupId" yaml:"rulegroupid"`
	DataSourceID    uint           `gorm:"index" json:"dataSourceId" yaml:"datasourceid"`
	DataSourceIDs   string         `gorm:"type:text" json:"dataSourceIds" yaml:"datasourceids"` // JSON数组 e.g. "[1,2,3]"，优先使用
	Expr            string         `gorm:"type:text;not null" json:"expr" yaml:"expr"` // PromQL / InfluxQL（旧格式，兼容）
	QueryExpr       string         `gorm:"type:text" json:"queryExpr" yaml:"queryexpr"`     // 查询表达式（不含阈值判断，新格式）
	Conditions      string         `gorm:"type:json" json:"conditions" yaml:"conditions"`    // 阈值条件 JSON（新格式）
	EvalInterval    int            `gorm:"default:15" json:"evalInterval" yaml:"evalinterval"` // 采集频率(秒)，默认15
	Duration        string         `gorm:"size:20;default:'0s'" json:"duration" yaml:"duration"`      // 持续触发时长 e.g. "5m"
	Severity        string         `gorm:"size:20;default:'warning'" json:"severity" yaml:"severity"` // critical, warning, info
	Labels          string         `gorm:"type:text" json:"labels" yaml:"labels"`                  // JSON 额外标签 {"key":"val"}
	Annotations     string         `gorm:"type:text" json:"annotations" yaml:"annotations"`             // JSON {"title":"...","description":"..."}
	Enabled         bool           `gorm:"default:true" json:"enabled" yaml:"enabled"`
	NotifyOnResolve bool           `gorm:"default:true" json:"notifyOnResolve" yaml:"notifyonresolve"`
	LastEvalAt      *time.Time     `json:"lastEvalAt" yaml:"-"`

	// 展示用虚拟字段（不存储，查询时回填）
	RuleGroupName string `gorm:"-" json:"ruleGroupName" yaml:"-"`
}

func (AlertRule) TableName() string {
	return "alert_rules"
}

// ThresholdCondition 阈值条件
type ThresholdCondition struct {
	Operator string  `json:"operator"` // >, >=, <, <=, ==, !=
	Value    float64 `json:"value"`    // 阈值
	Logic    string  `json:"logic"`    // AND, OR (与下一个条件的关系)
}
