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

// AIModelProxy AI模型代理模型
type AIModelProxy struct {
	gorm.Model
	Name        string `gorm:"type:varchar(100);not null;comment:AI模型代理名称" json:"name"`
	Description string `gorm:"type:varchar(500);comment:描述" json:"description"`
	ModelType   string `gorm:"type:varchar(50);not null;default:'ollama';comment:模型类型 ollama/openai/custom" json:"modelType"`
	Status      int    `gorm:"type:tinyint;default:1;comment:状态 1:启用 0:禁用" json:"status"`

	// 连接配置
	TargetURL string `gorm:"type:varchar(500);not null;comment:目标URL" json:"targetUrl"`
	APIKey    string `gorm:"type:varchar(500);comment:API密钥(加密)" json:"apiKey"`
	Timeout   int    `gorm:"type:int;default:300;comment:超时时间(秒)" json:"timeout"`

	// 代理访问Token（永久有效）
	ProxyToken string `gorm:"type:varchar(64);uniqueIndex;comment:代理访问Token(UUID)" json:"proxyToken"`

	// 分组关联
	GroupID uint `gorm:"not null;index;comment:资产分组ID" json:"groupId"`
}

// AIModelProxyAgent AI模型代理与Agent主机关联表
type AIModelProxyAgent struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	ProxyID   uint      `gorm:"column:proxy_id;not null;index;comment:AI模型代理ID" json:"proxyId"`
	HostID    uint      `gorm:"column:host_id;not null;index;comment:主机ID" json:"hostId"`
}

// AIModelProxyRequest AI模型代理请求
type AIModelProxyRequest struct {
	ID           uint   `json:"id"`
	Name         string `json:"name" binding:"required,min=2,max=100"`
	Description  string `json:"description"`
	ModelType    string `json:"modelType" binding:"required,oneof=ollama openai custom"`
	Status       int    `json:"status"`
	TargetURL    string `json:"targetUrl" binding:"required,url"`
	APIKey       string `json:"apiKey"`
	Timeout      int    `json:"timeout"`
	GroupID      uint   `json:"groupId" binding:"required"`
	AgentHostIDs []uint `json:"agentHostIds" binding:"required,min=1"`
}

// AIModelProxyVO AI模型代理VO
type AIModelProxyVO struct {
	ID             uint     `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	ModelType      string   `json:"modelType"`
	ModelTypeText  string   `json:"modelTypeText"`
	Status         int      `json:"status"`
	StatusText     string   `json:"statusText"`
	TargetURL      string   `json:"targetUrl"`
	APIKey         string   `json:"apiKey"` // 仅在详情接口返回（脱敏）
	Timeout        int      `json:"timeout"`
	ProxyToken     string   `json:"proxyToken"`
	ProxyURL       string   `json:"proxyUrl"` // 完整的代理访问URL
	GroupID        uint     `json:"groupId"`
	GroupName      string   `json:"groupName"`
	AgentHostIDs   []uint   `json:"agentHostIds"`
	AgentHostNames []string `json:"agentHostNames"`
	AgentOnline    bool     `json:"agentOnline"` // 是否有在线的Agent
	CreateTime     string   `json:"createTime"`
	UpdateTime     string   `json:"updateTime"`
}

// ToModel 转换为模型
func (req *AIModelProxyRequest) ToModel() *AIModelProxy {
	proxy := &AIModelProxy{
		Name:        req.Name,
		Description: req.Description,
		ModelType:   req.ModelType,
		Status:      req.Status,
		TargetURL:   req.TargetURL,
		APIKey:      req.APIKey,
		GroupID:     req.GroupID,
	}

	// 设置超时时间，默认300秒
	if req.Timeout > 0 {
		proxy.Timeout = req.Timeout
	} else {
		proxy.Timeout = 300
	}

	return proxy
}

// TableName 指定表名
func (AIModelProxy) TableName() string {
	return "ai_model_proxies"
}

// TableName 指定表名
func (AIModelProxyAgent) TableName() string {
	return "ai_model_proxy_agents"
}

// AIModelProxyRepo AI模型代理仓储接口
type AIModelProxyRepo interface {
	// Create 创建AI模型代理
	Create(proxy *AIModelProxy, agentHostIDs []uint) error
	// Update 更新AI模型代理
	Update(proxy *AIModelProxy, agentHostIDs []uint) error
	// Delete 删除AI模型代理
	Delete(id uint) error
	// GetByID 根据ID获取AI模型代理
	GetByID(id uint) (*AIModelProxy, error)
	// GetByToken 根据Token获取AI模型代理
	GetByToken(token string) (*AIModelProxy, error)
	// List 获取AI模型代理列表
	List(page, pageSize int, groupID uint, status *int, keyword string) ([]*AIModelProxy, int64, error)
	// GetAgentHostIDs 获取AI模型代理绑定的Agent主机ID列表
	GetAgentHostIDs(proxyID uint) ([]uint, error)
	// RegenerateToken 重新生成Token
	RegenerateToken(id uint, newToken string) error
}
