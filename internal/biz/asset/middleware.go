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

package asset

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// 中间件类型常量
const (
	MiddlewareTypeMySQL      = "mysql"
	MiddlewareTypeRedis      = "redis"
	MiddlewareTypeClickHouse = "clickhouse"
	MiddlewareTypeMongoDB    = "mongodb"
	MiddlewareTypeKafka      = "kafka"
	MiddlewareTypeMilvus     = "milvus"
)

// 各中间件类型的默认端口
var DefaultPorts = map[string]int{
	MiddlewareTypeMySQL:      3306,
	MiddlewareTypeRedis:      6379,
	MiddlewareTypeClickHouse: 9000,
	MiddlewareTypeMongoDB:    27017,
	MiddlewareTypeKafka:      9092,
	MiddlewareTypeMilvus:     19530,
}

// Middleware 中间件模型
type Middleware struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	Name             string         `gorm:"type:varchar(100);not null;comment:中间件名称" json:"name"`
	Type             string         `gorm:"type:varchar(20);not null;comment:类型: mysql/redis/clickhouse/mongodb/kafka/milvus" json:"type"`
	GroupID          uint           `gorm:"column:group_id;comment:所属业务分组ID" json:"groupId"`
	HostID           uint           `gorm:"column:host_id;comment:关联主机ID" json:"hostId"`
	Host             string         `gorm:"type:varchar(255);not null;comment:连接地址" json:"host"`
	Port             int            `gorm:"type:int;not null;comment:连接端口" json:"port"`
	Username         string         `gorm:"type:varchar(100);comment:用户名" json:"username"`
	Password         string         `gorm:"type:varchar(500);comment:密码(加密存储)" json:"password,omitempty"`
	DatabaseName     string         `gorm:"type:varchar(100);comment:默认数据库/索引" json:"databaseName"`
	ConnectionParams string         `gorm:"type:text;comment:额外连接参数(JSON)" json:"connectionParams"`
	Tags             string         `gorm:"type:varchar(500);comment:标签" json:"tags"`
	Description      string         `gorm:"type:varchar(500);comment:备注" json:"description"`
	Status           int            `gorm:"type:tinyint;default:-1;comment:状态 1:在线 0:离线 -1:未知" json:"status"`
	Version          string         `gorm:"type:varchar(50);comment:中间件版本" json:"version"`
	LastChecked      *time.Time     `gorm:"column:last_checked;comment:最后检测时间" json:"lastChecked,omitempty"`
}

// TableName 指定表名
func (Middleware) TableName() string {
	return "middlewares"
}

// MiddlewareRequest 创建/更新中间件请求
type MiddlewareRequest struct {
	ID               uint   `json:"id"`
	Name             string `json:"name" binding:"required,min=2,max=100"`
	Type             string `json:"type" binding:"required,oneof=mysql redis clickhouse mongodb kafka milvus"`
	GroupID          uint   `json:"groupId"`
	HostID           uint   `json:"hostId"`
	Host             string `json:"host" binding:"required"`
	Port             int    `json:"port" binding:"required,min=1,max=65535"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	DatabaseName     string `json:"databaseName"`
	ConnectionParams string `json:"connectionParams"`
	Tags             string `json:"tags"`
	Description      string `json:"description"`
}

// MiddlewareListRequest 列表查询参数
type MiddlewareListRequest struct {
	Page     int    `form:"page" binding:"required,min=1"`
	PageSize int    `form:"pageSize" binding:"required,min=1,max=100"`
	Keyword  string `form:"keyword"`
	Type     string `form:"type"`
	GroupID  *uint  `form:"groupId"`
	Status   *int   `form:"status"`
}

// MiddlewareInfoVO 中间件信息VO
type MiddlewareInfoVO struct {
	ID               uint     `json:"id"`
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	TypeText         string   `json:"typeText"`
	GroupID          uint     `json:"groupId"`
	GroupName        string   `json:"groupName"`
	HostID           uint     `json:"hostId"`
	HostName         string   `json:"hostName,omitempty"`
	Host             string   `json:"host"`
	Port             int      `json:"port"`
	Username         string   `json:"username"`
	DatabaseName     string   `json:"databaseName"`
	ConnectionParams string   `json:"connectionParams"`
	Tags             []string `json:"tags"`
	Description      string   `json:"description"`
	Status           int      `json:"status"`
	StatusText       string   `json:"statusText"`
	Version          string   `json:"version"`
	LastChecked      string   `json:"lastChecked,omitempty"`
	CreateTime       string   `json:"createTime"`
	UpdateTime       string   `json:"updateTime"`
}

// ToModel 转换为模型
func (req *MiddlewareRequest) ToModel() *Middleware {
	return &Middleware{
		Name:             req.Name,
		Type:             req.Type,
		GroupID:          req.GroupID,
		HostID:           req.HostID,
		Host:             req.Host,
		Port:             req.Port,
		Username:         req.Username,
		Password:         req.Password,
		DatabaseName:     req.DatabaseName,
		ConnectionParams: req.ConnectionParams,
		Tags:             req.Tags,
		Description:      req.Description,
		Status:           -1, // 初始状态未知
	}
}

// GetTypeText 获取类型文本
func GetMiddlewareTypeText(mwType string) string {
	switch mwType {
	case MiddlewareTypeMySQL:
		return "MySQL"
	case MiddlewareTypeRedis:
		return "Redis"
	case MiddlewareTypeClickHouse:
		return "ClickHouse"
	case MiddlewareTypeMongoDB:
		return "MongoDB"
	case MiddlewareTypeKafka:
		return "Kafka"
	case MiddlewareTypeMilvus:
		return "Milvus"
	default:
		return mwType
	}
}

// MiddlewareRepo 中间件仓储接口
type MiddlewareRepo interface {
	Create(ctx context.Context, mw *Middleware) error
	Update(ctx context.Context, mw *Middleware) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*Middleware, error)
	GetByIDDecrypted(ctx context.Context, id uint) (*Middleware, error)
	List(ctx context.Context, page, pageSize int, keyword, mwType string, groupIDs []uint, status *int, accessibleIDs []uint) ([]*Middleware, int64, error)
	BatchDelete(ctx context.Context, ids []uint) error
	UpdateStatus(ctx context.Context, id uint, status int, version string) error
}
