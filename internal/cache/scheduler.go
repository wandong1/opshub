package cache

import (
	"context"
	"time"

	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
)

// CacheSyncScheduler 缓存同步调度器
type CacheSyncScheduler struct {
	manager  *CacheManager
	stopChan chan struct{}
}

// NewCacheSyncScheduler 创建缓存同步调度器
func NewCacheSyncScheduler(manager *CacheManager) *CacheSyncScheduler {
	return &CacheSyncScheduler{
		manager:  manager,
		stopChan: make(chan struct{}),
	}
}

// Start 启动定期同步任务
func (s *CacheSyncScheduler) Start() {
	appLogger.Info("启动缓存同步调度器")

	// 启动时立即同步一次
	go s.syncOnce()

	// 定期清理孤儿缓存（每 10 分钟）
	go s.cleanupLoop(10 * time.Minute)

	// 定期检查缓存健康度（每 5 分钟）
	go s.healthCheckLoop(5 * time.Minute)
}

// Stop 停止调度器
func (s *CacheSyncScheduler) Stop() {
	appLogger.Info("停止缓存同步调度器")
	close(s.stopChan)
}

// syncOnce 执行一次同步
func (s *CacheSyncScheduler) syncOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.manager.SyncAgentStatusFromDB(ctx); err != nil {
		appLogger.Error("同步 Agent 状态失败", zap.Error(err))
	}
}

// cleanupLoop 定期清理孤儿缓存
func (s *CacheSyncScheduler) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := s.manager.CleanExpiredCache(ctx); err != nil {
				appLogger.Error("清理过期缓存失败", zap.Error(err))
			}
			cancel()

		case <-s.stopChan:
			return
		}
	}
}

// healthCheckLoop 定期检查缓存健康度
func (s *CacheSyncScheduler) healthCheckLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			s.checkCacheHealth(ctx)
			cancel()

		case <-s.stopChan:
			return
		}
	}
}

// checkCacheHealth 检查缓存健康度
func (s *CacheSyncScheduler) checkCacheHealth(ctx context.Context) {
	cache := s.manager.GetAgentCache()

	// 检查在线 Agent 数量
	onlineAgents, err := cache.GetOnlineAgents(ctx)
	if err != nil {
		appLogger.Warn("获取在线 Agent 列表失败", zap.Error(err))
		return
	}

	appLogger.Debug("缓存健康检查",
		zap.Int("onlineAgents", len(onlineAgents)))

	// 可以添加更多健康检查指标
	// 例如：缓存命中率、响应时间等
}

// WarmupCache 预热缓存（应用启动时调用）
func (s *CacheSyncScheduler) WarmupCache() error {
	appLogger.Info("开始预热缓存")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 同步所有 Agent 状态
	if err := s.manager.SyncAgentStatusFromDB(ctx); err != nil {
		return err
	}

	appLogger.Info("缓存预热完成")
	return nil
}
