package dto

// GroupCreateRequest 创建巡检组请求
type GroupCreateRequest struct {
	Name               string `json:"name" binding:"required"`
	Description        string `json:"description"`
	Status             string `json:"status"`
	Sort               int    `json:"sort"`
	PrometheusURL      string `json:"prometheus_url"`
	PrometheusUsername string `json:"prometheus_username"`
	PrometheusPassword string `json:"prometheus_password"`
	ExecutionMode      string `json:"execution_mode"`
	GroupIDs           string `json:"group_ids"`
}

// GroupUpdateRequest 更新巡检组请求
type GroupUpdateRequest struct {
	Name               string `json:"name"`
	Description        string `json:"description"`
	Status             string `json:"status"`
	Sort               int    `json:"sort"`
	PrometheusURL      string `json:"prometheus_url"`
	PrometheusUsername string `json:"prometheus_username"`
	PrometheusPassword string `json:"prometheus_password"`
	ExecutionMode      string `json:"execution_mode"`
	GroupIDs           string `json:"group_ids"`
}

// GroupListRequest 巡检组列表请求
type GroupListRequest struct {
	Page     int    `form:"page" binding:"required,min=1"`
	PageSize int    `form:"page_size" binding:"required,min=1,max=100"`
	Name     string `form:"name"`
	Status   string `form:"status"`
}

// GroupResponse 巡检组响应
type GroupResponse struct {
	ID                 uint   `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	Status             string `json:"status"`
	Sort               int    `json:"sort"`
	PrometheusURL      string `json:"prometheus_url"`
	PrometheusUsername string `json:"prometheus_username"`
	ExecutionMode      string `json:"execution_mode"`
	GroupIDs           string `json:"group_ids"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}
