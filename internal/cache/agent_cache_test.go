package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	agentmodel "github.com/ydcloud-dy/opshub/internal/agent"
	"github.com/ydcloud-dy/opshub/internal/biz/asset"
)

// setupTestRedis 创建测试用的 Redis 实例
func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("创建 miniredis 失败: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, mr
}

func TestAgentCache_SetAndGetAgentStatus(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewAgentCache(client)
	ctx := context.Background()

	// 测试数据
	agentID := "test-agent-001"
	status := &AgentStatusCache{
		Status:    "online",
		LastSeen:  time.Now(),
		Version:   "1.0.0",
		Hostname:  "test-host",
		OS:        "linux",
		Arch:      "amd64",
		UpdatedAt: time.Now(),
	}

	// 设置状态
	err := cache.SetAgentStatus(ctx, agentID, status)
	assert.NoError(t, err)

	// 获取状态
	cached, err := cache.GetAgentStatus(ctx, agentID)
	assert.NoError(t, err)
	assert.NotNil(t, cached)
	assert.Equal(t, status.Status, cached.Status)
	assert.Equal(t, status.Version, cached.Version)
	assert.Equal(t, status.Hostname, cached.Hostname)

	// 验证在线列表
	onlineAgents, err := cache.GetOnlineAgents(ctx)
	assert.NoError(t, err)
	assert.Contains(t, onlineAgents, agentID)
}

func TestAgentCache_BatchGetAgentStatus(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewAgentCache(client)
	ctx := context.Background()

	// 准备测试数据
	agentIDs := []string{"agent-001", "agent-002", "agent-003"}
	for i, agentID := range agentIDs {
		status := &AgentStatusCache{
			Status:    "online",
			LastSeen:  time.Now(),
			Version:   "1.0.0",
			Hostname:  "host-" + string(rune('0'+i)),
			OS:        "linux",
			Arch:      "amd64",
			UpdatedAt: time.Now(),
		}
		err := cache.SetAgentStatus(ctx, agentID, status)
		assert.NoError(t, err)
	}

	// 批量获取
	statuses, err := cache.BatchGetAgentStatus(ctx, agentIDs)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(statuses))

	for _, agentID := range agentIDs {
		assert.Contains(t, statuses, agentID)
		assert.Equal(t, "online", statuses[agentID].Status)
	}
}

func TestAgentCache_DeleteAgentStatus(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewAgentCache(client)
	ctx := context.Background()

	agentID := "test-agent-001"
	status := &AgentStatusCache{
		Status:   "online",
		LastSeen: time.Now(),
	}

	// 设置状态
	cache.SetAgentStatus(ctx, agentID, status)

	// 删除状态
	err := cache.DeleteAgentStatus(ctx, agentID)
	assert.NoError(t, err)

	// 验证已删除
	cached, err := cache.GetAgentStatus(ctx, agentID)
	assert.NoError(t, err)
	assert.Nil(t, cached)

	// 验证不在在线列表中
	onlineAgents, err := cache.GetOnlineAgents(ctx)
	assert.NoError(t, err)
	assert.NotContains(t, onlineAgents, agentID)
}

func TestAgentCache_HostInfo(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewAgentCache(client)
	ctx := context.Background()

	hostID := uint(1)
	info := &HostInfoCache{
		ID:             hostID,
		Name:           "test-host",
		IP:             "192.168.1.100",
		Port:           22,
		AgentID:        "agent-001",
		AgentStatus:    "online",
		ConnectionMode: "agent",
		OS:             "linux",
		Arch:           "amd64",
		UpdatedAt:      time.Now(),
	}

	// 设置主机信息
	err := cache.SetHostInfo(ctx, hostID, info)
	assert.NoError(t, err)

	// 获取主机信息
	cached, err := cache.GetHostInfo(ctx, hostID)
	assert.NoError(t, err)
	assert.NotNil(t, cached)
	assert.Equal(t, info.Name, cached.Name)
	assert.Equal(t, info.IP, cached.IP)
	assert.Equal(t, info.AgentStatus, cached.AgentStatus)
}

func TestAgentCache_BatchGetHostInfo(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewAgentCache(client)
	ctx := context.Background()

	// 准备测试数据
	hostIDs := []uint{1, 2, 3}
	for _, hostID := range hostIDs {
		info := &HostInfoCache{
			ID:          hostID,
			Name:        "host-" + string(rune('0'+hostID)),
			IP:          "192.168.1." + string(rune('0'+hostID)),
			Port:        22,
			AgentStatus: "online",
			UpdatedAt:   time.Now(),
		}
		err := cache.SetHostInfo(ctx, hostID, info)
		assert.NoError(t, err)
	}

	// 批量获取
	infos, err := cache.BatchGetHostInfo(ctx, hostIDs)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(infos))

	for _, hostID := range hostIDs {
		assert.Contains(t, infos, hostID)
		assert.Equal(t, "online", infos[hostID].AgentStatus)
	}
}

