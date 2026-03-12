package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	agentmodel "github.com/ydcloud-dy/opshub/internal/agent"
	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
)

const (
	// Agent 状态缓存 key 前缀
	AgentStatusPrefix = "agent:status:"     // agent:status:{agent_id}
	AgentListKey      = "agent:list"        // 所有 Agent ID 列表（Set）
	AgentOnlineKey    = "agent:online"      // 在线 Agent ID 列表（Set）

	// 主机信息缓存 key 前缀
	HostInfoPrefix    = "host:info:"        // host:info:{host_id}
	HostMetricsPrefix = "host:metrics:"     // host:metrics:{host_id}
	HostListKey       = "host:list"         // 所有主机 ID 列表（Set）

	// 批量查询缓存
	HostBatchPrefix   = "host:batch:"       // host:batch:{group_id}

	// TTL 配置
	AgentStatusTTL    = 3 * time.Minute     // Agent 状态缓存 3 分钟
	HostInfoTTL       = 10 * time.Minute    // 主机基本信息缓存 10 分钟
	HostMetricsTTL    = 1 * time.Minute     // 主机监控指标缓存 1 分钟
	BatchQueryTTL     = 30 * time.Second    // 批量查询缓存 30 秒
)

// AgentStatusCache Agent 状态缓存
type AgentStatusCache struct {
	Status    string    `json:"status"`     // online/offline/installed
	LastSeen  time.Time `json:"last_seen"`
	Version   string    `json:"version"`
	Hostname  string    `json:"hostname"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HostInfoCache 主机基本信息缓存
type HostInfoCache struct {
	ID             uint      `json:"id"`
	Name           string    `json:"name"`
	IP             string    `json:"ip"`
	Port           int       `json:"port"`
	AgentID        string    `json:"agent_id"`
	AgentStatus    string    `json:"agent_status"`
	ConnectionMode string    `json:"connection_mode"`
	OS             string    `json:"os"`
	Arch           string    `json:"arch"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// HostMetricsCache 主机监控指标缓存
type HostMetricsCache struct {
	HostID       uint      `json:"host_id"`
	CPUUsage     float64   `json:"cpu_usage"`
	MemoryUsage  float64   `json:"memory_usage"`
	DiskUsage    float64   `json:"disk_usage"`
	NetworkIn    uint64    `json:"network_in"`
	NetworkOut   uint64    `json:"network_out"`
	CollectedAt  time.Time `json:"collected_at"`
}

// AgentCache Agent 缓存管理器
type AgentCache struct {
	rdb *redis.Client
}

// NewAgentCache 创建 Agent 缓存管理器
func NewAgentCache(rdb *redis.Client) *AgentCache {
	return &AgentCache{rdb: rdb}
}

// SetAgentStatus 设置 Agent 状态（同时更新在线列表）
func (c *AgentCache) SetAgentStatus(ctx context.Context, agentID string, status *AgentStatusCache) error {
	key := AgentStatusPrefix + agentID

	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	pipe := c.rdb.Pipeline()

	// 设置状态缓存
	pipe.Set(ctx, key, data, AgentStatusTTL)

	// 更新 Agent 列表
	pipe.SAdd(ctx, AgentListKey, agentID)

	// 更新在线/离线列表
	if status.Status == "online" {
		pipe.SAdd(ctx, AgentOnlineKey, agentID)
	} else {
		pipe.SRem(ctx, AgentOnlineKey, agentID)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// GetAgentStatus 获取 Agent 状态
func (c *AgentCache) GetAgentStatus(ctx context.Context, agentID string) (*AgentStatusCache, error) {
	key := AgentStatusPrefix + agentID

	data, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // 缓存未命中
	}
	if err != nil {
		return nil, err
	}

	var status AgentStatusCache
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return nil, fmt.Errorf("反序列化失败: %w", err)
	}

	return &status, nil
}

// BatchGetAgentStatus 批量获取 Agent 状态
func (c *AgentCache) BatchGetAgentStatus(ctx context.Context, agentIDs []string) (map[string]*AgentStatusCache, error) {
	if len(agentIDs) == 0 {
		return make(map[string]*AgentStatusCache), nil
	}

	pipe := c.rdb.Pipeline()
	cmds := make(map[string]*redis.StringCmd)

	for _, agentID := range agentIDs {
		key := AgentStatusPrefix + agentID
		cmds[agentID] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	result := make(map[string]*AgentStatusCache)
	for agentID, cmd := range cmds {
		data, err := cmd.Result()
		if err == redis.Nil {
			continue // 缓存未命中，跳过
		}
		if err != nil {
			appLogger.Warn("获取 Agent 状态失败", zap.String("agentID", agentID), zap.Error(err))
			continue
		}

		var status AgentStatusCache
		if err := json.Unmarshal([]byte(data), &status); err != nil {
			appLogger.Warn("反序列化 Agent 状态失败", zap.String("agentID", agentID), zap.Error(err))
			continue
		}

		result[agentID] = &status
	}

	return result, nil
}

// GetOnlineAgents 获取所有在线 Agent ID 列表
func (c *AgentCache) GetOnlineAgents(ctx context.Context) ([]string, error) {
	members, err := c.rdb.SMembers(ctx, AgentOnlineKey).Result()
	if err != nil {
		return nil, err
	}
	return members, nil
}

// DeleteAgentStatus 删除 Agent 状态缓存
func (c *AgentCache) DeleteAgentStatus(ctx context.Context, agentID string) error {
	key := AgentStatusPrefix + agentID

	pipe := c.rdb.Pipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, AgentListKey, agentID)
	pipe.SRem(ctx, AgentOnlineKey, agentID)

	_, err := pipe.Exec(ctx)
	return err
}

// SetHostInfo 设置主机基本信息
func (c *AgentCache) SetHostInfo(ctx context.Context, hostID uint, info *HostInfoCache) error {
	key := fmt.Sprintf("%s%d", HostInfoPrefix, hostID)

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	pipe := c.rdb.Pipeline()
	pipe.Set(ctx, key, data, HostInfoTTL)
	pipe.SAdd(ctx, HostListKey, hostID)

	_, err = pipe.Exec(ctx)
	return err
}

// GetHostInfo 获取主机基本信息
func (c *AgentCache) GetHostInfo(ctx context.Context, hostID uint) (*HostInfoCache, error) {
	key := fmt.Sprintf("%s%d", HostInfoPrefix, hostID)

	data, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var info HostInfoCache
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, fmt.Errorf("反序列化失败: %w", err)
	}

	return &info, nil
}

