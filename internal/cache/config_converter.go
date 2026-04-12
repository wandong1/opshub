package cache

import (
	"time"
)

// ConvertConfigToCacheConfig 将配置文件中的 CacheConfig 转换为 cache 包的 CacheConfig
func ConvertConfigToCacheConfig(cfg interface{}) *CacheConfig {
	// 类型断言，支持从 conf.CacheConfig 转换
	type configInterface interface {
		GetBatchFlushInterval() int
		GetBatchSize() int
		GetBatchQueueMaxSize() int
		GetRedisTTL() int
		GetLockTimeout() int
		GetOfflineThreshold() int
	}

	// 如果传入的是 map（从配置文件解析）
	if m, ok := cfg.(map[string]interface{}); ok {
		return &CacheConfig{
			BatchFlushInterval: time.Duration(getIntFromMap(m, "batch_flush_interval", 300)) * time.Second,
			BatchSize:          getIntFromMap(m, "batch_size", 100),
			BatchQueueMaxSize:  getIntFromMap(m, "batch_queue_max_size", 10000),
			RedisTTL:           time.Duration(getIntFromMap(m, "redis_ttl", 600)) * time.Second,
			LockTimeout:        time.Duration(getIntFromMap(m, "lock_timeout", 60)) * time.Second,
			OfflineThreshold:   time.Duration(getIntFromMap(m, "offline_threshold", 600)) * time.Second,
		}
	}

	// 如果传入的是结构体（从 conf.CacheConfig）
	type cacheConfigStruct struct {
		BatchFlushInterval int
		BatchSize          int
		BatchQueueMaxSize  int
		RedisTTL           int
		LockTimeout        int
		OfflineThreshold   int
	}

	if c, ok := cfg.(cacheConfigStruct); ok {
		return &CacheConfig{
			BatchFlushInterval: time.Duration(c.BatchFlushInterval) * time.Second,
			BatchSize:          c.BatchSize,
			BatchQueueMaxSize:  c.BatchQueueMaxSize,
			RedisTTL:           time.Duration(c.RedisTTL) * time.Second,
			LockTimeout:        time.Duration(c.LockTimeout) * time.Second,
			OfflineThreshold:   time.Duration(c.OfflineThreshold) * time.Second,
		}
	}

	// 默认配置
	return DefaultCacheConfig()
}

func getIntFromMap(m map[string]interface{}, key string, defaultValue int) int {
	if v, ok := m[key]; ok {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return defaultValue
}
