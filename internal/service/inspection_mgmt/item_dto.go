package inspection_mgmt

// ItemCreateRequest 创建巡检项请求
type ItemCreateRequest struct {
	Name              string `json:"name" binding:"required"`
	Description       string `json:"description"`
	GroupID           uint   `json:"groupId" binding:"required"`
	Sort              int    `json:"sort"`
	Status            string `json:"status"`
	ExecutionStrategy string `json:"executionStrategy"`
	ExecutionType     string `json:"executionType" binding:"required"`
	Command           string `json:"command"`
	ScriptType        string `json:"scriptType"`
	ScriptContent     string `json:"scriptContent"`
	ScriptFile        string `json:"scriptFile"`
	ScriptArgs        string `json:"scriptArgs"`
	PromQLQuery       string `json:"promqlQuery"`
	ProbeCategory     string `json:"probeCategory"`
	ProbeType         string `json:"probeType"`
	ProbeConfigID     uint   `json:"probeConfigId"`
	HostMatchType     string `json:"hostMatchType"`
	HostTags          string `json:"hostTags"`
	HostIDs           string `json:"hostIds"`
	AssertionType     string `json:"assertionType"`
	AssertionValue    string `json:"assertionValue"`
	VariableName      string `json:"variableName"`
	VariableRegex     string `json:"variableRegex"`
	Timeout           int    `json:"timeout"`
}

// ItemUpdateRequest 更新巡检项请求
type ItemUpdateRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	GroupID           uint   `json:"groupId"`
	Sort              int    `json:"sort"`
	Status            string `json:"status"`
	ExecutionStrategy string `json:"executionStrategy"`
	ExecutionType     string `json:"executionType"`
	Command           string `json:"command"`
	ScriptType        string `json:"scriptType"`
	ScriptContent     string `json:"scriptContent"`
	ScriptFile        string `json:"scriptFile"`
	ScriptArgs        string `json:"scriptArgs"`
	PromQLQuery       string `json:"promqlQuery"`
	ProbeCategory     string `json:"probeCategory"`
	ProbeType         string `json:"probeType"`
	ProbeConfigID     uint   `json:"probeConfigId"`
	HostMatchType     string `json:"hostMatchType"`
	HostTags          string `json:"hostTags"`
	HostIDs           string `json:"hostIds"`
	AssertionType     string `json:"assertionType"`
	AssertionValue    string `json:"assertionValue"`
	VariableName      string `json:"variableName"`
	VariableRegex     string `json:"variableRegex"`
	Timeout           int    `json:"timeout"`
}

// ItemListRequest 巡检项列表请求
type ItemListRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"pageSize" binding:"omitempty,min=1,max=1000"`
	GroupID  uint   `form:"groupId"`
	Name     string `form:"name"`
	Status   string `form:"status"`
}

// ItemResponse 巡检项响应
type ItemResponse struct {
	ID                uint   `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	GroupID           uint   `json:"groupId"`
	Sort              int    `json:"sort"`
	Status            string `json:"status"`
	ExecutionStrategy string `json:"executionStrategy"`
	ExecutionType     string `json:"executionType"`
	Command           string `json:"command"`
	ScriptType        string `json:"scriptType"`
	ScriptContent     string `json:"scriptContent"`
	ScriptFile        string `json:"scriptFile"`
	ScriptArgs        string `json:"scriptArgs"`
	PromQLQuery       string `json:"promqlQuery"`
	ProbeCategory     string `json:"probeCategory"`
	ProbeType         string `json:"probeType"`
	ProbeConfigID     uint   `json:"probeConfigId"`
	HostMatchType     string `json:"hostMatchType"`
	HostTags          string `json:"hostTags"`
	HostIDs           string `json:"hostIds"`
	AssertionType     string `json:"assertionType"`
	AssertionValue    string `json:"assertionValue"`
	VariableName      string `json:"variableName"`
	VariableRegex     string `json:"variableRegex"`
	Timeout           int    `json:"timeout"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

// TestRunRequest 测试运行请求
type TestRunRequest struct {
	GroupID uint   `json:"groupId"`
	ItemIDs []uint `json:"itemIds"`
}

// TestRunResponse 测试运行响应
type TestRunResponse struct {
	ItemID           uint                   `json:"itemId"`
	ItemName         string                 `json:"itemName"`
	HostID           uint                   `json:"hostId"`
	HostName         string                 `json:"hostName"`
	HostIp           string                 `json:"hostIp"`
	Status           string                 `json:"status"`
	Output           string                 `json:"output"`
	ErrorMessage     string                 `json:"errorMessage"`
	Duration         float64                `json:"duration"`
	AssertionResult  string                 `json:"assertionResult"`
	AssertionDetails map[string]interface{} `json:"assertionDetails"`
}
