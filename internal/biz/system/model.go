// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package system

import (
	"time"

	"gorm.io/gorm"
)

// SysConfig 系统配置表
type SysConfig struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Key       string         `gorm:"type:varchar(100);uniqueIndex;not null;comment:配置键" json:"key"`
	Value     string         `gorm:"type:text;comment:配置值" json:"value"`
	Type      string         `gorm:"type:varchar(20);default:'string';comment:配置类型(string/int/bool/json)" json:"type"`
	Group     string         `gorm:"type:varchar(50);index;comment:配置分组(basic/security)" json:"group"`
	Remark    string         `gorm:"type:varchar(200);comment:备注说明" json:"remark"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (SysConfig) TableName() string {
	return "sys_config"
}

// SysUserLoginAttempt 用户登录失败记录表
type SysUserLoginAttempt struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Username    string     `gorm:"type:varchar(50);index;not null;comment:用户名" json:"username"`
	FailCount   int        `gorm:"default:0;comment:失败次数" json:"failCount"`
	LastFailAt  time.Time  `gorm:"comment:最后失败时间" json:"lastFailAt"`
	LockedUntil *time.Time `gorm:"comment:锁定截止时间" json:"lockedUntil"`
}

// TableName 指定表名
func (SysUserLoginAttempt) TableName() string {
	return "sys_user_login_attempt"
}

// ConfigKey 配置键常量
const (
	// 基础配置
	ConfigKeySystemName        = "system_name"
	ConfigKeySystemLogo        = "system_logo"
	ConfigKeySystemDescription = "system_description"

	// 安全配置
	ConfigKeyPasswordMinLength = "password_min_length"
	ConfigKeySessionTimeout    = "session_timeout"
	ConfigKeyEnableCaptcha     = "enable_captcha"
	ConfigKeyMaxLoginAttempts  = "max_login_attempts"
	ConfigKeyLockoutDuration   = "lockout_duration"

	// 数据保留策略配置
	ConfigKeyInspectionRecordRetention = "inspection_record_retention" // 智能巡检执行记录保留数量
)

// ConfigGroup 配置分组常量
const (
	ConfigGroupBasic    = "basic"
	ConfigGroupSecurity = "security"
	ConfigGroupDataRetention = "data_retention" // 数据保留策略
)

// DefaultConfigs 默认配置
var DefaultConfigs = map[string]SysConfig{
	ConfigKeySystemName: {
		Key:    ConfigKeySystemName,
		Value:  "OpsHub",
		Type:   "string",
		Group:  ConfigGroupBasic,
		Remark: "系统名称",
	},
	ConfigKeySystemLogo: {
		Key:    ConfigKeySystemLogo,
		Value:  "",
		Type:   "string",
		Group:  ConfigGroupBasic,
		Remark: "系统Logo路径",
	},
	ConfigKeySystemDescription: {
		Key:    ConfigKeySystemDescription,
		Value:  "运维管理平台",
		Type:   "string",
		Group:  ConfigGroupBasic,
		Remark: "系统描述",
	},
	ConfigKeyPasswordMinLength: {
		Key:    ConfigKeyPasswordMinLength,
		Value:  "8",
		Type:   "int",
		Group:  ConfigGroupSecurity,
		Remark: "密码最小长度",
	},
	ConfigKeySessionTimeout: {
		Key:    ConfigKeySessionTimeout,
		Value:  "3600",
		Type:   "int",
		Group:  ConfigGroupSecurity,
		Remark: "Session超时时间(秒)",
	},
	ConfigKeyEnableCaptcha: {
		Key:    ConfigKeyEnableCaptcha,
		Value:  "true",
		Type:   "bool",
		Group:  ConfigGroupSecurity,
		Remark: "是否开启验证码",
	},
	ConfigKeyMaxLoginAttempts: {
		Key:    ConfigKeyMaxLoginAttempts,
		Value:  "5",
		Type:   "int",
		Group:  ConfigGroupSecurity,
		Remark: "最大登录失败次数",
	},
	ConfigKeyLockoutDuration: {
		Key:    ConfigKeyLockoutDuration,
		Value:  "300",
		Type:   "int",
		Group:  ConfigGroupSecurity,
		Remark: "账户锁定时间(秒)",
	},
	// 数据保留策略配置
	ConfigKeyInspectionRecordRetention: {
		Key:    ConfigKeyInspectionRecordRetention,
		Value:  "1000000",
		Type:   "int",
		Group:  ConfigGroupDataRetention,
		Remark: "智能巡检执行记录保留数量（默认100万条）",
	},
	// 定制功能默认配置
	ConfigKeyCustomTeleAIAuthEnabled: {
		Key:    ConfigKeyCustomTeleAIAuthEnabled,
		Value:  "false",
		Type:   "bool",
		Group:  ConfigGroupCustom,
		Remark: "是否启用 TeleAI Authorization 自动填充",
	},
	// Grafana 集成默认配置
	ConfigKeyGrafanaEnabled: {
		Key:    ConfigKeyGrafanaEnabled,
		Value:  "true",
		Type:   "bool",
		Group:  ConfigGroupIntegrationGrafana,
		Remark: "是否启用 Grafana 集成",
	},
	ConfigKeyGrafanaURL: {
		Key:    ConfigKeyGrafanaURL,
		Value:  "http://grafana_mon:3000/grafana_2syulinm/",
		Type:   "string",
		Group:  ConfigGroupIntegrationGrafana,
		Remark: "Grafana 访问地址",
	},
	ConfigKeyGrafanaSubpath: {
		Key:    ConfigKeyGrafanaSubpath,
		Value:  "/grafana_2syulinm/",
		Type:   "string",
		Group:  ConfigGroupIntegrationGrafana,
		Remark: "Grafana Sub-path",
	},
}

// 集成配置 Key 常量
const (
	ConfigGroupIntegrationGrafana       = "integration.grafana"
	ConfigKeyGrafanaEnabled             = "integration.grafana.enabled"
	ConfigKeyGrafanaURL                 = "integration.grafana.url"
	ConfigKeyGrafanaSubpath             = "integration.grafana.subpath"
)

// 定制功能配置 Key 常量
const (
	ConfigGroupCustom                = "custom"
	ConfigKeyCustomTeleAIAuthEnabled = "custom.teleai_auth.enabled"
)

// TeleAIAuthConfig 定制 TeleAI Authorization 自动填充全局开关
type TeleAIAuthConfig struct {
	Enabled bool `json:"enabled"`
}

// GrafanaIntegrationConfig Grafana 集成配置
type GrafanaIntegrationConfig struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Subpath string `json:"subpath"`
}

// BasicConfig 基础配置响应结构
type BasicConfig struct {
	SystemName        string `json:"systemName"`
	SystemLogo        string `json:"systemLogo"`
	SystemDescription string `json:"systemDescription"`
}

// SecurityConfig 安全配置响应结构
type SecurityConfig struct {
	PasswordMinLength int  `json:"passwordMinLength"`
	SessionTimeout    int  `json:"sessionTimeout"`
	EnableCaptcha     bool `json:"enableCaptcha"`
	MaxLoginAttempts  int  `json:"maxLoginAttempts"`
	LockoutDuration   int  `json:"lockoutDuration"`
}

// AllConfig 所有配置响应结构
type AllConfig struct {
	Basic         BasicConfig         `json:"basic"`
	Security      SecurityConfig      `json:"security"`
	DataRetention DataRetentionConfig `json:"dataRetention"`
}

// DataRetentionConfig 数据保留策略配置响应结构
type DataRetentionConfig struct {
	InspectionRecordRetention int `json:"inspectionRecordRetention"` // 智能巡检执行记录保留数量
}

// SysAPIKey API Key 表
type SysAPIKey struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(100);not null;comment:API Key名称" json:"name"`
	KeyHash     string         `gorm:"type:varchar(64);uniqueIndex;not null;comment:API Key哈希值(SHA256)" json:"-"`
	KeyPrefix   string         `gorm:"type:varchar(10);not null;comment:密钥前缀(用于展示)" json:"keyPrefix"`
	KeySuffix   string         `gorm:"type:varchar(10);not null;comment:密钥后缀(用于展示)" json:"keySuffix"`
	Description string         `gorm:"type:varchar(500);comment:描述" json:"description"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (SysAPIKey) TableName() string {
	return "sys_api_keys"
}

// APIKeyVO API Key 视图对象（用于列表展示，包含脱敏密钥和统计信息）
type APIKeyVO struct {
	ID             uint      `json:"id"`
	Name           string    `json:"name"`
	MaskedKey      string    `json:"maskedKey"`      // 脱敏密钥（前缀+***+后缀）
	Description    string    `json:"description"`
	TotalCalls     int64     `json:"totalCalls"`     // 总调用次数（从Redis读取）
	LastCalledAt   time.Time `json:"lastCalledAt"`   // 最后调用时间（从Redis读取）
	CreatedAt      time.Time `json:"createdAt"`
}

// CreateAPIKeyResponse 创建 API Key 响应（仅此接口返回完整明文密钥）
type CreateAPIKeyResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	APIKey      string    `json:"apiKey"`      // 完整明文密钥（仅创建时返回一次）
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ConfigGroup 配置分组常量 - API Key
const (
	ConfigGroupAPIKey = "api_key" // API Key 管理
)

// Redis Key 前缀常量
const (
	RedisKeyAPIKeyCallCount  = "apikey:call_count:"   // API Key 调用次数 apikey:call_count:{key_hash}
	RedisKeyAPIKeyLastCalled = "apikey:last_called:"  // API Key 最后调用时间 apikey:last_called:{key_hash}
)
