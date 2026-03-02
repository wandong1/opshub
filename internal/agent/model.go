package agent

import "time"

// AgentInfo Agent信息模型
type AgentInfo struct {
	ID         uint       `gorm:"primarykey" json:"id"`
	AgentID    string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"agentId"`
	HostID     uint       `gorm:"uniqueIndex;not null" json:"hostId"`
	Version    string     `gorm:"type:varchar(50)" json:"version"`
	Hostname   string     `gorm:"type:varchar(100)" json:"hostname"`
	OS         string     `gorm:"type:varchar(100)" json:"os"`
	Arch       string     `gorm:"type:varchar(50)" json:"arch"`
	Status     string     `gorm:"type:varchar(20);default:'installed'" json:"status"`
	LastSeen   *time.Time `json:"lastSeen,omitempty"`
	CertExpiry *time.Time `json:"certExpiry,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

func (AgentInfo) TableName() string {
	return "agent_info"
}
