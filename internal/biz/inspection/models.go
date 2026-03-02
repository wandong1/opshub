package inspection

import (
	"time"

	"gorm.io/gorm"
)

// Probe category constants.
const (
	ProbeCategoryNetwork     = "network"     // 基础网络（Ping）
	ProbeCategoryLayer4      = "layer4"      // 四层协议（TCP/UDP）
	ProbeCategoryApplication = "application" // 应用服务（HTTP/DNS/WS/SSL）— 预留
	ProbeCategoryWorkflow    = "workflow"     // 业务流程 — 预留
	ProbeCategoryMiddleware  = "middleware"   // 中间件&数据库 — 预留
)

// Variable type constants.
const (
	VariableTypePlain  = "plain"
	VariableTypeSecret = "secret"
)

// Execution mode constants.
const (
	ExecModeLocal = "local"
	ExecModeAgent = "agent"
	ExecModeProxy = "proxy"
)

// CategoryTypeMap maps category to allowed probe types.
var CategoryTypeMap = map[string][]string{
	ProbeCategoryNetwork:     {"ping"},
	ProbeCategoryLayer4:      {"tcp", "udp"},
	ProbeCategoryApplication: {"http", "https", "websocket"},
	ProbeCategoryWorkflow:    {"workflow"},
	ProbeCategoryMiddleware:  {"mysql", "redis", "kafka", "clickhouse", "mongodb", "rabbitmq", "rocketmq", "postgresql", "sqlserver", "milvus"},
}

// TypeToCategoryMap is the reverse mapping from type to category.
var TypeToCategoryMap = func() map[string]string {
	m := make(map[string]string)
	for cat, types := range CategoryTypeMap {
		for _, t := range types {
			m[t] = cat
		}
	}
	return m
}()

// ProbeConfig represents a network probe configuration.
type ProbeConfig struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(100);not null" json:"name"`
	Type        string         `gorm:"type:varchar(20);not null" json:"type"` // ping/tcp/udp
	Category    string         `gorm:"type:varchar(20);not null;default:'network';index" json:"category"`
	Target      string         `gorm:"type:varchar(255);not null" json:"target"`
	Port        int            `gorm:"default:0" json:"port"`
	GroupID     uint           `gorm:"index" json:"groupId"`
	GroupIDs    string         `gorm:"type:varchar(500);default:''" json:"groupIds"` // 逗号分隔多分组ID
	Timeout     int            `gorm:"default:5" json:"timeout"`
	Count       int            `gorm:"default:4" json:"count"`
	PacketSize  int            `gorm:"default:64" json:"packetSize"`
	Description string         `gorm:"type:varchar(500)" json:"description"`
	Tags         string         `gorm:"type:varchar(500)" json:"tags"`
	ExecMode     string         `gorm:"type:varchar(10);default:'local'" json:"execMode"`     // local/agent/proxy
	AgentHostIDs string         `gorm:"type:varchar(500);default:''" json:"agentHostIds"`      // 逗号分隔主机ID
	RetryCount   int            `gorm:"default:0" json:"retryCount"`                           // 失败重试次数 0-5
	Method       string         `gorm:"type:varchar(10);default:'GET'" json:"method"`
	URL          string         `gorm:"type:varchar(2000)" json:"url"`
	Headers      string         `gorm:"type:text" json:"headers"`
	Params       string         `gorm:"type:text" json:"params"`
	Body         string         `gorm:"type:longtext" json:"body"`
	ProxyURL     string         `gorm:"type:varchar(500)" json:"proxyUrl"`
	Assertions   string         `gorm:"type:text" json:"assertions"`
	ContentType  string         `gorm:"type:varchar(100)" json:"contentType"`
	SkipVerify   *bool          `gorm:"default:true" json:"skipVerify"`
	WSMessage    string         `gorm:"type:text" json:"wsMessage"`
	WSMessageType int           `gorm:"default:1" json:"wsMessageType"`
	WSMessageFormat string      `gorm:"type:varchar(20);default:'text'" json:"wsMessageFormat"`
	WSReadTimeout int           `gorm:"default:5" json:"wsReadTimeout"`
	Status       int8           `gorm:"default:1" json:"status"`                               // 1=enabled 0=disabled
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ProbeConfig) TableName() string { return "probe_configs" }

