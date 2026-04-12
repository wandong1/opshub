package cache

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	agentrepo "github.com/ydcloud-dy/opshub/internal/data/agent"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	BatchQueueKey   = "agent:batch:queue"
	BatchPendingKey = "agent:batch:pending"
	BatchLockKey    = "agent:batch:lock"
)

// AgentData Agent 数据结构
type AgentData struct {
	AgentID  string
	Status   string
	LastSeen time.Time
}

// BatchWorker 批量同步 Worker
type BatchWorker struct {
	rdb        *redis.Client
	db         *gorm.DB
	agentRepo  *agentrepo.Repository
	cfg        *CacheConfig
	metrics    *Metrics
	scripts    *LuaScripts
	instanceID string
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewBatchWorker 创建批量同步 Worker
func NewBatchWorker(
	rdb *redis.Client,
	db *gorm.DB,
	agentRepo *agentrepo.Repository,
	cfg *CacheConfig,
	metrics *Metrics,
	scripts *LuaScripts,
) *BatchWorker {
	hostname, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s-%d", hostname, os.Getpid())

	return &BatchWorker{
		rdb:        rdb,
		db:         db,
		agentRepo:  agentRepo,
		cfg:        cfg,
		metrics:    metrics,
		scripts:    scripts,
		instanceID: instanceID,
		stopCh:     make(chan struct{}),
	}
}

// Start 启动 Worker
func (w *BatchWorker) Start() {
	w.wg.Add(1)
	go w.run()
	appLogger.Info("BatchWorker 已启动", zap.String("instanceID", w.instanceID))
}

// Stop 停止 Worker
func (w *BatchWorker) Stop() {
	close(w.stopCh)
	w.wg.Wait()
	appLogger.Info("BatchWorker 已停止", zap.String("instanceID", w.instanceID))
}

// run 主循环
func (w *BatchWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.cfg.BatchFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 尝试获取分布式锁
			if w.tryAcquireLock() {
				w.processBatch()
				w.releaseLock()
			} else {
				appLogger.Debug("其他副本正在处理批量同步，跳过",
					zap.String("instanceID", w.instanceID))
			}

		case <-w.stopCh:
			// 停止前最后一次同步
			if w.tryAcquireLock() {
				w.processBatch()
				w.releaseLock()
			}
			return
		}
	}
}

// tryAcquireLock 尝试获取分布式锁
func (w *BatchWorker) tryAcquireLock() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// SET agent:batch:lock {instanceID} NX EX 60
	ok, err := w.rdb.SetNX(ctx, BatchLockKey, w.instanceID, w.cfg.LockTimeout).Result()
	if err != nil {
		appLogger.Error("获取分布式锁失败",
			zap.String("instanceID", w.instanceID),
			zap.Error(err))
		w.metrics.LockAcquireErrors.Inc()
		return false
	}

	if ok {
		appLogger.Debug("成功获取分布式锁",
			zap.String("instanceID", w.instanceID))
		w.metrics.LockAcquireSuccess.Inc()
	} else {
		w.metrics.LockContentionCount.Inc()
	}

	return ok
}

// releaseLock 释放分布式锁
func (w *BatchWorker) releaseLock() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 使用 Lua 脚本原子释放（验证 owner）
	result, err := w.scripts.ReleaseLock.Run(ctx, w.rdb,
		[]string{BatchLockKey},
		w.instanceID).Int()

	if err != nil {
		appLogger.Error("释放分布式锁失败",
			zap.String("instanceID", w.instanceID),
			zap.Error(err))
		return
	}

	if result == 1 {
		appLogger.Debug("成功释放分布式锁",
			zap.String("instanceID", w.instanceID))
	} else {
		appLogger.Warn("释放锁失败：不是锁的持有者",
			zap.String("instanceID", w.instanceID))
	}
}

// processBatch 处理一批数据
func (w *BatchWorker) processBatch() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()

	// 1. 从 Redis 队列批量取出数据
	agentIDs, err := w.dequeueFromRedis(ctx, w.cfg.BatchSize)
	if err != nil {
		appLogger.Error("从队列取数据失败",
			zap.String("instanceID", w.instanceID),
			zap.Error(err))
		w.metrics.BatchFlushErrors.Inc()
		return
	}

	if len(agentIDs) == 0 {
		appLogger.Debug("队列为空，跳过批量同步",
			zap.String("instanceID", w.instanceID))
		return
	}

	appLogger.Info("开始批量同步",
		zap.String("instanceID", w.instanceID),
		zap.Int("count", len(agentIDs)))

	// 2. 从 Redis 读取最新的 last_seen
	agentDataMap, err := w.fetchAgentDataFromRedis(ctx, agentIDs)
	if err != nil {
		appLogger.Error("从 Redis 读取数据失败",
			zap.String("instanceID", w.instanceID),
			zap.Error(err))
		// 数据重新入队
		w.requeueAgents(ctx, agentIDs)
		w.metrics.BatchFlushErrors.Inc()
		return
	}

	// 3. 批量更新 MySQL
	if err := w.batchUpdateMySQL(ctx, agentDataMap); err != nil {
		appLogger.Error("批量更新 MySQL 失败",
			zap.String("instanceID", w.instanceID),
			zap.Error(err))
		// 数据重新入队
		w.requeueAgents(ctx, agentIDs)
		w.metrics.BatchFlushErrors.Inc()
		return
	}

	// 4. 记录指标
	duration := time.Since(startTime)
	w.metrics.BatchFlushDuration.Observe(duration.Seconds())
	w.metrics.BatchFlushCount.Add(float64(len(agentIDs)))
	w.metrics.BatchFlushSuccess.Inc()

	appLogger.Info("批量同步完成",
		zap.String("instanceID", w.instanceID),
		zap.Int("count", len(agentIDs)),
		zap.Duration("duration", duration))
}

