package cache

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics 监控指标
type Metrics struct {
	// 批量队列大小
	BatchQueueSize prometheus.Gauge

	// 批量同步耗时
	BatchFlushDuration prometheus.Histogram

	// 批量同步数量
	BatchFlushCount prometheus.Counter

	// 批量同步成功次数
	BatchFlushSuccess prometheus.Counter

	// 批量同步失败次数
	BatchFlushErrors prometheus.Counter

	// 状态变化立即写入次数
	ImmediateWriteCount prometheus.Counter

	// Redis 故障降级次数
	RedisFallbackCount prometheus.Counter

	// 锁竞争次数
	LockContentionCount prometheus.Counter

	// 锁获取成功次数
	LockAcquireSuccess prometheus.Counter

	// 锁获取失败次数
	LockAcquireErrors prometheus.Counter

	// 重新入队次数
	RequeueCount prometheus.Counter

	// 重新入队失败次数
	RequeueErrors prometheus.Counter
}

// NewMetrics 创建监控指标
func NewMetrics() *Metrics {
	return &Metrics{
		BatchQueueSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "agent_batch_queue_size",
			Help: "Current size of agent batch queue",
		}),

		BatchFlushDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "agent_batch_flush_duration_seconds",
			Help:    "Duration of batch flush operations",
			Buckets: prometheus.DefBuckets,
		}),

		BatchFlushCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "agent_batch_flush_total",
			Help: "Total number of agents flushed in batch operations",
		}),

		BatchFlushSuccess: promauto.NewCounter(prometheus.CounterOpts{
			Name: "agent_batch_flush_success_total",
			Help: "Total number of successful batch flush operations",
		}),

		BatchFlushErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "agent_batch_flush_errors_total",
			Help: "Total number of failed batch flush operations",
		}),

		ImmediateWriteCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "agent_immediate_write_total",
			Help: "Total number of immediate writes due to status change",
		}),

		RedisFallbackCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "agent_redis_fallback_total",
			Help: "Total number of Redis fallback to MySQL",
		}),

		LockContentionCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "agent_lock_contention_total",
			Help: "Total number of lock contention events",
		}),

		LockAcquireSuccess: promauto.NewCounter(prometheus.CounterOpts{
			Name: "agent_lock_acquire_success_total",
			Help: "Total number of successful lock acquisitions",
		}),

		LockAcquireErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "agent_lock_acquire_errors_total",
			Help: "Total number of lock acquisition errors",
		}),

		RequeueCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "agent_requeue_total",
			Help: "Total number of agents requeued after failure",
		}),

		RequeueErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "agent_requeue_errors_total",
			Help: "Total number of requeue errors",
		}),
	}
}

// CacheConfig 缓存配置
type CacheConfig struct {
	// 批量同步间隔（推荐 5 分钟）
	BatchFlushInterval time.Duration

	// 每批次处理数量（推荐 100-500）
	BatchSize int

	// 队列最大长度（告警阈值）
	BatchQueueMaxSize int

	// Redis 数据 TTL
	RedisTTL time.Duration

	// 分布式锁超时时间
	LockTimeout time.Duration

	// 离线检测阈值
	OfflineThreshold time.Duration
}

// DefaultCacheConfig 默认配置
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		BatchFlushInterval: 5 * time.Minute,
		BatchSize:          100,
		BatchQueueMaxSize:  10000,
		RedisTTL:           10 * time.Minute,
		LockTimeout:        60 * time.Second,
		OfflineThreshold:   10 * time.Minute,
	}
}