// ProbeTask represents a scheduled probe task.
type ProbeTask struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"type:varchar(100);not null" json:"name"`
	GroupID       uint           `json:"groupId"`
	CronExpr      string         `gorm:"type:varchar(50);not null" json:"cronExpr"`
	PushgatewayID uint           `json:"pushgatewayId"`
	Concurrency   int            `gorm:"default:5" json:"concurrency"`
	Status        int8           `gorm:"default:1" json:"status"`
	LastRunAt     *time.Time     `json:"lastRunAt"`
	LastResult    string         `gorm:"type:varchar(20)" json:"lastResult"`
	Description   string         `gorm:"type:varchar(500)" json:"description"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ProbeTask) TableName() string { return "probe_tasks" }

// ProbeTaskConfig is the many-to-many association between ProbeTask and ProbeConfig.
type ProbeTaskConfig struct {
	ProbeTaskID   uint `gorm:"primaryKey;column:probe_task_id" json:"probeTaskId"`
	ProbeConfigID uint `gorm:"primaryKey;column:probe_config_id" json:"probeConfigId"`
}

func (ProbeTaskConfig) TableName() string { return "probe_task_configs" }

// ProbeResult stores the outcome of a single probe execution.
type ProbeResult struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ProbeTaskID     uint      `gorm:"index" json:"probeTaskId"`
	ProbeConfigID   uint      `gorm:"index" json:"probeConfigId"`
	Success         bool      `json:"success"`
	Latency         float64   `json:"latency"`
	PacketLoss      float64   `json:"packetLoss"`
	PingRttAvg      float64   `json:"pingRttAvg"`
	PingRttMin      float64   `json:"pingRttMin"`
	PingRttMax      float64   `json:"pingRttMax"`
	PingStddev      float64   `json:"pingStddev"`
	PingPacketsSent int       `json:"pingPacketsSent"`
	PingPacketsRecv int       `json:"pingPacketsRecv"`
	TCPConnectTime  float64   `json:"tcpConnectTime"`
	UDPWriteTime    float64   `json:"udpWriteTime"`
	UDPReadTime     float64   `json:"udpReadTime"`
	HTTPStatusCode    int     `json:"httpStatusCode"`
	HTTPResponseTime  float64 `json:"httpResponseTime"`
	HTTPContentLength int64   `json:"httpContentLength"`
	AssertionSuccess  bool    `json:"assertionSuccess"`
	AssertionDetail   string  `gorm:"type:text" json:"assertionDetail"`
	ErrorMessage    string    `gorm:"type:text" json:"errorMessage"`
	Detail          string    `gorm:"type:text" json:"detail"`
	AgentHostID     uint      `gorm:"default:0" json:"agentHostId"`   // 执行的Agent主机ID，0=本地
	RetryAttempt    int       `gorm:"default:0" json:"retryAttempt"`  // 实际重试次数
	CreatedAt       time.Time `json:"createdAt"`
}

func (ProbeResult) TableName() string { return "probe_results" }

// PushgatewayConfig stores Pushgateway connection info.
type PushgatewayConfig struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"type:varchar(100);not null" json:"name"`
	URL       string         `gorm:"type:varchar(255);not null" json:"url"`
	Username  string         `gorm:"type:varchar(100)" json:"username"`
	Password  string         `gorm:"type:varchar(255)" json:"password"`
	IsDefault int8           `gorm:"default:0" json:"isDefault"`
	Status    int8           `gorm:"default:1" json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (PushgatewayConfig) TableName() string { return "pushgateway_configs" }

