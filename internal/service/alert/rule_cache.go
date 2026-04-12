package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"

	biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
	alertdata "github.com/ydcloud-dy/opshub/internal/data/alert"
)

const (
	redisRuleCacheKey   = "alert:rules:enabled"  // Redis 缓存 key
	redisRuleReloadChan = "alert:rules:reload"   // Pub/Sub 通道
	ruleCacheTTL        = 10 * time.Minute       // Redis 缓存 TTL
	localCacheTTL       = 5 * time.Second        // 本地缓存 TTL
)

// RuleCache 规则缓存管理器
type RuleCache struct {
	rdb      *redis.Client
	ruleRepo *alertdata.RuleRepo

	// 本地内存缓存
	mu       sync.RWMutex
	rules    []*biz.AlertRule
	lastLoad time.Time

	// Pub/Sub 订阅
	pubsub *redis.PubSub
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewRuleCache 创建规则缓存管理器
func NewRuleCache(rdb *redis.Client, ruleRepo *alertdata.RuleRepo) *RuleCache {
	return &RuleCache{
		rdb:      rdb,
		ruleRepo: ruleRepo,
		stopCh:   make(chan struct{}),
	}
}

// Start 启动缓存管理器（订阅 Pub/Sub）
func (c *RuleCache) Start(ctx context.Context) {
	c.wg.Add(1)
	defer c.wg.Done()

	appLogger.Info("规则缓存管理器启动，开始订阅规则重载事件")

	// 订阅 Pub/Sub 通道
	c.pubsub = c.rdb.Subscribe(ctx, redisRuleReloadChan)
	defer c.pubsub.Close()

	ch := c.pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			appLogger.Info("规则缓存管理器停止（context 取消）")
			return
		case <-c.stopCh:
			appLogger.Info("规则缓存管理器停止（主动停止）")
			return
		case msg := <-ch:
			if msg != nil && msg.Payload == "reload" {
				appLogger.Info("收到规则重载通知，清空本地缓存")
				c.InvalidateCache()
			}
		}
	}
}

// Stop 停止缓存管理器
func (c *RuleCache) Stop() {
	close(c.stopCh)
	c.wg.Wait()
	appLogger.Info("规则缓存管理器已停止")
}

// GetEnabledRules 获取启用的规则列表（优先本地缓存）
func (c *RuleCache) GetEnabledRules(ctx context.Context) ([]*biz.AlertRule, error) {
	// 1. 尝试从本地内存缓存读取
	c.mu.RLock()
	if c.rules != nil && time.Since(c.lastLoad) < localCacheTTL {
		rules := c.rules
		c.mu.RUnlock()
		appLogger.Debug("从本地缓存读取规则列表", zap.Int("count", len(rules)))
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
		appLogger.Info("从 Redis 缓存读取规则列表", zap.Int("count", len(rules)))
		return rules, nil
	}

	// 3. Redis 未命中，从 MySQL 读取并写入 Redis
	appLogger.Warn("Redis 缓存未命中，从 MySQL 加载规则列表", zap.Error(err))
	rules, err = c.loadFromMySQL(ctx)
	if err != nil {
		return nil, fmt.Errorf("从 MySQL 加载规则失败: %w", err)
	}

	c.mu.Lock()
	c.rules = rules
	c.lastLoad = time.Now()
	c.mu.Unlock()

	return rules, nil
}

// InvalidateCache 使缓存失效（清空本地缓存）
func (c *RuleCache) InvalidateCache() {
	c.mu.Lock()
	c.rules = nil
	c.lastLoad = time.Time{}
	c.mu.Unlock()
	appLogger.Debug("本地缓存已清空")
}

// PublishReloadEvent 发布规则重载事件
func (c *RuleCache) PublishReloadEvent(ctx context.Context) error {
	// 先清空本地缓存
	c.InvalidateCache()

	// 发布 Pub/Sub 事件
	err := c.rdb.Publish(ctx, redisRuleReloadChan, "reload").Err()
	if err != nil {
		return fmt.Errorf("发布规则重载事件失败: %w", err)
	}

	appLogger.Info("已发布规则重载事件")
	return nil
}

// loadFromRedis 从 Redis 加载规则列表
func (c *RuleCache) loadFromRedis(ctx context.Context) ([]*biz.AlertRule, error) {
	val, err := c.rdb.Get(ctx, redisRuleCacheKey).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("Redis 缓存未命中")
	} else if err != nil {
		return nil, fmt.Errorf("读取 Redis 失败: %w", err)
	}

	var rules []*biz.AlertRule
	if err := json.Unmarshal([]byte(val), &rules); err != nil {
		return nil, fmt.Errorf("解析 Redis 数据失败: %w", err)
	}

	return rules, nil
}

// loadFromMySQL 从 MySQL 加载规则列表并写入 Redis
func (c *RuleCache) loadFromMySQL(ctx context.Context) ([]*biz.AlertRule, error) {
	// 从数据库查询启用的规则
	rules, err := c.ruleRepo.ListEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询 MySQL 失败: %w", err)
	}

	appLogger.Info("从 MySQL 加载规则列表", zap.Int("count", len(rules)))

	// 写入 Redis 缓存
	if err := c.saveToRedis(ctx, rules); err != nil {
		appLogger.Warn("写入 Redis 缓存失败", zap.Error(err))
		// 写入失败不影响返回结果
	}

	return rules, nil
}

// saveToRedis 保存规则列表到 Redis
func (c *RuleCache) saveToRedis(ctx context.Context, rules []*biz.AlertRule) error {
	data, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("序列化规则列表失败: %w", err)
	}

	err = c.rdb.Set(ctx, redisRuleCacheKey, data, ruleCacheTTL).Err()
	if err != nil {
		return fmt.Errorf("写入 Redis 失败: %w", err)
	}

	appLogger.Debug("规则列表已写入 Redis 缓存", zap.Int("count", len(rules)))
	return nil
}
