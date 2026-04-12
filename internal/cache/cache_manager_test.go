package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	agentmodel "github.com/ydcloud-dy/opshub/internal/agent"
	agentrepo "github.com/ydcloud-dy/opshub/internal/data/agent"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestEnv 设置测试环境
func setupTestEnv(t *testing.T) (*CacheManager, *redis.Client, *gorm.DB, *miniredis.Miniredis) {
	// 创建 miniredis
	mr, err := miniredis.Run()
	assert.NoError(t, err)

	// 创建 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 自动迁移
	err = db.AutoMigrate(&agentmodel.AgentInfo{})
	assert.NoError(t, err)

	// 创建 Repository
	agentRepo := agentrepo.NewRepository(db)

	// 创建 CacheManager
	cfg := &CacheConfig{
		BatchFlushInterval: 1 * time.Second, // 测试用短间隔
		BatchSize:          10,
		BatchQueueMaxSize:  100,
		RedisTTL:           10 * time.Second,
		LockTimeout:        5 * time.Second,
		OfflineThreshold:   10 * time.Second,
	}

	manager := NewCacheManager(rdb, agentRepo, db, cfg)

	return manager, rdb, db, mr
}

// TestUpdateAgentStatus_StatusChange 测试状态变化立即写入
func TestUpdateAgentStatus_StatusChange(t *testing.T) {
	manager, rdb, db, mr := setupTestEnv(t)
	defer mr.Close()
	defer rdb.Close()

	ctx := context.Background()
	agentID := "test-agent-001"

	// 1. 首次注册（online）
	err := manager.UpdateAgentStatus(ctx, agentID, map[string]any{
		"status":    "online",
		"last_seen": time.Now(),
	})
	assert.NoError(t, err)

	// 验证 Redis 已更新
	result, err := rdb.HGet(ctx, "agent:status:"+agentID, "status").Result()
	assert.NoError(t, err)
	assert.Equal(t, "online", result)

	// 验证 MySQL 已更新（状态变化应立即写入）
	var agentInfo agentmodel.AgentInfo
	err = db.Where("agent_id = ?", agentID).First(&agentInfo).Error
	assert.NoError(t, err)
	assert.Equal(t, "online", agentInfo.Status)

	// 2. 状态变化（online → offline）
	time.Sleep(100 * time.Millisecond)
	err = manager.UpdateAgentStatus(ctx, agentID, map[string]any{
		"status":    "offline",
		"last_seen": time.Now(),
	})
	assert.NoError(t, err)

	// 验证 Redis 已更新
	result, err = rdb.HGet(ctx, "agent:status:"+agentID, "status").Result()
	assert.NoError(t, err)
	assert.Equal(t, "offline", result)

	// 验证 MySQL 已更新
	err = db.Where("agent_id = ?", agentID).First(&agentInfo).Error
	assert.NoError(t, err)
	assert.Equal(t, "offline", agentInfo.Status)
}

// TestUpdateAgentStatus_NoStatusChange 测试状态未变化加入队列
func TestUpdateAgentStatus_NoStatusChange(t *testing.T) {
	manager, rdb, db, mr := setupTestEnv(t)
	defer mr.Close()
	defer rdb.Close()

	ctx := context.Background()
	agentID := "test-agent-002"

	// 1. 首次注册
	err := manager.UpdateAgentStatus(ctx, agentID, map[string]any{
		"status":    "online",
		"last_seen": time.Now(),
	})
	assert.NoError(t, err)

	// 2. 状态未变化（online → online）
	time.Sleep(100 * time.Millisecond)
	lastSeen1 := time.Now()
	err = manager.UpdateAgentStatus(ctx, agentID, map[string]any{
		"status":    "online",
		"last_seen": lastSeen1,
	})
	assert.NoError(t, err)

	// 验证 Redis 已更新
	result, err := rdb.HGet(ctx, "agent:status:"+agentID, "last_seen").Result()
	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// 验证已加入批量队列
	queueLen, err := rdb.LLen(ctx, BatchQueueKey).Result()
	assert.NoError(t, err)
	assert.Greater(t, queueLen, int64(0))

	// 验证在去重集合中
	exists, err := rdb.SIsMember(ctx, BatchPendingKey, agentID).Result()
	assert.NoError(t, err)
	assert.True(t, exists)
}

// TestBatchWorker_ProcessBatch 测试批量同步
func TestBatchWorker_ProcessBatch(t *testing.T) {
	manager, rdb, db, mr := setupTestEnv(t)
	defer mr.Close()
	defer rdb.Close()

	ctx := context.Background()

	// 创建多个 Agent
	agentIDs := []string{"agent-001", "agent-002", "agent-003"}
	for _, agentID := range agentIDs {
		// 首次注册
		err := manager.UpdateAgentStatus(ctx, agentID, map[string]any{
			"status":    "online",
			"last_seen": time.Now(),
		})
		assert.NoError(t, err)

		// 心跳更新（状态未变化）
		time.Sleep(50 * time.Millisecond)
		err = manager.UpdateAgentStatus(ctx, agentID, map[string]any{
			"status":    "online",
			"last_seen": time.Now(),
		})
		assert.NoError(t, err)
	}

	// 验证队列中有数据
	queueLen, err := rdb.LLen(ctx, BatchQueueKey).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(len(agentIDs)), queueLen)

	// 启动 BatchWorker
	manager.StartBatchWorker()
	defer manager.StopBatchWorker()

	// 等待批量同步完成
	time.Sleep(2 * time.Second)

	// 验证队列已清空
	queueLen, err = rdb.LLen(ctx, BatchQueueKey).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), queueLen)

	// 验证 MySQL 已更新
	for _, agentID := range agentIDs {
		var agentInfo agentmodel.AgentInfo
		err = db.Where("agent_id = ?", agentID).First(&agentInfo).Error
		assert.NoError(t, err)
		assert.Equal(t, "online", agentInfo.Status)
		assert.NotNil(t, agentInfo.LastSeen)
	}
}

// TestLuaScript_EnqueueDedup 测试入队去重
func TestLuaScript_EnqueueDedup(t *testing.T) {
	manager, rdb, _, mr := setupTestEnv(t)
	defer mr.Close()
	defer rdb.Close()

	ctx := context.Background()
	agentID := "test-agent-003"

	// 第一次入队
	err := manager.enqueueToRedis(ctx, agentID)
	assert.NoError(t, err)

	// 验证队列长度
	queueLen, err := rdb.LLen(ctx, BatchQueueKey).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), queueLen)

	// 第二次入队（应该被去重）
	err = manager.enqueueToRedis(ctx, agentID)
	assert.NoError(t, err)

	// 验证队列长度不变
	queueLen, err = rdb.LLen(ctx, BatchQueueKey).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), queueLen)
}

// TestRedisFallback 测试 Redis 故障降级
func TestRedisFallback(t *testing.T) {
	manager, rdb, db, mr := setupTestEnv(t)
	defer rdb.Close()

	ctx := context.Background()
	agentID := "test-agent-004"

	// 关闭 Redis
	mr.Close()

	// 更新状态（应该降级到直接写 MySQL）
	err := manager.UpdateAgentStatus(ctx, agentID, map[string]any{
		"status":    "online",
		"last_seen": time.Now(),
	})
	assert.NoError(t, err)

	// 验证 MySQL 已更新
	var agentInfo agentmodel.AgentInfo
	err = db.Where("agent_id = ?", agentID).First(&agentInfo).Error
	assert.NoError(t, err)
	assert.Equal(t, "online", agentInfo.Status)
}
