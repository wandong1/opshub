package dto

// RecordListRequest 执行记录列表请求
type RecordListRequest struct {
	Page      int    `form:"page" binding:"required,min=1"`
	PageSize  int    `form:"page_size" binding:"required,min=1,max=100"`
	TaskID    uint   `form:"task_id"`
	GroupID   uint   `form:"group_id"`
	ItemID    uint   `form:"item_id"`
	HostID    uint   `form:"host_id"`
	Status    string `form:"status"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
}

// RecordResponse 执行记录响应
type RecordResponse struct {
	ID                 uint   `json:"id"`
	TaskID             uint   `json:"task_id"`
	GroupID            uint   `json:"group_id"`
	ItemID             uint   `json:"item_id"`
	ItemName           string `json:"item_name"`
	HostID             uint   `json:"host_id"`
	HostName           string `json:"host_name"`
	Status             string `json:"status"`
	Output             string `json:"output"`
	ErrorMessage       string `json:"error_message"`
	Duration           float64 `json:"duration"`
	AssertionResult    string `json:"assertion_result"`
	AssertionDetails   string `json:"assertion_details"`
	ExtractedVariables string `json:"extracted_variables"`
	ExecutedAt         string `json:"executed_at"`
	CreatedAt          string `json:"created_at"`
}
