package inspection_mgmt

// RecordListRequest 执行记录列表请求
type RecordListRequest struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"pageSize" binding:"omitempty,min=1,max=1000"`
	TaskID    uint   `form:"taskId"`
	GroupID   uint   `form:"groupId"`
	ItemID    uint   `form:"itemId"`
	HostID    uint   `form:"hostId"`
	Status    string `form:"status"`
	StartTime string `form:"startTime"`
	EndTime   string `form:"endTime"`
}

// RecordResponse 执行记录响应
type RecordResponse struct {
	ID                 uint    `json:"id"`
	TaskID             uint    `json:"task_id"`
	GroupID            uint    `json:"group_id"`
	GroupName          string  `json:"group_name"`
	ItemID             uint    `json:"item_id"`
	ItemName           string  `json:"item_name"`
	HostID             uint    `json:"host_id"`
	HostName           string  `json:"host_name"`
	Status             string  `json:"status"`
	Output             string  `json:"output"`
	ErrorMessage       string  `json:"error_message"`
	Duration           float64 `json:"duration"`
	AssertionResult    string  `json:"assertion_result"`
	AssertionDetails   string  `json:"assertion_details"`
	ExtractedVariables string  `json:"extracted_variables"`
	ExecutedAt         string  `json:"executed_at"`
	CreatedAt          string  `json:"created_at"`
}
