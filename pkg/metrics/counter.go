package metrics

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

// RedisCounter is a persistent counter backed by Redis.
// Counter values survive application restarts by persisting to Redis.
// Redis INCR is atomic, so concurrent increments are safe without additional locks.
type RedisCounter struct {
	rdb    *redis.Client
	prefix string
}

// NewRedisCounter creates a new RedisCounter with the given prefix.
// prefix example: "srehub:counter"
func NewRedisCounter(rdb *redis.Client, prefix string) *RedisCounter {
	return &RedisCounter{rdb: rdb, prefix: prefix}
}

// Inc atomically increments the counter for the given metric and label set.
// Returns the new cumulative value.
func (c *RedisCounter) Inc(ctx context.Context, metricName string, labels map[string]string) float64 {
	key := c.buildKey(metricName, labels)
	val, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0
	}
	return float64(val)
}

// Get returns the current cumulative value for the given metric and label set.
func (c *RedisCounter) Get(ctx context.Context, metricName string, labels map[string]string) float64 {
	key := c.buildKey(metricName, labels)
	val, err := c.rdb.Get(ctx, key).Float64()
	if err != nil {
		return 0
	}
	return val
}

// IntoGauge creates a prometheus Gauge collector pre-set with the current counter value.
// Use this to push counter values to Pushgateway (which accepts any collector).
func (c *RedisCounter) IntoGauge(ctx context.Context, metricName, help string, labelNames []string, labelValues []string, labels map[string]string) prometheus.Collector {
	val := c.Get(ctx, metricName, labels)
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: metricName, Help: help}, labelNames)
	g.WithLabelValues(labelValues...).Set(val)
	return g
}

// buildKey builds a stable Redis key from metric name and labels.
// Format: {prefix}:{metricName}:{sorted_label_k=v,...}
func (c *RedisCounter) buildKey(metricName string, labels map[string]string) string {
	if len(labels) == 0 {
		return fmt.Sprintf("%s:%s", c.prefix, metricName)
	}

	// Sort keys for stability
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+labels[k])
	}
	return fmt.Sprintf("%s:%s:%s", c.prefix, metricName, strings.Join(pairs, ","))
}
