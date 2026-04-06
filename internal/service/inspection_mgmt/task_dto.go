package inspection_mgmt

// TaskCreateRequest 创建定时任务请求
type TaskCreateRequest struct {
	Name            string `json:"name" binding:"required"`
	Description     string `json:"description"`
	TaskType        string `json:"task_type" binding:"required,oneof=probe inspection"`
	CronExpr        string `json:"cron_expr" binding:"required"`
	Enabled         bool   `json:"enabled"`
	GroupIDs        string `json:"group_ids"`
	ItemIDs         string `json:"item_ids"`
	PushgatewayID   uint   `json:"pushgateway_id"`
	Concurrency     int    `json:"concurrency"`
	Owner           string `json:"owner"`
	// 需求一新增字段
	ExecutionMode   string `json:"execution_mode"`   // 执行方式覆盖
	AgentHostIDs    string `json:"agent_host_ids"`   // Agent 主机 ID 列表（JSON 数组）
	BusinessGroupID uint   `json:"business_group_id"` // 业务分组 ID
	CustomVariables string `json:"custom_variables"` // 自定义变量（JSON 对象，拨测任务）
}

// TaskUpdateRequest 更新定时任务请求
type TaskUpdateRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	TaskType        string `json:"task_type"`
	CronExpr        string `json:"cron_expr"`
	Enabled         bool   `json:"enabled"`
	GroupIDs        string `json:"group_ids"`
	ItemIDs         string `json:"item_ids"`
	PushgatewayID   uint   `json:"pushgateway_id"`
	Concurrency     int    `json:"concurrency"`
	Owner           string `json:"owner"`
	// 需求一新增字段
	ExecutionMode   string `json:"execution_mode"`
	AgentHostIDs    string `json:"agent_host_ids"`
	BusinessGroupID uint   `json:"business_group_id"`
	CustomVariables string `json:"custom_variables"`
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
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	TaskType        string `json:"task_type"`
	CronExpr        string `json:"cron_expr"`
	Status          string `json:"status"`
	Enabled         bool   `json:"enabled"`
	GroupIDs        string `json:"group_ids"`
	ItemIDs         string `json:"item_ids"`
	PushgatewayID   uint   `json:"pushgateway_id"`
	Concurrency     int    `json:"concurrency"`
	Owner           string `json:"owner"`
	ExecutionMode   string `json:"execution_mode"`
	AgentHostIDs    string `json:"agent_host_ids"`
	BusinessGroupID uint   `json:"business_group_id"`
	CustomVariables string `json:"custom_variables"`
	LastRunAt       string `json:"last_run_at"`
	LastRunStatus   string `json:"last_run_status"`
	NextRunAt       string `json:"next_run_at"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}