func TestAgentCache_HostMetrics(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewAgentCache(client)
	ctx := context.Background()

	hostID := uint(1)
	metrics := &HostMetricsCache{
		HostID:      hostID,
		CPUUsage:    45.5,
		MemoryUsage: 60.2,
		DiskUsage:   75.8,
		NetworkIn:   1024000,
		NetworkOut:  512000,
		CollectedAt: time.Now(),
	}

	// 设置监控指标
	err := cache.SetHostMetrics(ctx, hostID, metrics)
	assert.NoError(t, err)

	// 获取监控指标
	cached, err := cache.GetHostMetrics(ctx, hostID)
	assert.NoError(t, err)
	assert.NotNil(t, cached)
	assert.Equal(t, metrics.CPUUsage, cached.CPUUsage)
	assert.Equal(t, metrics.MemoryUsage, cached.MemoryUsage)
	assert.Equal(t, metrics.DiskUsage, cached.DiskUsage)
}

func TestAgentCache_InvalidateHostCache(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewAgentCache(client)
	ctx := context.Background()

	hostID := uint(1)

	// 设置主机信息和监控指标
	info := &HostInfoCache{
		ID:   hostID,
		Name: "test-host",
	}
	metrics := &HostMetricsCache{
		HostID:   hostID,
		CPUUsage: 50.0,
	}

	cache.SetHostInfo(ctx, hostID, info)
	cache.SetHostMetrics(ctx, hostID, metrics)

	// 使缓存失效
	err := cache.InvalidateHostCache(ctx, hostID)
	assert.NoError(t, err)

	// 验证已失效
	cachedInfo, _ := cache.GetHostInfo(ctx, hostID)
	assert.Nil(t, cachedInfo)

	cachedMetrics, _ := cache.GetHostMetrics(ctx, hostID)
	assert.Nil(t, cachedMetrics)
}

func TestAgentCache_TTL(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewAgentCache(client)
	ctx := context.Background()

	agentID := "test-agent-001"
	status := &AgentStatusCache{
		Status:   "online",
		LastSeen: time.Now(),
	}

	// 设置状态
	cache.SetAgentStatus(ctx, agentID, status)

	// 检查 TTL
	key := AgentStatusPrefix + agentID
	ttl := client.TTL(ctx, key).Val()
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, AgentStatusTTL)

	// 快进时间（miniredis 支持）
	mr.FastForward(AgentStatusTTL + time.Second)

	// 验证已过期
	cached, err := cache.GetAgentStatus(ctx, agentID)
	assert.NoError(t, err)
	assert.Nil(t, cached)
}

func TestConvertAgentInfoToCache(t *testing.T) {
	now := time.Now()
	info := &agentmodel.AgentInfo{
		AgentID:  "agent-001",
		HostID:   1,
		Status:   "online",
		LastSeen: &now,
		Version:  "1.0.0",
		Hostname: "test-host",
		OS:       "linux",
		Arch:     "amd64",
	}

	cached := ConvertAgentInfoToCache(info)

	assert.Equal(t, info.Status, cached.Status)
	assert.Equal(t, *info.LastSeen, cached.LastSeen)
	assert.Equal(t, info.Version, cached.Version)
	assert.Equal(t, info.Hostname, cached.Hostname)
	assert.Equal(t, info.OS, cached.OS)
	assert.Equal(t, info.Arch, cached.Arch)
}

func TestConvertHostToCache(t *testing.T) {
	host := &asset.Host{
		Name:           "test-host",
		IP:             "192.168.1.100",
		Port:           22,
		AgentID:        "agent-001",
		AgentStatus:    "online",
		ConnectionMode: "agent",
		OS:             "linux",
		Arch:           "amd64",
	}
	host.ID = 1

	cached := ConvertHostToCache(host)

	assert.Equal(t, host.ID, cached.ID)
	assert.Equal(t, host.Name, cached.Name)
	assert.Equal(t, host.IP, cached.IP)
	assert.Equal(t, host.Port, cached.Port)
	assert.Equal(t, host.AgentID, cached.AgentID)
	assert.Equal(t, host.AgentStatus, cached.AgentStatus)
	assert.Equal(t, host.ConnectionMode, cached.ConnectionMode)
}

// 基准测试
func BenchmarkAgentCache_SetAgentStatus(b *testing.B) {
	client, mr := setupTestRedis(&testing.T{})
	defer mr.Close()

	cache := NewAgentCache(client)
	ctx := context.Background()

	status := &AgentStatusCache{
		Status:   "online",
		LastSeen: time.Now(),
		Version:  "1.0.0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agentID := "agent-" + string(rune(i))
		cache.SetAgentStatus(ctx, agentID, status)
	}
}

func BenchmarkAgentCache_GetAgentStatus(b *testing.B) {
	client, mr := setupTestRedis(&testing.T{})
	defer mr.Close()

	cache := NewAgentCache(client)
	ctx := context.Background()

	// 准备数据
	agentID := "test-agent"
	status := &AgentStatusCache{
		Status:   "online",
		LastSeen: time.Now(),
	}
	cache.SetAgentStatus(ctx, agentID, status)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetAgentStatus(ctx, agentID)
	}
}

func BenchmarkAgentCache_BatchGetAgentStatus(b *testing.B) {
	client, mr := setupTestRedis(&testing.T{})
	defer mr.Close()

	cache := NewAgentCache(client)
	ctx := context.Background()

	// 准备 100 个 Agent
	agentIDs := make([]string, 100)
	for i := 0; i < 100; i++ {
		agentID := "agent-" + string(rune(i))
		agentIDs[i] = agentID
		status := &AgentStatusCache{
			Status:   "online",
			LastSeen: time.Now(),
		}
		cache.SetAgentStatus(ctx, agentID, status)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.BatchGetAgentStatus(ctx, agentIDs)
	}
}
