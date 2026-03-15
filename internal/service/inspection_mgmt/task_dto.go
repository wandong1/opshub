package inspection_mgmt

// TaskCreateRequest 创建定时任务请求
type TaskCreateRequest struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	TaskType      string `json:"task_type" binding:"required,oneof=probe inspection"`
	CronExpr      string `json:"cron_expr" binding:"required"`
	Enabled       bool   `json:"enabled"`
	GroupIDs      string `json:"group_ids"`
	ItemIDs       string `json:"item_ids"`
	PushgatewayID uint   `json:"pushgateway_id"`
	Concurrency   int    `json:"concurrency"`
}

// TaskUpdateRequest 更新定时任务请求
type TaskUpdateRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	TaskType      string `json:"task_type"`
	CronExpr      string `json:"cron_expr"`
	Enabled       bool   `json:"enabled"`
	GroupIDs      string `json:"group_ids"`
	ItemIDs       string `json:"item_ids"`
	PushgatewayID uint   `json:"pushgateway_id"`
	Concurrency   int    `json:"concurrency"`
}

// TaskListRequest 定时任务列表请求
type TaskListRequest struct {
	Page     int    `form:"page" binding:"required,min=1"`
	PageSize int    `form:"page_size" binding:"required,min=1,max=100"`
	Name     string `form:"name"`
	Enabled  *bool  `form:"enabled"`
}

// TaskResponse 定时任务响应
type TaskResponse struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	TaskType      string `json:"task_type"`
	CronExpr      string `json:"cron_expr"`
	Status        string `json:"status"`
	Enabled       bool   `json:"enabled"`
	GroupIDs      string `json:"group_ids"`
	ItemIDs       string `json:"item_ids"`
	PushgatewayID uint   `json:"pushgateway_id"`
	Concurrency   int    `json:"concurrency"`
	LastRunAt     string `json:"last_run_at"`
	LastRunStatus string `json:"last_run_status"`
	NextRunAt     string `json:"next_run_at"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}