// ProbeVariable represents a global environment variable for probe configs.
type ProbeVariable struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(100);not null;uniqueIndex:idx_name_group" json:"name"`
	Value       string         `gorm:"type:text;not null" json:"value"`
	VarType     string         `gorm:"type:varchar(10);not null;default:'plain'" json:"varType"`
	GroupIDs    string         `gorm:"type:varchar(500);default:'';uniqueIndex:idx_name_group" json:"groupIds"`
	Description string         `gorm:"type:varchar(500)" json:"description"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ProbeVariable) TableName() string { return "probe_variables" }

// ===== Workflow (业务流程) 结构体 =====

// WorkflowDefinition is the JSON structure stored in ProbeConfig.Body for workflow probes.
type WorkflowDefinition struct {
	Variables     map[string]string `json:"variables"`
	StopOnFailure bool              `json:"stopOnFailure"`
	Steps         []WorkflowStep    `json:"steps"`
}

// WorkflowStepAssertion mirrors probers.Assertion to avoid circular imports.
type WorkflowStepAssertion struct {
	Name      string `json:"name"`
	Source    string `json:"source"`
	Path      string `json:"path"`
	Condition string `json:"condition"`
	Value     string `json:"value"`
}

// WorkflowStep defines a single step in a workflow probe.
type WorkflowStep struct {
	Name          string                  `json:"name"`
	StepType      string                  `json:"stepType"`      // http(default)/ws_connect/ws_send/ws_receive/ws_disconnect
	Delay         int                     `json:"delay"`         // seconds to wait before executing
	Method        string                  `json:"method"`
	URL           string                  `json:"url"`
	ContentType   string                  `json:"contentType"`
	Headers       map[string]string       `json:"headers"`
	Params        map[string]string       `json:"params"`
	Body          string                  `json:"body"`
	Timeout       int                     `json:"timeout"`
	SkipVerify    *bool                   `json:"skipVerify"`
	WSMessage     string                  `json:"wsMessage"`
	WSMessageType int                     `json:"wsMessageType"` // 1=text, 2=binary
	WSMessageFormat string                `json:"wsMessageFormat"`
	WSReadTimeout int                     `json:"wsReadTimeout"`
	WSReceiveMode string                  `json:"wsReceiveMode"` // "single"(default) / "stream"
	Assertions    []WorkflowStepAssertion `json:"assertions"`
	Extractions   []StepExtraction        `json:"extractions"`
	ExecMode      string                  `json:"execMode"`
	ProxyURL      string                  `json:"proxyUrl"`
}

// StepExtraction defines how to extract a variable from a step's response.
type StepExtraction struct {
	Name   string `json:"name"`   // variable name
	Source string `json:"source"` // "body" or "header"
	Path   string `json:"path"`   // GJSON path for body, header name for header
}

// WorkflowResult holds the outcome of a workflow probe execution.
type WorkflowResult struct {
	Success      bool                 `json:"success"`
	TotalLatency float64              `json:"totalLatency"`
	StepResults  []WorkflowStepResult `json:"stepResults"`
	Error        string               `json:"error,omitempty"`
}

// WorkflowStepResult holds the outcome of a single workflow step.
type WorkflowStepResult struct {
	StepName         string            `json:"stepName"`
	StepIndex        int               `json:"stepIndex"`
	StepType         string            `json:"stepType,omitempty"`
	Success          bool              `json:"success"`
	Skipped          bool              `json:"skipped"`
	HTTPStatusCode   int               `json:"httpStatusCode,omitempty"`
	HTTPResponseTime float64           `json:"httpResponseTime,omitempty"`
	Latency          float64           `json:"latency,omitempty"`
	ResponseBody     string            `json:"responseBody,omitempty"`
	ResponseHeaders  map[string]string `json:"responseHeaders,omitempty"`
	ExtractedVars    map[string]string `json:"extractedVars,omitempty"`
	AssertionResults []WorkflowAssertionResult `json:"assertionResults,omitempty"`
	Error            string            `json:"error,omitempty"`
}

// WorkflowAssertionResult mirrors probers.AssertionResult.
type WorkflowAssertionResult struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Actual  string `json:"actual"`
	Error   string `json:"error"`
}
