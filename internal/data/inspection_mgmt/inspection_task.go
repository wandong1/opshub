package inspection_mgmt

import (
	"time"

	"gorm.io/gorm"
)

// InspectionTask 定时任务
type InspectionTask struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	TaskType    string         `gorm:"size:20;default:'probe'" json:"task_type"` // probe/inspection
	CronExpr    string         `gorm:"size:50;not null" json:"cron_expr"`
	Status      string         `gorm:"size:20;default:'pending'" json:"status"` // pending/running/success/failed
	Enabled     bool           `gorm:"default:true" json:"enabled"`

	// 执行范围
	GroupIDs string `gorm:"type:text" json:"group_ids"` // JSON 数组 [1,2,3] - 巡检组ID或资产组ID
	ItemIDs  string `gorm:"type:text" json:"item_ids"`  // JSON 数组 [1,2,3] - 巡检项ID或拨测配置ID

	// Pushgateway
	PushgatewayID uint `gorm:"default:0" json:"pushgateway_id"`

	// 拨测任务专用字段
	Concurrency int `gorm:"default:5" json:"concurrency"` // 并发数

	// 负责人
	Owner string `gorm:"size:100" json:"owner"`

	// 任务调度级别的执行方式覆盖
	// 拨测任务：local/agent，空表示沿用各拨测配置原有设置
	// 巡检任务：auto/agent/ssh，空表示沿用各巡检组原有设置
	ExecutionMode string `gorm:"size:20;default:''" json:"execution_mode"`

	// Agent 主机 ID 列表 ✅ 移除 default:''
	AgentHostIDs string `gorm:"type:text" json:"agent_host_ids"`

	// 业务分组
	BusinessGroupID uint `gorm:"default:0" json:"business_group_id"`

	// 自定义变量 ✅ 移除 default:''
	CustomVariables string `gorm:"type:text" json:"custom_variables"`

	// 执行记录
	LastRunAt     *time.Time `json:"last_run_at"`
	LastRunStatus string     `gorm:"size:20" json:"last_run_status"` // success/failed
	NextRunAt     *time.Time `json:"next_run_at"`
}

func (InspectionTask) TableName() string {
	return "inspection_tasks"
}
