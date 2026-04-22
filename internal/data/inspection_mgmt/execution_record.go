package inspection_mgmt

import (
	"time"

	"gorm.io/gorm"
)

// InspectionExecutionRecord 巡检执行记录主表
type InspectionExecutionRecord struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 任务信息
	TaskID   uint   `gorm:"not null;index:idx_task_id" json:"taskId"`
	TaskName string `gorm:"size:200;not null" json:"taskName"`

	// 执行统计
	TotalItems     int `gorm:"not null;default:0" json:"totalItems"`
	TotalHosts     int `gorm:"not null;default:0" json:"totalHosts"`
	TotalExecutions int `gorm:"not null;default:0" json:"totalExecutions"`
	SuccessCount   int `gorm:"not null;default:0" json:"successCount"`
	FailedCount    int `gorm:"not null;default:0" json:"failedCount"`

	// 断言统计
	AssertionPassCount int `gorm:"not null;default:0" json:"assertionPassCount"`
	AssertionFailCount int `gorm:"not null;default:0" json:"assertionFailCount"`
	AssertionSkipCount int `gorm:"not null;default:0" json:"assertionSkipCount"`

	// 执行信息
	Status      string     `gorm:"size:20;not null;default:'running';index:idx_status" json:"status"` // running/success/failed/partial
	Duration    float64    `gorm:"not null;default:0" json:"duration"`                                 // 总执行时长(秒)
	StartedAt   time.Time  `gorm:"not null;index:idx_started_at" json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// 触发方式
	TriggerType string `gorm:"size:20;default:'scheduled'" json:"triggerType"` // scheduled/manual

	// 配置快照
	GroupIDs   string `gorm:"type:json" json:"groupIds"`   // JSON数组
	GroupNames string `gorm:"type:json" json:"groupNames"` // JSON数组
}

func (InspectionExecutionRecord) TableName() string {
	return "inspection_execution_records"
}

// InspectionExecutionDetail 巡检执行明细表
type InspectionExecutionDetail struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`

	// 关联信息
	ExecutionID uint   `gorm:"not null;index:idx_execution_id" json:"executionId"`
	GroupID     uint   `gorm:"not null;index:idx_group_id" json:"groupId"`
	GroupName   string `gorm:"size:200;not null" json:"groupName"`
	ItemID      uint   `gorm:"not null;index:idx_item_id" json:"itemId"`
	ItemName    string `gorm:"size:200;not null" json:"itemName"`
	HostID      uint   `gorm:"not null;index:idx_host_id" json:"hostId"`
	HostName    string `gorm:"size:200;not null" json:"hostName"`
	HostIP      string `gorm:"size:50;not null" json:"hostIp"`

	// 执行配置信息
	BusinessGroup   string `gorm:"size:200" json:"businessGroup"`       // 业务分组名称
	ExecutionType   string `gorm:"size:50" json:"executionType"`        // command/script/probe/promql
	ExecutionMode   string `gorm:"size:50" json:"executionMode"`        // auto/agent/ssh
	Command         string `gorm:"type:text" json:"command"`            // 执行的命令
	ScriptType      string `gorm:"size:50" json:"scriptType"`           // shell/python/etc
	ScriptContent   string `gorm:"type:text" json:"scriptContent"`      // 脚本内容
	AssertionType   string `gorm:"size:50" json:"assertionType"`        // 断言类型
	AssertionValue  string `gorm:"type:text" json:"assertionValue"`     // 断言值

	// 执行结果
	Status       string  `gorm:"size:20;not null;index:idx_status" json:"status"` // success/failed
	Output       string  `gorm:"type:text" json:"output"`
	ErrorMessage string  `gorm:"type:text" json:"errorMessage"`
	Duration     float64 `gorm:"not null;default:0" json:"duration"` // 执行时长(秒)

	// 断言结果
	AssertionResult  string `gorm:"size:20" json:"assertionResult"`       // pass/fail/skip
	AssertionDetails string `gorm:"type:text" json:"assertionDetails"`    // JSON格式
	ExtractedVariables string `gorm:"type:text" json:"extractedVariables"` // JSON格式

	// 巡检级别和风险等级（快照）
	InspectionLevel string `gorm:"size:20" json:"inspectionLevel"` // high/medium/low
	RiskLevel       string `gorm:"size:20" json:"riskLevel"`       // high/medium/low

	ExecutedAt time.Time `gorm:"not null" json:"executedAt"`
}

func (InspectionExecutionDetail) TableName() string {
	return "inspection_execution_details"
}
