package dto

// ItemCreateRequest 创建巡检项请求
type ItemCreateRequest struct {
	Name              string `json:"name" binding:"required"`
	Description       string `json:"description"`
	GroupID           uint   `json:"group_id" binding:"required"`
	Sort              int    `json:"sort"`
	Status            string `json:"status"`
	ExecutionStrategy string `json:"execution_strategy"`
	ExecutionType     string `json:"execution_type" binding:"required"`
	Command           string `json:"command"`
	ScriptType        string `json:"script_type"`
	ScriptContent     string `json:"script_content"`
	ScriptFile        string `json:"script_file"`
	PromQLQuery       string `json:"promql_query"`
	HostMatchType     string `json:"host_match_type"`
	HostTags          string `json:"host_tags"`
	HostIDs           string `json:"host_ids"`
	AssertionType     string `json:"assertion_type"`
	AssertionValue    string `json:"assertion_value"`
	VariableName      string `json:"variable_name"`
	VariableRegex     string `json:"variable_regex"`
	Timeout           int    `json:"timeout"`
}

// ItemUpdateRequest 更新巡检项请求
type ItemUpdateRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	GroupID           uint   `json:"group_id"`
	Sort              int    `json:"sort"`
	Status            string `json:"status"`
	ExecutionStrategy string `json:"execution_strategy"`
	ExecutionType     string `json:"execution_type"`
	Command           string `json:"command"`
	ScriptType        string `json:"script_type"`
	ScriptContent     string `json:"script_content"`
	ScriptFile        string `json:"script_file"`
	PromQLQuery       string `json:"promql_query"`
	HostMatchType     string `json:"host_match_type"`
	HostTags          string `json:"host_tags"`
	HostIDs           string `json:"host_ids"`
	AssertionType     string `json:"assertion_type"`
	AssertionValue    string `json:"assertion_value"`
	VariableName      string `json:"variable_name"`
	VariableRegex     string `json:"variable_regex"`
	Timeout           int    `json:"timeout"`
}

// ItemListRequest 巡检项列表请求
type ItemListRequest struct {
	Page     int    `form:"page" binding:"required,min=1"`
	PageSize int    `form:"page_size" binding:"required,min=1,max=100"`
	GroupID  uint   `form:"group_id"`
	Name     string `form:"name"`
	Status   string `form:"status"`
}

// ItemResponse 巡检项响应
type ItemResponse struct {
	ID                uint   `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	GroupID           uint   `json:"group_id"`
	Sort              int    `json:"sort"`
	Status            string `json:"status"`
	ExecutionStrategy string `json:"execution_strategy"`
	ExecutionType     string `json:"execution_type"`
	Command           string `json:"command"`
	ScriptType        string `json:"script_type"`
	ScriptContent     string `json:"script_content"`
	ScriptFile        string `json:"script_file"`
	PromQLQuery       string `json:"promql_query"`
	HostMatchType     string `json:"host_match_type"`
	HostTags          string `json:"host_tags"`
	HostIDs           string `json:"host_ids"`
	AssertionType     string `json:"assertion_type"`
	AssertionValue    string `json:"assertion_value"`
	VariableName      string `json:"variable_name"`
	VariableRegex     string `json:"variable_regex"`
	Timeout           int    `json:"timeout"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// TestRunRequest 测试运行请求
type TestRunRequest struct {
	ItemIDs []uint `json:"item_ids" binding:"required"`
}

// TestRunResponse 测试运行响应
type TestRunResponse struct {
	ItemID           uint                   `json:"item_id"`
	ItemName         string                 `json:"item_name"`
	HostID           uint                   `json:"host_id"`
	HostName         string                 `json:"host_name"`
	Status           string                 `json:"status"`
	Output           string                 `json:"output"`
	ErrorMessage     string                 `json:"error_message"`
	Duration         float64                `json:"duration"`
	AssertionResult  string                 `json:"assertion_result"`
	AssertionDetails map[string]interface{} `json:"assertion_details"`
}
