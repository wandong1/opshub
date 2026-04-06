package inspection_mgmt

import (
	"time"

	"gorm.io/gorm"
)

// InspectionRecord 执行记录
type InspectionRecord struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	TaskID      uint           `gorm:"index" json:"task_id"`
	GroupID     uint           `gorm:"index" json:"group_id"`
	ItemID      uint           `gorm:"not null;index" json:"item_id"`
	HostID      uint           `gorm:"index" json:"host_id"`
	HostName    string         `gorm:"size:200" json:"host_name"`
	HostIP      string         `gorm:"size:50" json:"host_ip"`

	// 执行结果
	Status       string  `gorm:"size:20;not null" json:"status"` // success/failed
	Output       string  `gorm:"type:text" json:"output"`
	ErrorMessage string  `gorm:"type:text" json:"error_message"`
	Duration     float64 `json:"duration"` // 秒

	// 断言结果
	AssertionResult  string `gorm:"size:20" json:"assertion_result"`  // pass/fail/skip
	AssertionDetails string `gorm:"type:text" json:"assertion_details"` // JSON

	// 变量提取
	ExtractedVariables string `gorm:"type:text" json:"extracted_variables"` // JSON

	// 执行时间
	ExecutedAt time.Time `gorm:"index" json:"executed_at"`
}

func (InspectionRecord) TableName() string {
	return "inspection_records"
}
