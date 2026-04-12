package alert

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	alertdata "github.com/ydcloud-dy/opshub/internal/data/alert"
)

const (
	redisSilenceCacheKey   = "alert:silence:rules"     // Redis 缓存 key
	redisSilenceReloadChan = "alert:silence:reload"    // Pub/Sub 通道
	silenceCacheTTL        = 5 * time.Minute           // Redis 缓存 TTL
	localSilenceCacheTTL   = 30 * time.Second          // 本地缓存 TTL
)

// SilenceRuleCache 屏蔽规则缓存管理器
type SilenceRuleCache struct {
	rdb             *redis.Client
	silenceRuleRepo *alertdata.SilenceRuleRepo

	// 本地内存缓存
	mu       sync.RWMutex
	rules    []*biz.AlertSilenceRule
	lastLoad time.Time

	// Pub/Sub 订阅
	pubsub *redis.PubSub
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewSilenceRuleCache 创建屏蔽规则缓存管理器
func NewSilenceRuleCache(rdb *redis.Client, silenceRuleRepo *alertdata.SilenceRuleRepo) *SilenceRuleCache {
	return &SilenceRuleCache{
		rdb:             rdb,
		silenceRuleRepo: silenceRuleRepo,
		stopCh:          make(chan struct{}),
	}
}

// Start 启动缓存管理器（订阅 Pub/Sub）
func (c *SilenceRuleCache) Start(ctx context.Context) {
	c.wg.Add(1)
	defer c.wg.Done()

	appLogger.Info("屏蔽规则缓存管理器启动，开始订阅屏蔽规则重载事件")

	// 订阅 Pub/Sub 通道
	c.pubsub = c.rdb.Subscribe(ctx, redisSilenceReloadChan)
	defer c.pubsub.Close()

	ch := c.pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			appLogger.Info("屏蔽规则缓存管理器停止（context 取消）")
			return
		case <-c.stopCh:
			appLogger.Info("屏蔽规则缓存管理器停止（主动停止）")
			return
		case msg := <-ch:
			if msg != nil && msg.Payload == "reload" {
				appLogger.Info("收到屏蔽规则重载通知，清空本地缓存")
				c.InvalidateCache()
			}
		}
	}
}

// Stop 停止缓存管理器
func (c *SilenceRuleCache) Stop() {
	close(c.stopCh)
	c.wg.Wait()
	appLogger.Info("屏蔽规则缓存管理器已停止")
}

// GetActiveRules 获取活跃的屏蔽规则列表（优先本地缓存）
func (c *SilenceRuleCache) GetActiveRules(ctx context.Context) ([]*biz.AlertSilenceRule, error) {
	// 1. 尝试从本地内存缓存读取
	c.mu.RLock()
	if c.rules != nil && time.Since(c.lastLoad) < localSilenceCacheTTL {
		rules := c.rules
		c.mu.RUnlock()
		appLogger.Debug("从本地缓存读取屏蔽规则列表", zap.Int("count", len(rules)))
		return rules, nil
	}
	c.mu.RUnlock()

	// 2. 本地缓存失效，从 Redis 读取
	rules, err := c.loadFromRedis(ctx)
	if err == nil {
		c.mu.Lock()
		c.rules = rules
		c.lastLoad = time.Now()
		c.mu.Unlock()
		appLogger.Info("从 Redis 缓存读取屏蔽规则列表", zap.Int("count", len(rules)))
		return rules, nil
	}

	// 3. Redis 未命中，从 MySQL 读取并写入 Redis
	appLogger.Warn("Redis 缓存未命中，从 MySQL 加载屏蔽规则列表", zap.Error(err))
	rules, err = c.loadFromMySQL(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.rules = rules
	c.lastLoad = time.Now()
	c.mu.Unlock()

	return rules, nil
}

// InvalidateCache 使缓存失效（清空本地缓存）
func (c *SilenceRuleCache) InvalidateCache() {
	c.mu.Lock()
	c.rules = nil
	c.lastLoad = time.Time{}
	c.mu.Unlock()
	appLogger.Debug("屏蔽规则本地缓存已清空")
}

// PublishReloadEvent 发布屏蔽规则重载事件
func (c *SilenceRuleCache) PublishReloadEvent(ctx context.Context) error {
	// 先清空本地缓存
	c.InvalidateCache()

	// 发布 Pub/Sub 事件
	err := c.rdb.Publish(ctx, redisSilenceReloadChan, "reload").Err()
	if err != nil {
		return err
	}

	appLogger.Info("已发布屏蔽规则重载事件")
	return nil
}

// loadFromRedis 从 Redis 加载屏蔽规则列表
func (c *SilenceRuleCache) loadFromRedis(ctx context.Context) ([]*biz.AlertSilenceRule, error) {
	val, err := c.rdb.Get(ctx, redisSilenceCacheKey).Result()
	if err == redis.Nil {
		return nil, err
	} else if err != nil {
		return nil, err
	}

	var rules []*biz.AlertSilenceRule
	if err := json.Unmarshal([]byte(val), &rules); err != nil {
		return nil, err
	}

	return rules, nil
}

// loadFromMySQL 从 MySQL 加载屏蔽规则列表并写入 Redis
func (c *SilenceRuleCache) loadFromMySQL(ctx context.Context) ([]*biz.AlertSilenceRule, error) {
	// 从数据库查询活跃的屏蔽规则
	rules, err := c.silenceRuleRepo.ListActiveRules(ctx)
	if err != nil {
		return nil, err
	}

	appLogger.Info("从 MySQL 加载屏蔽规则列表", zap.Int("count", len(rules)))

	// 写入 Redis 缓存
	if err := c.saveToRedis(ctx, rules); err != nil {
		appLogger.Warn("写入 Redis 缓存失败", zap.Error(err))
		// 写入失败不影响返回结果
	}

	return rules, nil
}

// saveToRedis 保存屏蔽规则列表到 Redis
func (c *SilenceRuleCache) saveToRedis(ctx context.Context, rules []*biz.AlertSilenceRule) error {
	data, err := json.Marshal(rules)
	if err != nil {
		return err
	}

	err = c.rdb.Set(ctx, redisSilenceCacheKey, data, silenceCacheTTL).Err()
	if err != nil {
		return err
	}

	appLogger.Debug("屏蔽规则列表已写入 Redis 缓存", zap.Int("count", len(rules)))
	return nil
}
