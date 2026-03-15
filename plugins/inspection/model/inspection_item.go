package model

import (
	"time"

	"gorm.io/gorm"
)

// InspectionItem 巡检项
type InspectionItem struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	GroupID     uint           `gorm:"not null;index" json:"group_id"`
	Sort        int            `gorm:"default:0" json:"sort"`
	Status      string         `gorm:"size:20;default:'enabled'" json:"status"` // enabled/disabled

	// 执行策略
	ExecutionStrategy string `gorm:"size:20;default:'concurrent'" json:"execution_strategy"` // concurrent/sequential

	// 执行类型
	ExecutionType string `gorm:"size:20;not null" json:"execution_type"` // command/script/promql

	// 命令执行
	Command string `gorm:"type:text" json:"command"`

	// 脚本执行
	ScriptType    string `gorm:"size:20" json:"script_type"`    // shell/python
	ScriptContent string `gorm:"type:text" json:"script_content"`
	ScriptFile    string `gorm:"size:200" json:"script_file"`

	// PromQL 查询
	PromQLQuery string `gorm:"type:text" json:"promql_query"`

	// 主机匹配
	HostMatchType string `gorm:"size:20;default:'tag'" json:"host_match_type"` // tag/id
	HostTags      string `gorm:"type:text" json:"host_tags"`                   // 逗号分隔
	HostIDs       string `gorm:"type:text" json:"host_ids"`                    // JSON 数组 [1,2,3]

	// 断言规则
	AssertionType  string `gorm:"size:30" json:"assertion_type"`  // gt/gte/lt/lte/eq/contains/not_contains/regex/not_regex
	AssertionValue string `gorm:"size:500" json:"assertion_value"`

	// 变量提取
	VariableName  string `gorm:"size:50" json:"variable_name"`
	VariableRegex string `gorm:"size:500" json:"variable_regex"`

	// 超时设置
	Timeout int `gorm:"default:60" json:"timeout"` // 秒
}

func (InspectionItem) TableName() string {
	return "inspection_items"
}
