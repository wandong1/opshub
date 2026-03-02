package asset

import "gorm.io/gorm"

// ServiceLabel 服务标签预添加模型
type ServiceLabel struct {
	gorm.Model
	Name            string `gorm:"type:varchar(100);not null;uniqueIndex;comment:标签名" json:"name"`
	MatchProcesses  string `gorm:"type:varchar(500);not null;comment:匹配进程名(逗号分隔)" json:"matchProcesses"`
	Description     string `gorm:"type:varchar(500);comment:描述" json:"description"`
	Status          int    `gorm:"type:tinyint;default:1;comment:状态 1:启用 0:禁用" json:"status"`
}

// ServiceLabelRequest 服务标签请求
type ServiceLabelRequest struct {
	ID             uint   `json:"id"`
	Name           string `json:"name" binding:"required,min=1,max=100"`
	MatchProcesses string `json:"matchProcesses" binding:"required"`
	Description    string `json:"description"`
	Status         int    `json:"status"`
}

// ServiceLabelVO 服务标签VO
type ServiceLabelVO struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	MatchProcesses string `json:"matchProcesses"`
	Description    string `json:"description"`
	Status         int    `json:"status"`
	CreateTime     string `json:"createTime"`
	UpdateTime     string `json:"updateTime"`
}

// ToModel 转换为模型
func (req *ServiceLabelRequest) ToModel() *ServiceLabel {
	return &ServiceLabel{
		Name:           req.Name,
		MatchProcesses: req.MatchProcesses,
		Description:    req.Description,
		Status:         req.Status,
	}
}
