package inspection

// ProbeConfigRequest is the DTO for creating/updating a probe config.
type ProbeConfigRequest struct {
	ID           uint   `json:"id"`
	Name         string `json:"name" binding:"required,min=1,max=100"`
	Type         string `json:"type" binding:"required"`
	Category     string `json:"category"`
	Target       string `json:"target" binding:"required,max=255"`
	Port         int    `json:"port"`
	GroupID      uint   `json:"groupId"`
	GroupIDs     string `json:"groupIds"`
	Timeout      int    `json:"timeout"`
	Count        int    `json:"count"`
	PacketSize   int    `json:"packetSize"`
	Description  string `json:"description"`
	Tags         string `json:"tags"`
	ExecMode     string `json:"execMode"`
	AgentHostIDs string `json:"agentHostIds"`
	RetryCount   int    `json:"retryCount"`
	Status       int8   `json:"status"`
	Method       string `json:"method"`
	URL          string `json:"url"`
	Headers      string `json:"headers"`
	Params       string `json:"params"`
	Body         string `json:"body"`
	ProxyURL     string `json:"proxyUrl"`
	Assertions   string `json:"assertions"`
	ContentType  string `json:"contentType"`
	SkipVerify    *bool  `json:"skipVerify"`
	WSMessage     string `json:"wsMessage"`
	WSMessageType int    `json:"wsMessageType"`
	WSReadTimeout int    `json:"wsReadTimeout"`
}

// ToModel converts the request to a ProbeConfig model.
func (r *ProbeConfigRequest) ToModel() *ProbeConfig {
	category := r.Category
	if category == "" {
		if cat, ok := TypeToCategoryMap[r.Type]; ok {
			category = cat
		} else {
			category = ProbeCategoryNetwork
		}
	}
	m := &ProbeConfig{
		Name:         r.Name,
		Type:         r.Type,
		Category:     category,
		Target:       r.Target,
		Port:         r.Port,
		GroupID:      r.GroupID,
		GroupIDs:     r.GroupIDs,
		Timeout:      r.Timeout,
		Count:        r.Count,
		PacketSize:   r.PacketSize,
		Description:  r.Description,
		Tags:         r.Tags,
		ExecMode:     r.ExecMode,
		AgentHostIDs: r.AgentHostIDs,
		RetryCount:   r.RetryCount,
		Status:       r.Status,
		Method:       r.Method,
		URL:          r.URL,
		Headers:      r.Headers,
		Params:       r.Params,
		Body:         r.Body,
		ProxyURL:     r.ProxyURL,
		Assertions:   r.Assertions,
		ContentType:  r.ContentType,
		SkipVerify:   r.SkipVerify,
		WSMessage:    r.WSMessage,
		WSMessageType: r.WSMessageType,
		WSReadTimeout: r.WSReadTimeout,
	}
	if m.ExecMode == "" {
		m.ExecMode = ExecModeLocal
	}
	if m.RetryCount < 0 {
		m.RetryCount = 0
	}
	if m.RetryCount > 5 {
		m.RetryCount = 5
	}
	if m.Timeout == 0 {
		m.Timeout = 5
	}
	if m.Count == 0 {
		m.Count = 4
	}
	if m.PacketSize == 0 {
		m.PacketSize = 64
	}
	if category == ProbeCategoryApplication && m.Method == "" {
		m.Method = "GET"
	}
	return m
}

// ProbeTaskRequest is the DTO for creating/updating a probe task.
type ProbeTaskRequest struct {
	ID             uint   `json:"id"`
	Name           string `json:"name" binding:"required,min=1,max=100"`
	ProbeConfigIDs []uint `json:"probeConfigIds" binding:"required,min=1"`
	GroupID        uint   `json:"groupId"`
	CronExpr       string `json:"cronExpr" binding:"required"`
	PushgatewayID  uint   `json:"pushgatewayId"`
	Concurrency    int    `json:"concurrency"`
	Status         int8   `json:"status"`
	Description    string `json:"description"`
}

// ToModel converts the request to a ProbeTask model.
func (r *ProbeTaskRequest) ToModel() *ProbeTask {
	concurrency := r.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	if concurrency > 50 {
		concurrency = 50
	}
	return &ProbeTask{
		Name:          r.Name,
		GroupID:       r.GroupID,
		CronExpr:      r.CronExpr,
		PushgatewayID: r.PushgatewayID,
		Concurrency:   concurrency,
		Status:        r.Status,
		Description:   r.Description,
	}
}

// PushgatewayConfigRequest is the DTO for creating/updating a Pushgateway config.
type PushgatewayConfigRequest struct {
	ID        uint   `json:"id"`
	Name      string `json:"name" binding:"required,min=1,max=100"`
	URL       string `json:"url" binding:"required,url"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	IsDefault int8   `json:"isDefault"`
	Status    int8   `json:"status"`
}

// ToModel converts the request to a PushgatewayConfig model.
func (r *PushgatewayConfigRequest) ToModel() *PushgatewayConfig {
	return &PushgatewayConfig{
		Name:      r.Name,
		URL:       r.URL,
		Username:  r.Username,
		Password:  r.Password,
		IsDefault: r.IsDefault,
		Status:    r.Status,
	}
}

// ProbeConfigImportExport is used for YAML/JSON import/export.
type ProbeConfigImportExport struct {
	Name          string `json:"name" yaml:"name"`
	Type          string `json:"type" yaml:"type"`
	Category      string `json:"category" yaml:"category"`
	Target        string `json:"target" yaml:"target"`
	Port          int    `json:"port" yaml:"port"`
	Timeout       int    `json:"timeout" yaml:"timeout"`
	GroupIDs      string `json:"groupIds" yaml:"group_ids"`
	Count         int    `json:"count" yaml:"count"`
	PacketSize    int    `json:"packetSize" yaml:"packet_size"`
	Description   string `json:"description" yaml:"description"`
	Tags          string `json:"tags" yaml:"tags"`
	ExecMode      string `json:"execMode" yaml:"exec_mode"`
	AgentHostIDs  string `json:"agentHostIds" yaml:"agent_host_ids"`
	RetryCount    int    `json:"retryCount" yaml:"retry_count"`
	SkipVerify    *bool  `json:"skipVerify" yaml:"skip_verify"`
	WSMessage     string `json:"wsMessage" yaml:"ws_message"`
	WSMessageType int    `json:"wsMessageType" yaml:"ws_message_type"`
	WSReadTimeout int    `json:"wsReadTimeout" yaml:"ws_read_timeout"`
}

// ProbeVariableRequest is the DTO for creating/updating a probe variable.
type ProbeVariableRequest struct {
	ID          uint   `json:"id"`
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Value       string `json:"value" binding:"required"`
	VarType     string `json:"varType" binding:"required,oneof=plain secret"`
	GroupIDs    string `json:"groupIds"`
	Description string `json:"description"`
}
