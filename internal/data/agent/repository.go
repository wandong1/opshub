package agent

import (
	"context"
	"time"

	agentmodel "github.com/ydcloud-dy/opshub/internal/agent"
	"gorm.io/gorm"
)

// Repository Agent信息仓库
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建Agent仓库
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建Agent记录
func (r *Repository) Create(ctx context.Context, info *agentmodel.AgentInfo) error {
	return r.db.WithContext(ctx).Create(info).Error
}

// GetByAgentID 根据AgentID查询
func (r *Repository) GetByAgentID(ctx context.Context, agentID string) (*agentmodel.AgentInfo, error) {
	var info agentmodel.AgentInfo
	if err := r.db.WithContext(ctx).Where("agent_id = ?", agentID).First(&info).Error; err != nil {
		return nil, err
	}
	return &info, nil
}

// GetByHostID 根据HostID查询
func (r *Repository) GetByHostID(ctx context.Context, hostID uint) (*agentmodel.AgentInfo, error) {
	var info agentmodel.AgentInfo
	if err := r.db.WithContext(ctx).Where("host_id = ?", hostID).First(&info).Error; err != nil {
		return nil, err
	}
	return &info, nil
}

// UpdateStatus 更新Agent状态
func (r *Repository) UpdateStatus(ctx context.Context, agentID string, status string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&agentmodel.AgentInfo{}).
		Where("agent_id = ?", agentID).
		Updates(map[string]interface{}{
			"status":    status,
			"last_seen": &now,
		}).Error
}

// UpdateInfo 更新Agent信息
func (r *Repository) UpdateInfo(ctx context.Context, agentID string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&agentmodel.AgentInfo{}).
		Where("agent_id = ?", agentID).
		Updates(updates).Error
}

// List 列出所有Agent
func (r *Repository) List(ctx context.Context) ([]agentmodel.AgentInfo, error) {
	var agents []agentmodel.AgentInfo
	if err := r.db.WithContext(ctx).Find(&agents).Error; err != nil {
		return nil, err
	}
	return agents, nil
}

// Delete 删除Agent记录
func (r *Repository) Delete(ctx context.Context, agentID string) error {
	return r.db.WithContext(ctx).Where("agent_id = ?", agentID).Delete(&agentmodel.AgentInfo{}).Error
}

// DeleteByHostID 根据HostID删除Agent记录
func (r *Repository) DeleteByHostID(ctx context.Context, hostID uint) error {
	return r.db.WithContext(ctx).Where("host_id = ?", hostID).Delete(&agentmodel.AgentInfo{}).Error
}
