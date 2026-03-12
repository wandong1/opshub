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
	"time"

	"gorm.io/gorm"
)

// Website 站点模型
type Website struct {
	gorm.Model
	Name            string `gorm:"type:varchar(100);not null;comment:站点名称" json:"name"`
	URL             string `gorm:"type:varchar(500);not null;comment:站点URL" json:"url"`
	Icon            string `gorm:"type:varchar(200);comment:站点图标" json:"icon"`
	Type            string `gorm:"type:varchar(20);not null;comment:站点类型 external:外部 internal:内部" json:"type"`
	Credential      string `gorm:"type:varchar(500);comment:加密凭据" json:"credential"`
	SecureCopyURL   bool   `gorm:"type:tinyint;default:0;comment:安全复制URL" json:"secureCopyUrl"`
	AccessUser      string `gorm:"type:varchar(100);comment:访问用户名" json:"accessUser"`
	AccessPassword  string `gorm:"type:varchar(500);comment:访问密码(加密)" json:"accessPassword"`
	Description     string `gorm:"type:varchar(500);comment:备注" json:"description"`
	Status          int    `gorm:"type:tinyint;default:1;comment:状态 1:启用 0:禁用" json:"status"`
}

// WebsiteGroup 站点与业务分组关联表
type WebsiteGroup struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	WebsiteID uint      `gorm:"column:website_id;not null;index;comment:站点ID" json:"websiteId"`
	GroupID   uint      `gorm:"column:group_id;not null;index;comment:分组ID" json:"groupId"`
}

// WebsiteAgent 站点与Agent主机关联表（仅内部站点）
type WebsiteAgent struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	WebsiteID uint      `gorm:"column:website_id;not null;index;comment:站点ID" json:"websiteId"`
	HostID    uint      `gorm:"column:host_id;not null;index;comment:主机ID" json:"hostId"`
}

// WebsiteRequest 站点请求
type WebsiteRequest struct {
	ID             uint     `json:"id"`
	Name           string   `json:"name" binding:"required,min=2,max=100"`
	URL            string   `json:"url" binding:"required"`
	Icon           string   `json:"icon"`
	Type           string   `json:"type" binding:"required,oneof=external internal"`
	Credential     string   `json:"credential"`
	SecureCopyURL  bool     `json:"secureCopyUrl"`
	AccessUser     string   `json:"accessUser"`
	AccessPassword string   `json:"accessPassword"`
	Description    string   `json:"description"`
	Status         int      `json:"status"`
	GroupIDs       []uint   `json:"groupIds"`
	AgentHostIDs   []uint   `json:"agentHostIds"`
}

// WebsiteVO 站点VO
type WebsiteVO struct {
	ID             uint     `json:"id"`
	Name           string   `json:"name"`
	URL            string   `json:"url"`
	Icon           string   `json:"icon"`
	Type           string   `json:"type"`
	TypeText       string   `json:"typeText"`
	Credential     string   `json:"credential"`
	SecureCopyURL  bool     `json:"secureCopyUrl"`
	AccessUser     string   `json:"accessUser"`
	AccessPassword string   `json:"accessPassword"` // 访问密码（仅在详情接口返回）
	Description    string   `json:"description"`
	Status         int      `json:"status"`
	StatusText     string   `json:"statusText"`
	CreateTime     string   `json:"createTime"`
	UpdateTime     string   `json:"updateTime"`
	GroupNames     []string `json:"groupNames"`
	GroupIDs       []uint   `json:"groupIds"`
	AgentHostIDs   []uint   `json:"agentHostIds"`
	AgentHostNames []string `json:"agentHostNames"`
	AgentOnline    bool     `json:"agentOnline"` // 是否有在线的Agent
}

// ToModel 转换为模型
func (req *WebsiteRequest) ToModel() *Website {
	return &Website{
		Name:           req.Name,
		URL:            req.URL,
		Icon:           req.Icon,
		Type:           req.Type,
		Credential:     req.Credential,
		SecureCopyURL:  req.SecureCopyURL,
		AccessUser:     req.AccessUser,
		AccessPassword: req.AccessPassword,
		Description:    req.Description,
		Status:         req.Status,
	}
}

// TableName 指定表名
func (Website) TableName() string {
	return "websites"
}

// TableName 指定表名
func (WebsiteGroup) TableName() string {
	return "website_groups"
}

// TableName 指定表名
func (WebsiteAgent) TableName() string {
	return "website_agents"
}
