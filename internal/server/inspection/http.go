package inspection

import (
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	biz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	inspectiondata "github.com/ydcloud-dy/opshub/internal/data/inspection"
	svc "github.com/ydcloud-dy/opshub/internal/service/inspection"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/scheduler"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HTTPServer holds all inspection services and registers routes.
type HTTPServer struct {
	probeConfigService   *svc.ProbeConfigService
	probeTaskService     *svc.ProbeTaskService
	pushgatewayService   *svc.PushgatewayService
	probeVariableService *svc.ProbeVariableService
	scheduler            *scheduler.Scheduler
	executor             *biz.NetworkProbeExecutor
	healthExecutor       *assetbiz.HostHealthExecutor
}

// NewHTTPServer creates the HTTPServer.
func NewHTTPServer(
	probeConfigService *svc.ProbeConfigService,
	probeTaskService *svc.ProbeTaskService,
	pushgatewayService *svc.PushgatewayService,
	probeVariableService *svc.ProbeVariableService,
	sched *scheduler.Scheduler,
	executor *biz.NetworkProbeExecutor,
	healthExecutor *assetbiz.HostHealthExecutor,
) *HTTPServer {
	return &HTTPServer{
		probeConfigService:   probeConfigService,
		probeTaskService:     probeTaskService,
		pushgatewayService:   pushgatewayService,
		probeVariableService: probeVariableService,
		scheduler:            sched,
		executor:             executor,
		healthExecutor:       healthExecutor,
	}
}

// SetAgentCommandFactory injects Agent capability into executor and service.
func (s *HTTPServer) SetAgentCommandFactory(f biz.AgentCommandFactory) {
	if s.executor != nil {
		s.executor.SetAgentCommandFactory(f)
	}
	if s.probeConfigService != nil {
		s.probeConfigService.SetAgentCommandFactory(f)
	}
	if s.healthExecutor != nil {
		s.healthExecutor.SetAgentCommandFactory(assetbiz.AgentCommandFactory(f))
	}
}

// SetVariableResolver injects variable resolver into executor and service.
func (s *HTTPServer) SetVariableResolver(r *biz.VariableResolver) {
	if s.executor != nil {
		s.executor.SetVariableResolver(r)
	}
	if s.probeConfigService != nil {
		s.probeConfigService.SetVariableResolver(r)
	}
}

// RegisterRoutes registers all inspection routes under the given group.
func (s *HTTPServer) RegisterRoutes(r *gin.RouterGroup) {
	inspection := r.Group("/inspection")

	probes := inspection.Group("/probes")
	{
		probes.GET("", s.probeConfigService.List)
		probes.POST("", s.probeConfigService.Create)
		probes.POST("/test", s.probeConfigService.TestProbe)
		probes.GET("/:id", s.probeConfigService.Get)
		probes.PUT("/:id", s.probeConfigService.Update)
		probes.DELETE("/:id", s.probeConfigService.Delete)
		probes.POST("/import", s.probeConfigService.Import)
		probes.GET("/export", s.probeConfigService.Export)
		probes.POST("/:id/run", s.probeConfigService.RunOnce)
	}

	tasks := inspection.Group("/tasks")
	{
		tasks.GET("", s.probeTaskService.List)
		tasks.POST("", s.probeTaskService.Create)
		tasks.GET("/:id", s.probeTaskService.Get)
		tasks.PUT("/:id", s.probeTaskService.Update)
		tasks.DELETE("/:id", s.probeTaskService.Delete)
		tasks.PUT("/:id/toggle", s.probeTaskService.Toggle)
		tasks.GET("/:id/results", s.probeTaskService.Results)
	}

	pushgateways := inspection.Group("/pushgateways")
	{
		pushgateways.GET("", s.pushgatewayService.List)
		pushgateways.POST("", s.pushgatewayService.Create)
		pushgateways.PUT("/:id", s.pushgatewayService.Update)
		pushgateways.DELETE("/:id", s.pushgatewayService.Delete)
		pushgateways.POST("/:id/test", s.pushgatewayService.Test)
	}

	variables := inspection.Group("/variables")
	{
		variables.GET("", s.probeVariableService.List)
		variables.POST("", s.probeVariableService.Create)
		variables.GET("/:id", s.probeVariableService.Get)
		variables.PUT("/:id", s.probeVariableService.Update)
		variables.DELETE("/:id", s.probeVariableService.Delete)
	}
}

// Scheduler returns the scheduler instance for lifecycle management.
func (s *HTTPServer) Scheduler() *scheduler.Scheduler {
	return s.scheduler
}

// taskProvider adapts the ProbeTaskRepo to scheduler.TaskProvider.
type taskProvider struct {
	taskRepo   biz.ProbeTaskRepo
	configRepo biz.ProbeConfigRepo
}

func (p *taskProvider) GetEnabledTasks(ctx context.Context) ([]scheduler.Task, error) {
	tasks, err := p.taskRepo.GetEnabled(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]scheduler.Task, 0, len(tasks)+1)
	for _, t := range tasks {
		payload, _ := json.Marshal(map[string]uint{"task_id": t.ID})
		result = append(result, scheduler.Task{
			ID:       t.ID,
			Name:     t.Name,
			Type:     "network_probe",
			CronExpr: t.CronExpr,
			Payload:  string(payload),
			Enabled:  t.Status == 1,
		})
	}

	// 内置系统任务：主机健康检查（每5分钟）
	result = append(result, scheduler.Task{
		ID:       999999,
		Name:     "host_health_check",
		Type:     "host_health_check",
		CronExpr: "0 0/5 * * * ?",
		Payload:  "{}",
		Enabled:  true,
	})

	return result, nil
}

