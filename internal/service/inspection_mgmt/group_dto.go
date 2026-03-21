package inspection_mgmt

// GroupCreateRequest 创建巡检组请求
type GroupCreateRequest struct {
	Name              string `json:"name" binding:"required"`
	Description       string `json:"description"`
	Status            string `json:"status"`
	Sort              int    `json:"sort"`
	PrometheusURL     string `json:"prometheusUrl"`
	PrometheusUsername string `json:"prometheusUsername"`
	PrometheusPassword string `json:"prometheusPassword"`
	ExecutionMode     string `json:"executionMode"`
	ExecutionStrategy string `json:"executionStrategy"`
	Concurrency       int    `json:"concurrency"`
	GroupIDs          string `json:"groupIds"`
	CustomVariables   string `json:"customVariables"` // JSON 对象格式的自定义变量
	Labels            string `json:"labels"`          // JSON 数组格式的自定义标签 ["env:prod","team:ops"]
}

// GroupUpdateRequest 更新巡检组请求
type GroupUpdateRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	Status            string `json:"status"`
	Sort              int    `json:"sort"`
	PrometheusURL     string `json:"prometheusUrl"`
	PrometheusUsername string `json:"prometheusUsername"`
	PrometheusPassword string `json:"prometheusPassword"`
	ExecutionMode     string `json:"executionMode"`
	ExecutionStrategy string `json:"executionStrategy"`
	Concurrency       int    `json:"concurrency"`
	GroupIDs          string `json:"groupIds"`
	CustomVariables   string `json:"customVariables"` // JSON 对象格式的自定义变量
	Labels            string `json:"labels"`          // JSON 数组格式的自定义标签 ["env:prod","team:ops"]
}

// GroupListRequest 巡检组列表请求
type GroupListRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"pageSize" binding:"omitempty,min=1,max=100"`
	Name     string `form:"keyword"`
	Status   string `form:"status"`
}

// GroupResponse 巡检组响应
type GroupResponse struct {
	ID                uint     `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Status            string   `json:"status"`
	Sort              int      `json:"sort"`
	PrometheusURL     string   `json:"prometheusUrl"`
	PrometheusUsername string   `json:"prometheusUsername"`
	ExecutionMode     string   `json:"executionMode"`
	ExecutionStrategy string   `json:"executionStrategy"`
	Concurrency       int      `json:"concurrency"`
	GroupIDs          string   `json:"groupIds"`
	CustomVariables   string   `json:"customVariables"` // JSON 对象格式的自定义变量
	Labels            string   `json:"labels"`          // JSON 数组格式的自定义标签
	ItemCount         int      `json:"itemCount"`
	ItemNames         []string `json:"itemNames"`
	CreatedAt         string   `json:"createdAt"`
	UpdatedAt         string   `json:"updatedAt"`
}

// GroupExportData 巡检组导出数据
type GroupExportData struct {
	Name              string                  `json:"name" yaml:"name"`
	Description       string                  `json:"description" yaml:"description"`
	Status            string                  `json:"status" yaml:"status"`
	PrometheusURL     string                  `json:"prometheusUrl,omitempty" yaml:"prometheusUrl,omitempty"`
	PrometheusUsername string                  `json:"prometheusUsername,omitempty" yaml:"prometheusUsername,omitempty"`
	ExecutionMode     string                  `json:"executionMode" yaml:"executionMode"`
	ExecutionStrategy string                  `json:"executionStrategy" yaml:"executionStrategy"`
	Concurrency       int                     `json:"concurrency" yaml:"concurrency"`
	GroupIDs          []uint                  `json:"groupIds,omitempty" yaml:"groupIds,omitempty"`
	CustomVariables   map[string]string       `json:"customVariables,omitempty" yaml:"customVariables,omitempty"` // 自定义变量
	Items             []ItemExportData        `json:"items" yaml:"items"`
}

// ItemExportData 巡检项导出数据
type ItemExportData struct {
	Name              string   `json:"name" yaml:"name"`
	Description       string   `json:"description,omitempty" yaml:"description,omitempty"`
	ExecutionType     string   `json:"executionType" yaml:"executionType"`
	ExecutionStrategy string   `json:"executionStrategy" yaml:"executionStrategy"`
	Command           string   `json:"command,omitempty" yaml:"command,omitempty"`
	ScriptType        string   `json:"scriptType,omitempty" yaml:"scriptType,omitempty"`
	ScriptContent     string   `json:"scriptContent,omitempty" yaml:"scriptContent,omitempty"`
	ScriptFile        string   `json:"scriptFile,omitempty" yaml:"scriptFile,omitempty"`
	ScriptArgs        string   `json:"scriptArgs,omitempty" yaml:"scriptArgs,omitempty"`
	PromQLQuery       string   `json:"promqlQuery,omitempty" yaml:"promqlQuery,omitempty"`
	HostMatchType     string   `json:"hostMatchType,omitempty" yaml:"hostMatchType,omitempty"`
	HostTags          []string `json:"hostTags,omitempty" yaml:"hostTags,omitempty"`
	HostIDs           []uint   `json:"hostIds,omitempty" yaml:"hostIds,omitempty"`
	AssertionType     string   `json:"assertionType,omitempty" yaml:"assertionType,omitempty"`
	AssertionValue    string   `json:"assertionValue,omitempty" yaml:"assertionValue,omitempty"`
	VariableName      string   `json:"variableName,omitempty" yaml:"variableName,omitempty"`
	VariableRegex     string   `json:"variableRegex,omitempty" yaml:"variableRegex,omitempty"`
	Timeout           int      `json:"timeout" yaml:"timeout"`
	Status            string   `json:"status" yaml:"status"`
}

// GroupImportRequest 巡检组导入请求
type GroupImportRequest struct {
	Format string `json:"format" binding:"required,oneof=json yaml"` // json 或 yaml
	Data   string `json:"data" binding:"required"`                   // 导入的数据内容
}
