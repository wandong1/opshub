package scheduler

import "time"

// Option configures the Scheduler.
type Option func(*Scheduler)

// WithLockPrefix sets the Redis key prefix for distributed locks.
func WithLockPrefix(prefix string) Option {
	return func(s *Scheduler) { s.lockPrefix = prefix }
}

// WithLockTTL sets the TTL for distributed locks.
func WithLockTTL(ttl time.Duration) Option {
	return func(s *Scheduler) { s.lockTTL = ttl }
}
