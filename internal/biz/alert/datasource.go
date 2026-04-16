package alert

import (
	"time"

	"gorm.io/gorm"
)

// AlertDataSource 告警数据源
type AlertDataSource struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Name           string         `gorm:"size:100;not null" json:"name"`
	Type           string         `gorm:"size:30;not null" json:"type"` // prometheus, victoriametrics, influxdb
	URL            string         `gorm:"size:500" json:"url"`           // 直连模式时存储完整URL
	Username       string         `gorm:"size:100" json:"username"`
	Password       string         `gorm:"size:200" json:"password"`
	Token          string         `gorm:"size:500" json:"token"`
	Description    string         `gorm:"size:500" json:"description"`
	Status         int            `gorm:"default:1" json:"status"` // 1=启用 0=禁用

	// Agent代理相关字段
	AccessMode     string         `gorm:"size:20;default:'direct';index" json:"access_mode"` // "direct" 或 "agent"
	Host           string         `gorm:"size:255" json:"host"`                              // Agent代理模式时存储数据源地址
	Port           int            `gorm:"default:0" json:"port"`                             // Agent代理模式时存储数据源端口
	AgentHostIDs   string         `gorm:"type:text" json:"agent_host_ids"`                   // JSON数组: "[1,2,3]"
	// ProxyToken: 仅Agent模式使用，直连模式为空。使用稀疏唯一索引(MySQL 8.0+)或在应用层验证
	ProxyToken     string         `gorm:"size:100;index:idx_proxy_token,type:BTREE" json:"proxy_token"` // UUID，唯一标识（仅Agent模式）
	ProxyURL       string         `gorm:"size:500" json:"proxy_url"`                   // 自动生成: /api/v1/alert/proxy/datasource/{token}
	ProxyEnabled   bool           `gorm:"default:false" json:"proxy_enabled"`
}

func (AlertDataSource) TableName() string {
	return "alert_datasources"
}

// CreateDataSourceRequest 创建数据源的请求体
type CreateDataSourceRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	Type        string `json:"type" binding:"required,oneof=prometheus victoriametrics influxdb"`
	AccessMode  string `json:"access_mode" binding:"required,oneof=direct agent"`

	// URL字段（两种模式都使用）
	URL         string `json:"url" binding:"required,max=500"` // 直连和Agent模式都需要

	// 认证字段（两种模式都可选）
	Username    string `json:"username" binding:"omitempty,max=100"`
	Password    string `json:"password" binding:"omitempty,max=200"`
	Token       string `json:"token" binding:"omitempty,max=500"`
	Description string `json:"description" binding:"omitempty,max=500"`
	Status      int    `json:"status" binding:"omitempty,oneof=0 1"`
}

// UpdateDataSourceRequest 更新数据源的请求体
type UpdateDataSourceRequest struct {
	Name        string `json:"name" binding:"omitempty,max=100"`
	Type        string `json:"type" binding:"omitempty,oneof=prometheus victoriametrics influxdb"`

	// URL字段（两种模式都使用）
	URL         string `json:"url" binding:"omitempty,max=500"`

	// 认证字段
	Username    string `json:"username" binding:"omitempty,max=100"`
	Password    string `json:"password" binding:"omitempty,max=200"`
	Token       string `json:"token" binding:"omitempty,max=500"`
	Description string `json:"description" binding:"omitempty,max=500"`
	Status      int    `json:"status" binding:"omitempty,oneof=0 1"`
}