// NewInspectionServices is the factory function that assembles the full dependency chain.
// Returns the HTTPServer and starts the scheduler.
func NewInspectionServices(db *gorm.DB, redisClient *redis.Client, hostRepo assetbiz.HostRepo, credentialRepo assetbiz.CredentialRepo) *HTTPServer {
	// Repos
	configRepo := inspectiondata.NewProbeConfigRepo(db)
	taskRepo := inspectiondata.NewProbeTaskRepo(db)
	resultRepo := inspectiondata.NewProbeResultRepo(db)
	pgwRepo := inspectiondata.NewPushgatewayConfigRepo(db)
	variableRepo := inspectiondata.NewProbeVariableRepo(db)

	// Run one-time data migration for category backfill and task config association
	migrateData(db)

	// UseCases
	configUC := biz.NewProbeConfigUseCase(configRepo)
	taskUC := biz.NewProbeTaskUseCase(taskRepo, configRepo)
	resultUC := biz.NewProbeResultUseCase(resultRepo)
	pgwUC := biz.NewPushgatewayUseCase(pgwRepo)
	variableUC := biz.NewProbeVariableUseCase(variableRepo)

	// Group name lookup via AssetGroup table
	groupLookup := func(ctx context.Context, id uint) string {
		var group assetbiz.AssetGroup
		if err := db.WithContext(ctx).Select("name").First(&group, id).Error; err != nil {
			return ""
		}
		return group.Name
	}

	// Scheduler
	provider := &taskProvider{taskRepo: taskRepo, configRepo: configRepo}
	sched := scheduler.New(redisClient, provider)

	// Executor
	executor := biz.NewNetworkProbeExecutor(taskRepo, resultRepo, pgwRepo, groupLookup)
	variableResolver := biz.NewVariableResolver(variableRepo)
	executor.SetVariableResolver(variableResolver)
	sched.RegisterExecutor(executor)

	// Host health check executor
	healthExecutor := assetbiz.NewHostHealthExecutor(hostRepo, credentialRepo)
	sched.RegisterExecutor(healthExecutor)

	// Start scheduler
	if err := sched.Start(context.Background()); err != nil {
		appLogger.Error("启动巡检调度器失败", zap.Error(err))
	} else {
		appLogger.Info("巡检调度器启动成功")
	}

	// Services
	probeConfigSvc := svc.NewProbeConfigService(configUC)
	probeConfigSvc.SetVariableResolver(variableResolver)
	probeTaskSvc := svc.NewProbeTaskService(taskUC, resultUC, sched)
	pgwSvc := svc.NewPushgatewayService(pgwUC)
	variableSvc := svc.NewProbeVariableService(variableUC)

	return NewHTTPServer(probeConfigSvc, probeTaskSvc, pgwSvc, variableSvc, sched, executor, healthExecutor)
}

// migrateData performs one-time data migration:
// 1. Backfill category from type for existing probe_configs
// 2. Migrate probe_tasks.probe_config_id to probe_task_configs association table
func migrateData(db *gorm.DB) {
	// Backfill category for existing probe_configs where category is empty or default
	for probeType, category := range biz.TypeToCategoryMap {
		db.Exec("UPDATE probe_configs SET category = ? WHERE type = ? AND (category = '' OR category = 'network')", category, probeType)
	}

	// Migrate probe_config_id from probe_tasks to probe_task_configs
	// Only if the old column still exists
	var colExists bool
	db.Raw("SELECT COUNT(*) > 0 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'probe_tasks' AND COLUMN_NAME = 'probe_config_id'").Scan(&colExists)
	if colExists {
		db.Exec(`INSERT IGNORE INTO probe_task_configs (probe_task_id, probe_config_id)
			SELECT id, probe_config_id FROM probe_tasks WHERE probe_config_id > 0 AND deleted_at IS NULL`)
		appLogger.Info("巡检数据迁移完成: probe_config_id → probe_task_configs")
	}
}

// StopScheduler stops the inspection scheduler gracefully.
func StopScheduler(s *HTTPServer) {
	if s != nil && s.scheduler != nil {
		s.scheduler.Stop()
		appLogger.Info("巡检调度器已停止")
	}
}

// AutoMigrateModels returns the list of models for GORM auto-migration.
func AutoMigrateModels() []interface{} {
	return []interface{}{
		&biz.ProbeConfig{},
		&biz.ProbeTask{},
		&biz.ProbeResult{},
		&biz.PushgatewayConfig{},
		&biz.ProbeTaskConfig{},
		&biz.ProbeVariable{},
	}
}

// Migrate runs auto-migration for inspection tables.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(AutoMigrateModels()...)
}

func init() {
	// Ensure interface compliance at compile time.
	var _ scheduler.TaskProvider = (*taskProvider)(nil)
	var _ scheduler.TaskExecutor = (*biz.NetworkProbeExecutor)(nil)
}
