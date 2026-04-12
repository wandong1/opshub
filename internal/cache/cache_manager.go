package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	agentmodel "github.com/ydcloud-dy/opshub/internal/agent"
	agentrepo "github.com/ydcloud-dy/opshub/internal/data/agent"
	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CacheManager 缓存管理器，负责 Redis 与 MySQL 之间的数据一致性
type CacheManager struct {
	cache       *AgentCache
	agentRepo   *agentrepo.Repository
	db          *gorm.DB
	rdb         *redis.Client
	cfg         *CacheConfig
	metrics     *Metrics
	scripts     *LuaScripts
	batchWorker *BatchWorker
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(rdb *redis.Client, agentRepo *agentrepo.Repository, db *gorm.DB, cfg *CacheConfig) *CacheManager {
	if cfg == nil {
		cfg = DefaultCacheConfig()
	}

	metrics := NewMetrics()
	scripts := NewLuaScripts()

	manager := &CacheManager{
		cache:     NewAgentCache(rdb),
		agentRepo: agentRepo,
		db:        db,
		rdb:       rdb,
		cfg:       cfg,
		metrics:   metrics,
		scripts:   scripts,
	}

	// 创建批量同步 Worker
	manager.batchWorker = NewBatchWorker(rdb, db, agentRepo, cfg, metrics, scripts)

	return manager
}

// StartBatchWorker 启动批量同步 Worker
func (m *CacheManager) StartBatchWorker() {
	if m.batchWorker != nil {
		m.batchWorker.Start()
	}
}

// StopBatchWorker 停止批量同步 Worker
func (m *CacheManager) StopBatchWorker() {
	if m.batchWorker != nil {
		m.batchWorker.Stop()
	}
}

// GetAgentCache 获取 Agent 缓存实例
func (m *CacheManager) GetAgentCache() *AgentCache {
	return m.cache
}

// UpdateAgentStatus 更新 Agent 状态（混合策略）
// 策略：状态变化立即写 MySQL，状态未变化加入批量队列
func (m *CacheManager) UpdateAgentStatus(ctx context.Context, agentID string, updates map[string]any) error {
	// 1. 使用 Lua 脚本原子更新 Redis 并检测状态变化
	statusChanged, err := m.updateRedisAndDetectChange(ctx, agentID, updates)
	if err != nil {
		// Redis 故障降级：直接写 MySQL
		appLogger.Warn("Redis 故障，降级到直接写 MySQL",
			zap.String("agentID", agentID),
			zap.Error(err))
		m.metrics.RedisFallbackCount.Inc()
		return m.agentRepo.UpdateInfo(ctx, agentID, updates)
	}

	// 2. 状态变化 → 立即写 MySQL
	if statusChanged {
		appLogger.Info("检测到状态变化，立即同步 MySQL",
			zap.String("agentID", agentID),
			zap.Any("updates", updates))
		m.metrics.ImmediateWriteCount.Inc()
		return m.agentRepo.UpdateInfo(ctx, agentID, updates)
	}

	// 3. 状态未变化 → 加入 Redis 批量队列
	if err := m.enqueueToRedis(ctx, agentID); err != nil {
		appLogger.Warn("加入批量队列失败",
			zap.String("agentID", agentID),
			zap.Error(err))
		// 入队失败不影响主流程（下次心跳会重试）
	}

	return nil
}

// updateRedisAndDetectChange 使用 Lua 脚本原子更新 Redis 并检测状态变化
func (m *CacheManager) updateRedisAndDetectChange(ctx context.Context, agentID string, updates map[string]any) (bool, error) {
	key := fmt.Sprintf("agent:status:%s", agentID)

	// 提取 status 和 last_seen
	status, ok := updates["status"].(string)
	if !ok {
		status = "online" // 默认值
	}

	lastSeen, ok := updates["last_seen"].(time.Time)
	if !ok {
		lastSeen = time.Now()
	}

	// 执行 Lua 脚本
	result, err := m.scripts.UpdateAndDetectChange.Run(ctx, m.rdb,
		[]string{key},
		status,
		lastSeen.Unix(),
		int(m.cfg.RedisTTL.Seconds()),
	).Int()

	if err != nil {
		return false, err
	}

	// result: 0=未变化, 1=变化, 2=首次注册
	return result > 0, nil
}

// enqueueToRedis 加入 Redis 批量队列
func (m *CacheManager) enqueueToRedis(ctx context.Context, agentID string) error {
	result, err := m.scripts.EnqueueWithDedup.Run(ctx, m.rdb,
		[]string{BatchQueueKey, BatchPendingKey},
		agentID,
	).Int()

	if err != nil {
		return err
	}

	// result: -1=已存在, >0=队列长度
	if result == -1 {
		// 已在队列中，跳过
		return nil
	}

	// 更新队列长度指标
	m.metrics.BatchQueueSize.Set(float64(result))

	// 队列长度超过阈值，触发告警
	if result > m.cfg.BatchQueueMaxSize {
		appLogger.Warn("批量队列长度超过阈值",
			zap.Int("queueLen", result),
			zap.Int("maxSize", m.cfg.BatchQueueMaxSize))
		// TODO: 触发告警
	}

	return nil
}

// GetAgentStatusWithFallback 获取 Agent 状态（带降级）
// 策略：Cache-Aside，先查 Redis，未命中再查 MySQL 并回写
func (m *CacheManager) GetAgentStatusWithFallback(ctx context.Context, agentID string) (*AgentStatusCache, error) {
	// 1. 尝试从 Redis 获取
	cached, err := m.cache.GetAgentStatus(ctx, agentID)
	if err != nil {
		appLogger.Warn("Redis 查询失败，降级到 MySQL", zap.String("agentID", agentID), zap.Error(err))
	} else if cached != nil {
		return cached, nil // 缓存命中
	}

	// 2. 缓存未命中，从 MySQL 查询
	agentInfo, err := m.agentRepo.GetByAgentID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	// 3. 回写 Redis 缓存
	cacheData := ConvertAgentInfoToCache(agentInfo)
	if err := m.cache.SetAgentStatus(ctx, agentID, cacheData); err != nil {
		appLogger.Warn("回写 Agent 缓存失败", zap.String("agentID", agentID), zap.Error(err))
	}

	return cacheData, nil
}

// BatchGetAgentStatusWithFallback 批量获取 Agent 状态（带降级）
func (m *CacheManager) BatchGetAgentStatusWithFallback(ctx context.Context, agentIDs []string) (map[string]*AgentStatusCache, error) {
	if len(agentIDs) == 0 {
		return make(map[string]*AgentStatusCache), nil
	}

	// 1. 批量从 Redis 获取
	cached, err := m.cache.BatchGetAgentStatus(ctx, agentIDs)
	if err != nil {
		appLogger.Warn("Redis 批量查询失败，降级到 MySQL", zap.Error(err))
		cached = make(map[string]*AgentStatusCache)
	}

	// 2. 找出缓存未命中的 agentID
	missedIDs := make([]string, 0)
	for _, agentID := range agentIDs {
		if _, ok := cached[agentID]; !ok {
			missedIDs = append(missedIDs, agentID)
		}
	}

	// 3. 如果有未命中的，从 MySQL 查询
	if len(missedIDs) > 0 {
		var agentInfos []*agentmodel.AgentInfo
		if err := m.db.Where("agent_id IN ?", missedIDs).Find(&agentInfos).Error; err != nil {
			appLogger.Error("批量查询 Agent 信息失败", zap.Error(err))
		} else {
			// 回写缓存
			for _, info := range agentInfos {
				cacheData := ConvertAgentInfoToCache(info)
				cached[info.AgentID] = cacheData

				// 异步回写 Redis
				go func(agentID string, data *AgentStatusCache) {
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					if err := m.cache.SetAgentStatus(ctx, agentID, data); err != nil {
						appLogger.Warn("回写 Agent 缓存失败", zap.String("agentID", agentID), zap.Error(err))
					}
				}(info.AgentID, cacheData)
			}
		}
	}

	return cached, nil
}

// UpdateHostInfo 更新主机信息（保证一致性）
func (m *CacheManager) UpdateHostInfo(ctx context.Context, hostID uint, updates map[string]any) error {
	// 1. 更新 MySQL
	if err := m.db.Model(&asset.Host{}).Where("id = ?", hostID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新数据库失败: %w", err)
	}

	// 2. 使缓存失效（延迟双删策略）
	if err := m.cache.InvalidateHostCache(ctx, hostID); err != nil {
		appLogger.Warn("使主机缓存失效失败", zap.Uint("hostID", hostID), zap.Error(err))
	}

	// 3. 延迟再次删除（防止脏读）
	go func() {
		time.Sleep(500 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := m.cache.InvalidateHostCache(ctx, hostID); err != nil {
			appLogger.Warn("延迟删除主机缓存失败", zap.Uint("hostID", hostID), zap.Error(err))
		}
	}()

	return nil
}

// GetHostInfoWithFallback 获取主机信息（带降级）
func (m *CacheManager) GetHostInfoWithFallback(ctx context.Context, hostID uint) (*HostInfoCache, error) {
	// 1. 尝试从 Redis 获取
	cached, err := m.cache.GetHostInfo(ctx, hostID)
	if err != nil {
		appLogger.Warn("Redis 查询失败，降级到 MySQL", zap.Uint("hostID", hostID), zap.Error(err))
	} else if cached != nil {
		return cached, nil
	}

	// 2. 从 MySQL 查询
	var host asset.Host
	if err := m.db.Where("id = ?", hostID).First(&host).Error; err != nil {
		return nil, err
	}

	// 3. 回写缓存
	cacheData := ConvertHostToCache(&host)
	if err := m.cache.SetHostInfo(ctx, hostID, cacheData); err != nil {
		appLogger.Warn("回写主机缓存失败", zap.Uint("hostID", hostID), zap.Error(err))
	}

	return cacheData, nil
}

// BatchGetHostInfoWithFallback 批量获取主机信息（带降级）
func (m *CacheManager) BatchGetHostInfoWithFallback(ctx context.Context, hostIDs []uint) (map[uint]*HostInfoCache, error) {
	if len(hostIDs) == 0 {
		return make(map[uint]*HostInfoCache), nil
	}

	// 1. 批量从 Redis 获取
	cached, err := m.cache.BatchGetHostInfo(ctx, hostIDs)
	if err != nil {
		appLogger.Warn("Redis 批量查询失败，降级到 MySQL", zap.Error(err))
		cached = make(map[uint]*HostInfoCache)
	}

	// 2. 找出缓存未命中的 hostID
	missedIDs := make([]uint, 0)
	for _, hostID := range hostIDs {
		if _, ok := cached[hostID]; !ok {
			missedIDs = append(missedIDs, hostID)
		}
	}

	// 3. 从 MySQL 查询未命中的数据
	if len(missedIDs) > 0 {
		var hosts []asset.Host
		if err := m.db.Where("id IN ?", missedIDs).Find(&hosts).Error; err != nil {
			appLogger.Error("批量查询主机信息失败", zap.Error(err))
		} else {
			// 回写缓存
			for i := range hosts {
				cacheData := ConvertHostToCache(&hosts[i])
				cached[hosts[i].ID] = cacheData

				// 异步回写 Redis
				go func(hostID uint, data *HostInfoCache) {
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					if err := m.cache.SetHostInfo(ctx, hostID, data); err != nil {
						appLogger.Warn("回写主机缓存失败", zap.Uint("hostID", hostID), zap.Error(err))
					}
				}(hosts[i].ID, cacheData)
			}
		}
	}

	return cached, nil
}

// SyncAgentStatusFromDB 从数据库同步 Agent 状态到 Redis（用于初始化或修复）
func (m *CacheManager) SyncAgentStatusFromDB(ctx context.Context) error {
	var agentInfos []*agentmodel.AgentInfo
	if err := m.db.Find(&agentInfos).Error; err != nil {
		return fmt.Errorf("查询 Agent 信息失败: %w", err)
	}

	successCount := 0
	for _, info := range agentInfos {
		cacheData := ConvertAgentInfoToCache(info)
		if err := m.cache.SetAgentStatus(ctx, info.AgentID, cacheData); err != nil {
			appLogger.Warn("同步 Agent 状态失败", zap.String("agentID", info.AgentID), zap.Error(err))
		} else {
			successCount++
		}
	}

	appLogger.Info("Agent 状态同步完成", zap.Int("total", len(agentInfos)), zap.Int("success", successCount))
	return nil
}

// CleanExpiredCache 清理过期缓存（定期任务）
func (m *CacheManager) CleanExpiredCache(ctx context.Context) error {
	// Redis 会自动清理过期 key，这里主要清理孤儿数据

	// 1. 获取 Redis 中的所有 Agent ID
	cachedAgentIDs, err := m.cache.rdb.SMembers(ctx, AgentListKey).Result()
	if err != nil {
		return fmt.Errorf("获取缓存 Agent 列表失败: %w", err)
	}

	// 2. 查询数据库中的所有 Agent ID
	var dbAgentIDs []string
	if err := m.db.Model(&agentmodel.AgentInfo{}).Pluck("agent_id", &dbAgentIDs).Error; err != nil {
		return fmt.Errorf("查询数据库 Agent 列表失败: %w", err)
	}

	// 3. 找出数据库中不存在的 Agent ID（孤儿缓存）
	dbAgentMap := make(map[string]bool)
	for _, id := range dbAgentIDs {
		dbAgentMap[id] = true
	}

	orphanCount := 0
	for _, cachedID := range cachedAgentIDs {
		if !dbAgentMap[cachedID] {
			if err := m.cache.DeleteAgentStatus(ctx, cachedID); err != nil {
				appLogger.Warn("删除孤儿 Agent 缓存失败", zap.String("agentID", cachedID), zap.Error(err))
			} else {
				orphanCount++
			}
		}
	}

	if orphanCount > 0 {
		appLogger.Info("清理孤儿 Agent 缓存", zap.Int("count", orphanCount))
	}

	return nil
}
