package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ydcloud-dy/opshub/internal/cache"
)

const (
	redisRuleEvalPrefix = "alert:rule:eval:"
	defaultRedisTTL     = 24 * time.Hour
	defaultSyncInterval = 30 * time.Second
)

// RuleEvalState Redis 中存储的规则评估状态
type RuleEvalState struct {
	LastEvalAt time.Time `json:"lastEvalAt"`
	EvalCount  int64     `json:"evalCount"`
	LastValue  float64   `json:"lastValue"`
}

// EvalCache 评估时间缓存管理器
type EvalCache struct {
	rdb             *redis.Client
	db              *gorm.DB
	scripts         *cache.LuaScripts
	redisTTL        time.Duration
	syncInterval    time.Duration
}

// NewEvalCache 创建评估时间缓存管理器
func NewEvalCache(rdb *redis.Client, db *gorm.DB, scripts *cache.LuaScripts) *EvalCache {
	return &EvalCache{
		rdb:          rdb,
		db:           db,
		scripts:      scripts,
		redisTTL:     defaultRedisTTL,
		syncInterval: defaultSyncInterval,
	}
}

// SetRedisTTL 设置 Redis TTL
func (c *EvalCache) SetRedisTTL(ttl time.Duration) {
	c.redisTTL = ttl
}

// SetSyncInterval 设置同步间隔
func (c *EvalCache) SetSyncInterval(interval time.Duration) {
	c.syncInterval = interval
}

// UpdateEvalTime 更新单条规则评估时间到 Redis
func (c *EvalCache) UpdateEvalTime(ctx context.Context, ruleID uint, evalTime time.Time) error {
	key := fmt.Sprintf("%s%d", redisRuleEvalPrefix, ruleID)

	// 读取现有状态
	var state RuleEvalState
	val, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		// 首次评估
		state = RuleEvalState{
			LastEvalAt: evalTime,
			EvalCount:  1,
		}
	} else if err != nil {
		return fmt.Errorf("读取 Redis 失败: %w", err)
	} else {
		// 更新现有状态
		if err := json.Unmarshal([]byte(val), &state); err != nil {
			// JSON 解析失败，重置状态
			state = RuleEvalState{
				LastEvalAt: evalTime,
				EvalCount:  1,
			}
		} else {
			state.LastEvalAt = evalTime
			state.EvalCount++
		}
	}

	// 序列化并写入 Redis
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	if err := c.rdb.Set(ctx, key, data, c.redisTTL).Err(); err != nil {
		return fmt.Errorf("写入 Redis 失败: %w", err)
	}

	return nil
}

// BatchUpdateEvalTimes 批量更新评估时间到 Redis（Pipeline）
func (c *EvalCache) BatchUpdateEvalTimes(ctx context.Context, updates map[uint]time.Time) error {
	if len(updates) == 0 {
		return nil
	}

	pipe := c.rdb.Pipeline()

	for ruleID, evalTime := range updates {
		key := fmt.Sprintf("%s%d", redisRuleEvalPrefix, ruleID)

		// 简化版本：直接存储时间戳
		state := RuleEvalState{
			LastEvalAt: evalTime,
			EvalCount:  1,
		}

		data, err := json.Marshal(state)
		if err != nil {
			appLogger.Warn("序列化规则评估状态失败",
				zap.Uint("ruleID", ruleID),
				zap.Error(err))
			continue
		}

		pipe.Set(ctx, key, data, c.redisTTL)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("批量写入 Redis 失败: %w", err)
	}

	return nil
}

// GetEvalTimes 批量获取评估时间（优先 Redis，未命中查 MySQL）
func (c *EvalCache) GetEvalTimes(ctx context.Context, ruleIDs []uint) (map[uint]*time.Time, error) {
	if len(ruleIDs) == 0 {
		return make(map[uint]*time.Time), nil
	}

	result := make(map[uint]*time.Time)

	// 1. 批量从 Redis 读取
	pipe := c.rdb.Pipeline()
	cmds := make(map[uint]*redis.StringCmd)

	for _, ruleID := range ruleIDs {
		key := fmt.Sprintf("%s%d", redisRuleEvalPrefix, ruleID)
		cmds[ruleID] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		appLogger.Warn("批量读取 Redis 失败", zap.Error(err))
	}

	// 2. 解析 Redis 结果，收集未命中的规则 ID
	var missedIDs []uint

	for ruleID, cmd := range cmds {
		val, err := cmd.Result()
		if err == redis.Nil {
			// Redis 未命中，记录到 missedIDs
			missedIDs = append(missedIDs, ruleID)
			continue
		} else if err != nil {
			appLogger.Warn("读取 Redis key 失败",
				zap.Uint("ruleID", ruleID),
				zap.Error(err))
			missedIDs = append(missedIDs, ruleID)
			continue
		}

		// 解析 JSON
		var state RuleEvalState
		if err := json.Unmarshal([]byte(val), &state); err != nil {
			appLogger.Warn("解析规则评估状态失败",
				zap.Uint("ruleID", ruleID),
				zap.Error(err))
			missedIDs = append(missedIDs, ruleID)
			continue
		}

		// Redis 命中
		evalTime := state.LastEvalAt
		result[ruleID] = &evalTime
	}

	// 3. 从 MySQL 回填未命中的数据
	if len(missedIDs) > 0 {
		mysqlResult, err := c.getEvalTimesFromMySQL(ctx, missedIDs)
		if err != nil {
			appLogger.Warn("从 MySQL 回填评估时间失败", zap.Error(err))
		} else {
			// 合并 MySQL 结果
			for ruleID, evalTime := range mysqlResult {
				result[ruleID] = evalTime

				// 回写到 Redis（异步，不阻塞）
				if evalTime != nil {
					go func(id uint, t time.Time) {
						ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
						defer cancel()
						_ = c.UpdateEvalTime(ctx, id, t)
					}(ruleID, *evalTime)
				}
			}
		}
	}

	return result, nil
}

