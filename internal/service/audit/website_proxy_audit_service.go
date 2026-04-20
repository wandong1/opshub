// Copyright (c) 2026 DYCloud J.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED.

package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	auditbiz "github.com/ydcloud-dy/opshub/internal/biz/audit"
	"github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	websiteAuditQueueKey = "website:audit:queue"
	websiteAuditLockKey  = "website:audit:lock"
)

// WebsiteProxyAuditService 网站代理访问审计服务
type WebsiteProxyAuditService struct {
	useCase *auditbiz.WebsiteProxyAuditUseCase
	rdb     *redis.Client
}

// NewWebsiteProxyAuditService 创建网站代理访问审计服务
func NewWebsiteProxyAuditService(useCase *auditbiz.WebsiteProxyAuditUseCase, rdb *redis.Client) *WebsiteProxyAuditService {
	return &WebsiteProxyAuditService{
		useCase: useCase,
		rdb:     rdb,
	}
}

// EnqueueAuditEvent 入队审计事件（异步，fail-open）
func (s *WebsiteProxyAuditService) EnqueueAuditEvent(ctx context.Context, event *auditbiz.WebsiteProxyAuditLog) {
	if s.rdb == nil {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		logger.Error("序列化审计事件失败", zap.Error(err))
		return
	}

	if err := s.rdb.RPush(ctx, websiteAuditQueueKey, data).Err(); err != nil {
		logger.Error("审计事件入队失败", zap.Error(err))
	}
}

// WebsiteProxyAuditWorker 网站代理访问审计批量消费 worker
type WebsiteProxyAuditWorker struct {
	rdb     *redis.Client
	db      *gorm.DB
	useCase *auditbiz.WebsiteProxyAuditUseCase
	ticker  *time.Ticker
	stopCh  chan struct{}
}

// NewWebsiteProxyAuditWorker 创建审计 worker
func NewWebsiteProxyAuditWorker(rdb *redis.Client, db *gorm.DB, useCase *auditbiz.WebsiteProxyAuditUseCase) *WebsiteProxyAuditWorker {
	return &WebsiteProxyAuditWorker{
		rdb:     rdb,
		db:      db,
		useCase: useCase,
		ticker:  time.NewTicker(1 * time.Minute),
		stopCh:  make(chan struct{}),
	}
}

// Start 启动 worker
func (w *WebsiteProxyAuditWorker) Start() {
	if w.rdb == nil {
		logger.Warn("Redis 客户端未初始化，网站代理访问审计 worker 未启动")
		return
	}
	logger.Info("网站代理访问审计 worker 启动", zap.Duration("interval", 1*time.Minute))
	go w.run()
}

// Stop 停止 worker
func (w *WebsiteProxyAuditWorker) Stop() {
	logger.Info("网站代理访问审计 worker 停止中")
	close(w.stopCh)
	w.ticker.Stop()
}

// run worker 主循环
func (w *WebsiteProxyAuditWorker) run() {
	// 启动时立即执行一次，避免等待 1 分钟
	logger.Info("网站代理访问审计 worker 首次执行")
	w.processBatch()

	for {
		select {
		case <-w.ticker.C:
			w.processBatch()
		case <-w.stopCh:
			logger.Info("网站代理访问审计 worker 已停止")
			return
		}
	}
}

// processBatch 批量处理审计事件
func (w *WebsiteProxyAuditWorker) processBatch() {
	ctx := context.Background()

	// 检查队列长度
	queueLen, err := w.rdb.LLen(ctx, websiteAuditQueueKey).Result()
	if err != nil {
		logger.Error("获取审计队列长度失败", zap.Error(err))
		return
	}
	if queueLen == 0 {
		return
	}

	logger.Debug("开始处理审计事件批次", zap.Int64("queueLen", queueLen))

	// 尝试获取分布式锁（避免多实例重复消费）
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	locked, err := w.rdb.SetNX(ctx, websiteAuditLockKey, lockValue, 30*time.Second).Result()
	if err != nil {
		logger.Error("获取审计锁失败", zap.Error(err))
		return
	}
	if !locked {
		logger.Debug("审计锁已被其他实例持有，跳过本次处理")
		return
	}

	defer func() {
		// 释放锁
		val, _ := w.rdb.Get(ctx, websiteAuditLockKey).Result()
		if val == lockValue {
			w.rdb.Del(ctx, websiteAuditLockKey)
		}
	}()

	// 批量取出审计事件（最多 100 条）
	results, err := w.rdb.LRange(ctx, websiteAuditQueueKey, 0, 99).Result()
	if err != nil {
		logger.Error("从审计队列读取失败", zap.Error(err))
		return
	}
	if len(results) == 0 {
		return
	}

	logger.Info("从审计队列读取事件", zap.Int("count", len(results)))

	// 解析事件
	var logs []*auditbiz.WebsiteProxyAuditLog
	var parseErrors int
	for _, data := range results {
		var log auditbiz.WebsiteProxyAuditLog
		if err := json.Unmarshal([]byte(data), &log); err != nil {
			logger.Error("解析审计事件失败", zap.Error(err), zap.String("data", data))
			parseErrors++
			continue
		}
		logs = append(logs, &log)
	}

	if parseErrors > 0 {
		logger.Warn("部分审计事件解析失败", zap.Int("parseErrors", parseErrors), zap.Int("total", len(results)))
	}

	if len(logs) == 0 {
		logger.Warn("所有审计事件解析失败，清空队列", zap.Int("count", len(results)))
		w.rdb.LTrim(ctx, websiteAuditQueueKey, int64(len(results)), -1)
		return
	}

	// 批量写入数据库
	if err := w.useCase.BatchCreate(ctx, logs); err != nil {
		logger.Error("批量写入审计日志失败", zap.Error(err), zap.Int("count", len(logs)))
		// 写入失败不移除队列，下次重试
		return
	}

	// 从队列中移除已处理的事件
	w.rdb.LTrim(ctx, websiteAuditQueueKey, int64(len(results)), -1)

	logger.Info("批量处理网站代理访问审计事件成功", zap.Int("count", len(logs)), zap.Int("parseErrors", parseErrors))
}
