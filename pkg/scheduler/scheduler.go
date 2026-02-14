package scheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
)

// Task represents a scheduled task loaded from the database.
type Task struct {
	ID       uint
	Name     string
	Type     string // matches TaskExecutor.Type()
	CronExpr string
	Payload  string // JSON, task-specific config
	Enabled  bool
}

// TaskExecutor executes a specific type of task.
type TaskExecutor interface {
	Type() string
	Execute(ctx context.Context, task Task) error
}

// TaskProvider loads enabled tasks from the database.
type TaskProvider interface {
	GetEnabledTasks(ctx context.Context) ([]Task, error)
}

// Stats holds scheduler runtime statistics.
type Stats struct {
	TasksTotal      int64
	TasksEnabled    int64
	ExecSuccess     int64
	ExecFail        int64
	ExecSkipped     int64
	LockAcquired    int64
	LockSkipped     int64
	LastReloadEpoch int64
}

// Scheduler manages cron jobs with distributed locking.
type Scheduler struct {
	cron       *cron.Cron
	lock       *RedisLock
	provider   TaskProvider
	executors  map[string]TaskExecutor
	entries    map[uint]cron.EntryID
	mu         sync.RWMutex
	stats      Stats
	lockPrefix string
	lockTTL    time.Duration
	stopCh     chan struct{}
}

// New creates a new Scheduler.
func New(redisClient *redis.Client, provider TaskProvider, opts ...Option) *Scheduler {
	s := &Scheduler{
		provider:   provider,
		executors:  make(map[string]TaskExecutor),
		entries:    make(map[uint]cron.EntryID),
		lockPrefix: "opshub:scheduler:lock",
		lockTTL:    30 * time.Second,
		stopCh:     make(chan struct{}),
	}
	for _, o := range opts {
		o(s)
	}
	s.lock = NewRedisLock(redisClient, s.lockPrefix, s.lockTTL)
	// Second-level cron parser
	s.cron = cron.New(cron.WithSeconds(), cron.WithChain(cron.Recover(cron.DefaultLogger)))
	return s
}

// RegisterExecutor registers a TaskExecutor for a given type.
func (s *Scheduler) RegisterExecutor(exec TaskExecutor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executors[exec.Type()] = exec
}

// Start loads tasks and starts the cron scheduler.
func (s *Scheduler) Start(ctx context.Context) error {
	if err := s.loadTasks(ctx); err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}
	s.cron.Start()
	appLogger.Info("scheduler started", zap.Int("tasks", len(s.entries)))
	return nil
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	close(s.stopCh)
	stopCtx := s.cron.Stop()
	<-stopCtx.Done()
	appLogger.Info("scheduler stopped")
}

// Reload reloads tasks from the provider.
func (s *Scheduler) Reload(ctx context.Context) error {
	s.mu.Lock()
	// Remove all existing entries
	for id, entryID := range s.entries {
		s.cron.Remove(entryID)
		delete(s.entries, id)
	}
	s.mu.Unlock()

	if err := s.loadTasks(ctx); err != nil {
		return fmt.Errorf("reload tasks: %w", err)
	}
	atomic.StoreInt64(&s.stats.LastReloadEpoch, time.Now().Unix())
	appLogger.Info("scheduler reloaded", zap.Int("tasks", len(s.entries)))
	return nil
}

// GetStats returns a copy of the current stats.
func (s *Scheduler) GetStats() Stats {
	return Stats{
		TasksTotal:      atomic.LoadInt64(&s.stats.TasksTotal),
		TasksEnabled:    atomic.LoadInt64(&s.stats.TasksEnabled),
		ExecSuccess:     atomic.LoadInt64(&s.stats.ExecSuccess),
		ExecFail:        atomic.LoadInt64(&s.stats.ExecFail),
		ExecSkipped:     atomic.LoadInt64(&s.stats.ExecSkipped),
		LockAcquired:    atomic.LoadInt64(&s.stats.LockAcquired),
		LockSkipped:     atomic.LoadInt64(&s.stats.LockSkipped),
		LastReloadEpoch: atomic.LoadInt64(&s.stats.LastReloadEpoch),
	}
}

func (s *Scheduler) loadTasks(ctx context.Context) error {
	tasks, err := s.provider.GetEnabledTasks(ctx)
	if err != nil {
		return err
	}
	atomic.StoreInt64(&s.stats.TasksTotal, int64(len(tasks)))
	var enabled int64
	for _, t := range tasks {
		if !t.Enabled {
			continue
		}
		enabled++
		task := t // capture
		entryID, err := s.cron.AddFunc(task.CronExpr, func() {
			s.executeTask(task)
		})
		if err != nil {
			appLogger.Error("failed to add cron entry",
				zap.Uint("taskID", task.ID),
				zap.String("cron", task.CronExpr),
				zap.Error(err),
			)
			continue
		}
		s.mu.Lock()
		s.entries[task.ID] = entryID
		s.mu.Unlock()
	}
	atomic.StoreInt64(&s.stats.TasksEnabled, enabled)
	return nil
}

func (s *Scheduler) executeTask(task Task) {
	lockKey := fmt.Sprintf("task:%d", task.ID)
	release, ok := s.lock.TryLock(context.Background(), lockKey)
	if !ok {
		atomic.AddInt64(&s.stats.LockSkipped, 1)
		atomic.AddInt64(&s.stats.ExecSkipped, 1)
		return
	}
	defer release()
	atomic.AddInt64(&s.stats.LockAcquired, 1)

	s.mu.RLock()
	exec, exists := s.executors[task.Type]
	s.mu.RUnlock()
	if !exists {
		appLogger.Error("no executor for task type", zap.String("type", task.Type))
		atomic.AddInt64(&s.stats.ExecFail, 1)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := exec.Execute(ctx, task); err != nil {
		appLogger.Error("task execution failed",
			zap.Uint("taskID", task.ID),
			zap.String("name", task.Name),
			zap.Error(err),
		)
		atomic.AddInt64(&s.stats.ExecFail, 1)
		return
	}
	atomic.AddInt64(&s.stats.ExecSuccess, 1)
}