// getEvalTimesFromMySQL 从 MySQL 批量查询评估时间
func (c *EvalCache) getEvalTimesFromMySQL(ctx context.Context, ruleIDs []uint) (map[uint]*time.Time, error) {
	if len(ruleIDs) == 0 {
		return make(map[uint]*time.Time), nil
	}

	type queryResult struct {
		ID         uint       `gorm:"column:id"`
		LastEvalAt *time.Time `gorm:"column:last_eval_at"`
	}

	var rows []queryResult
	err := c.db.WithContext(ctx).
		Table("alert_rules").
		Select("id, last_eval_at").
		Where("id IN ?", ruleIDs).
		Scan(&rows).Error

	if err != nil {
		return nil, fmt.Errorf("查询 MySQL 失败: %w", err)
	}

	result := make(map[uint]*time.Time)
	for _, row := range rows {
		result[row.ID] = row.LastEvalAt
	}

	return result, nil
}

// SyncToMySQL 批量同步 Redis → MySQL
func (c *EvalCache) SyncToMySQL(ctx context.Context) error {
	startTime := time.Now()

	// 1. 扫描所有规则评估 key
	pattern := fmt.Sprintf("%s*", redisRuleEvalPrefix)
	keys, err := c.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("扫描 Redis keys 失败: %w", err)
	}

	if len(keys) == 0 {
		appLogger.Debug("没有需要同步的规则评估数据")
		return nil
	}

	appLogger.Info("开始批量同步规则评估时间",
		zap.Int("count", len(keys)))

	// 2. 批量读取 Redis 数据
	pipe := c.rdb.Pipeline()
	cmds := make(map[uint]*redis.StringCmd)

	for _, key := range keys {
		// 从 key 中提取 ruleID
		ruleIDStr := strings.TrimPrefix(key, redisRuleEvalPrefix)
		var ruleID uint
		if _, err := fmt.Sscanf(ruleIDStr, "%d", &ruleID); err != nil {
			appLogger.Warn("解析规则 ID 失败",
				zap.String("key", key),
				zap.Error(err))
			continue
		}

		cmds[ruleID] = pipe.Get(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return fmt.Errorf("批量读取 Redis 失败: %w", err)
	}

	// 3. 解析数据并准备批量更新
	var updates []updateItem

	for ruleID, cmd := range cmds {
		val, err := cmd.Result()
		if err != nil {
			appLogger.Warn("读取规则评估状态失败",
				zap.Uint("ruleID", ruleID),
				zap.Error(err))
			continue
		}

		var state RuleEvalState
		if err := json.Unmarshal([]byte(val), &state); err != nil {
			appLogger.Warn("解析规则评估状态失败",
				zap.Uint("ruleID", ruleID),
				zap.Error(err))
			continue
		}

		updates = append(updates, updateItem{
			RuleID:     ruleID,
			LastEvalAt: state.LastEvalAt,
		})
	}

	if len(updates) == 0 {
		appLogger.Debug("没有有效的评估数据需要同步")
		return nil
	}

	// 4. 批量更新 MySQL（使用 CASE WHEN）
	if err := c.batchUpdateMySQL(ctx, updates); err != nil {
		return fmt.Errorf("批量更新 MySQL 失败: %w", err)
	}

	duration := time.Since(startTime)
	appLogger.Info("批量同步规则评估时间完成",
		zap.Int("count", len(updates)),
		zap.Duration("duration", duration))

	return nil
}

// updateItem 批量更新项
type updateItem struct {
	RuleID     uint
	LastEvalAt time.Time
}

// batchUpdateMySQL 批量更新 MySQL（使用 CASE WHEN）
func (c *EvalCache) batchUpdateMySQL(ctx context.Context, updates []updateItem) error {
	if len(updates) == 0 {
		return nil
	}

	// 构建 CASE WHEN 语句
	var caseWhen strings.Builder
	var ruleIDs []uint
	var args []interface{}

	caseWhen.WriteString("CASE id ")

	for _, item := range updates {
		caseWhen.WriteString("WHEN ? THEN ? ")
		args = append(args, item.RuleID, item.LastEvalAt)
		ruleIDs = append(ruleIDs, item.RuleID)
	}

	caseWhen.WriteString("END")

	// 执行批量更新
	sql := fmt.Sprintf(`
		UPDATE alert_rules
		SET last_eval_at = %s
		WHERE id IN (?)
	`, caseWhen.String())

	args = append(args, ruleIDs)

	err := c.db.WithContext(ctx).Exec(sql, args...).Error
	if err != nil {
		return fmt.Errorf("执行批量更新失败: %w", err)
	}

	return nil
}