// dequeueFromRedis 从 Redis 队列批量取出数据
func (w *BatchWorker) dequeueFromRedis(ctx context.Context, batchSize int) ([]string, error) {
	result, err := w.scripts.DequeueBatch.Run(ctx, w.rdb,
		[]string{BatchQueueKey, BatchPendingKey},
		batchSize).Result()

	if err != nil {
		return nil, err
	}

	// 转换结果
	agentIDs := make([]string, 0)
	if items, ok := result.([]interface{}); ok {
		for _, item := range items {
			if agentID, ok := item.(string); ok {
				agentIDs = append(agentIDs, agentID)
			}
		}
	}

	return agentIDs, nil
}

// fetchAgentDataFromRedis 从 Redis 批量读取 Agent 数据
func (w *BatchWorker) fetchAgentDataFromRedis(ctx context.Context, agentIDs []string) (map[string]*AgentData, error) {
	// 使用 Pipeline 批量读取
	pipe := w.rdb.Pipeline()

	cmds := make(map[string]*redis.MapStringStringCmd)
	for _, agentID := range agentIDs {
		key := fmt.Sprintf("agent:status:%s", agentID)
		cmds[agentID] = pipe.HGetAll(ctx, key)
	}

	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}

	// 解析结果
	agentDataMap := make(map[string]*AgentData)
	for agentID, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil {
			appLogger.Warn("读取 Agent 数据失败，跳过",
				zap.String("agentID", agentID),
				zap.Error(err))
			continue
		}

		if len(data) == 0 {
			appLogger.Warn("Agent 数据为空，跳过",
				zap.String("agentID", agentID))
			continue
		}

		// 解析 last_seen
		lastSeenUnix, err := strconv.ParseInt(data["last_seen"], 10, 64)
		if err != nil {
			appLogger.Warn("解析 last_seen 失败，跳过",
				zap.String("agentID", agentID),
				zap.Error(err))
			continue
		}
		lastSeen := time.Unix(lastSeenUnix, 0)

		agentDataMap[agentID] = &AgentData{
			AgentID:  agentID,
			Status:   data["status"],
			LastSeen: lastSeen,
		}
	}

	return agentDataMap, nil
}

// batchUpdateMySQL 批量更新 MySQL
func (w *BatchWorker) batchUpdateMySQL(ctx context.Context, agentDataMap map[string]*AgentData) error {
	if len(agentDataMap) == 0 {
		return nil
	}

	// 构造 SQL: UPDATE agent_info SET last_seen = CASE agent_id WHEN ... END
	var (
		agentIDs  []string
		caseWhen  strings.Builder
		args      []interface{}
	)

	caseWhen.WriteString("CASE agent_id ")

	for agentID, data := range agentDataMap {
		agentIDs = append(agentIDs, agentID)
		caseWhen.WriteString("WHEN ? THEN ? ")
		args = append(args, agentID, data.LastSeen)
	}

	caseWhen.WriteString("END")

	// 添加 WHERE IN 参数
	args = append(args, agentIDs)

	// 构造完整 SQL
	sql := fmt.Sprintf(`
		UPDATE agent_info
		SET last_seen = %s
		WHERE agent_id IN (?)
	`, caseWhen.String())

	// 执行更新
	return w.db.WithContext(ctx).Exec(sql, args...).Error
}

// requeueAgents 失败重试（重新入队）
func (w *BatchWorker) requeueAgents(ctx context.Context, agentIDs []string) {
	// 使用 Pipeline 批量重新入队
	pipe := w.rdb.Pipeline()

	for _, agentID := range agentIDs {
		pipe.Eval(ctx, luaEnqueueWithDedup,
			[]string{BatchQueueKey, BatchPendingKey},
			agentID)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		appLogger.Error("重新入队失败",
			zap.String("instanceID", w.instanceID),
			zap.Error(err),
			zap.Int("count", len(agentIDs)))
		w.metrics.RequeueErrors.Inc()
	} else {
		appLogger.Info("数据已重新入队",
			zap.String("instanceID", w.instanceID),
			zap.Int("count", len(agentIDs)))
		w.metrics.RequeueCount.Add(float64(len(agentIDs)))
	}
}
