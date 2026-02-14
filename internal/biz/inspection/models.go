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

// CategoryTypeMap maps category to allowed probe types.
var CategoryTypeMap = map[string][]string{
	ProbeCategoryNetwork:     {"ping"},
	ProbeCategoryLayer4:      {"tcp", "udp"},
	ProbeCategoryApplication: {"http", "https", "dns", "websocket", "ssl_cert"},
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
	Timeout     int            `gorm:"default:5" json:"timeout"`
	Count       int            `gorm:"default:4" json:"count"`
	PacketSize  int            `gorm:"default:64" json:"packetSize"`
	Description string         `gorm:"type:varchar(500)" json:"description"`
	Tags        string         `gorm:"type:varchar(500)" json:"tags"`
	Status      int8           `gorm:"default:1" json:"status"` // 1=enabled 0=disabled
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
	ErrorMessage    string    `gorm:"type:text" json:"errorMessage"`
	Detail          string    `gorm:"type:text" json:"detail"`
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