// BatchGetHostInfo 批量获取主机信息
func (c *AgentCache) BatchGetHostInfo(ctx context.Context, hostIDs []uint) (map[uint]*HostInfoCache, error) {
	if len(hostIDs) == 0 {
		return make(map[uint]*HostInfoCache), nil
	}

	pipe := c.rdb.Pipeline()
	cmds := make(map[uint]*redis.StringCmd)

	for _, hostID := range hostIDs {
		key := fmt.Sprintf("%s%d", HostInfoPrefix, hostID)
		cmds[hostID] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	result := make(map[uint]*HostInfoCache)
	for hostID, cmd := range cmds {
		data, err := cmd.Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			appLogger.Warn("获取主机信息失败", zap.Uint("hostID", hostID), zap.Error(err))
			continue
		}

		var info HostInfoCache
		if err := json.Unmarshal([]byte(data), &info); err != nil {
			appLogger.Warn("反序列化主机信息失败", zap.Uint("hostID", hostID), zap.Error(err))
			continue
		}

		result[hostID] = &info
	}

	return result, nil
}

// SetHostMetrics 设置主机监控指标
func (c *AgentCache) SetHostMetrics(ctx context.Context, hostID uint, metrics *HostMetricsCache) error {
	key := fmt.Sprintf("%s%d", HostMetricsPrefix, hostID)

	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	return c.rdb.Set(ctx, key, data, HostMetricsTTL).Err()
}

// GetHostMetrics 获取主机监控指标
func (c *AgentCache) GetHostMetrics(ctx context.Context, hostID uint) (*HostMetricsCache, error) {
	key := fmt.Sprintf("%s%d", HostMetricsPrefix, hostID)

	data, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var metrics HostMetricsCache
	if err := json.Unmarshal([]byte(data), &metrics); err != nil {
		return nil, fmt.Errorf("反序列化失败: %w", err)
	}

	return &metrics, nil
}

// InvalidateHostCache 使主机缓存失效
func (c *AgentCache) InvalidateHostCache(ctx context.Context, hostID uint) error {
	infoKey := fmt.Sprintf("%s%d", HostInfoPrefix, hostID)
	metricsKey := fmt.Sprintf("%s%d", HostMetricsPrefix, hostID)

	pipe := c.rdb.Pipeline()
	pipe.Del(ctx, infoKey)
	pipe.Del(ctx, metricsKey)
	pipe.SRem(ctx, HostListKey, hostID)

	_, err := pipe.Exec(ctx)
	return err
}

// ConvertAgentInfoToCache 将 AgentInfo 转换为缓存对象
func ConvertAgentInfoToCache(info *agentmodel.AgentInfo) *AgentStatusCache {
	return &AgentStatusCache{
		Status:    info.Status,
		LastSeen:  *info.LastSeen,
		Version:   info.Version,
		Hostname:  info.Hostname,
		OS:        info.OS,
		Arch:      info.Arch,
		UpdatedAt: info.UpdatedAt,
	}
}

// ConvertHostToCache 将 Host 转换为缓存对象
func ConvertHostToCache(host *asset.Host) *HostInfoCache {
	return &HostInfoCache{
		ID:             host.ID,
		Name:           host.Name,
		IP:             host.IP,
		Port:           host.Port,
		AgentID:        host.AgentID,
		AgentStatus:    host.AgentStatus,
		ConnectionMode: host.ConnectionMode,
		OS:             host.OS,
		Arch:           host.Arch,
		UpdatedAt:      host.UpdatedAt,
	}
}
