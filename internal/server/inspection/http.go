package inspection

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	biz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	inspectiondata "github.com/ydcloud-dy/opshub/internal/data/inspection"
	inspectionmgmtdata "github.com/ydcloud-dy/opshub/internal/data/inspection_mgmt"
	"github.com/ydcloud-dy/opshub/internal/server/agent"
	svc "github.com/ydcloud-dy/opshub/internal/service/inspection"
	inspectionmgmtsvc "github.com/ydcloud-dy/opshub/internal/service/inspection_mgmt"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/metrics"
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
	probeV2Executor      *biz.NetworkProbeV2Executor
	healthExecutor       *assetbiz.HostHealthExecutor

	// 巡检管理服务
	inspectionGroupService     *inspectionmgmtsvc.GroupService
	inspectionItemService      *inspectionmgmtsvc.ItemService
	inspectionRecordService    *inspectionmgmtsvc.RecordService
	inspectionTaskService      *inspectionmgmtsvc.TaskService
	executionRecordService     *inspectionmgmtsvc.ExecutionRecordService
	inspectionExecutor         *inspectionmgmtsvc.InspectionExecutor

	// 手动运行中的任务取消函数（taskID → cancel）
	runningTasksMu sync.Mutex
	runningTasks   map[uint]context.CancelFunc

	// 巡检管理仓库（用于导出功能）
	hostRepo       assetbiz.HostRepo
	groupRepo      inspectionmgmtdata.GroupRepository
	itemRepo       inspectionmgmtdata.ItemRepository
	recordRepo     inspectionmgmtdata.RecordRepository
	execRecordRepo inspectionmgmtdata.ExecutionRecordRepository
}

// NewHTTPServer creates the HTTPServer.
func NewHTTPServer(
	probeConfigService *svc.ProbeConfigService,
	probeTaskService *svc.ProbeTaskService,
	pushgatewayService *svc.PushgatewayService,
	probeVariableService *svc.ProbeVariableService,
	sched *scheduler.Scheduler,
	executor *biz.NetworkProbeExecutor,
	probeV2Executor *biz.NetworkProbeV2Executor,
	healthExecutor *assetbiz.HostHealthExecutor,
	inspectionGroupService *inspectionmgmtsvc.GroupService,
	inspectionItemService *inspectionmgmtsvc.ItemService,
	inspectionRecordService *inspectionmgmtsvc.RecordService,
	inspectionTaskService *inspectionmgmtsvc.TaskService,
	executionRecordService *inspectionmgmtsvc.ExecutionRecordService,
	inspectionExecutor *inspectionmgmtsvc.InspectionExecutor,
	hostRepo assetbiz.HostRepo,
	groupRepo inspectionmgmtdata.GroupRepository,
	itemRepo inspectionmgmtdata.ItemRepository,
	recordRepo inspectionmgmtdata.RecordRepository,
	execRecordRepo inspectionmgmtdata.ExecutionRecordRepository,
) *HTTPServer {
	return &HTTPServer{
		probeConfigService:      probeConfigService,
		probeTaskService:        probeTaskService,
		pushgatewayService:      pushgatewayService,
		probeVariableService:    probeVariableService,
		scheduler:               sched,
		executor:                executor,
		probeV2Executor:         probeV2Executor,
		healthExecutor:          healthExecutor,
		inspectionGroupService:  inspectionGroupService,
		inspectionItemService:   inspectionItemService,
		inspectionRecordService: inspectionRecordService,
		inspectionTaskService:   inspectionTaskService,
		executionRecordService:  executionRecordService,
		inspectionExecutor:      inspectionExecutor,
		runningTasks:            make(map[uint]context.CancelFunc),
		hostRepo:                hostRepo,
		groupRepo:               groupRepo,
		itemRepo:                itemRepo,
		recordRepo:              recordRepo,
		execRecordRepo:          execRecordRepo,
	}
}

// SetAgentCommandFactory injects Agent capability into executor and service.
func (s *HTTPServer) SetAgentCommandFactory(f biz.AgentCommandFactory) {
	if s.executor != nil {
		s.executor.SetAgentCommandFactory(f)
	}
	if s.probeV2Executor != nil {
		s.probeV2Executor.SetAgentCommandFactory(f)
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

// SetTeleAIEnabled sets the global TeleAI Authorization auto-fill switch.
func (s *HTTPServer) SetTeleAIEnabled(enabled bool) {
	if s.executor != nil {
		s.executor.SetTeleAIEnabled(enabled)
	}
	if s.probeV2Executor != nil {
		s.probeV2Executor.SetTeleAIEnabled(enabled)
	}
	if s.probeConfigService != nil {
		s.probeConfigService.SetTeleAIEnabled(enabled)
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

	// 注册巡检管理路由
	s.RegisterInspectionMgmtRoutes(r)
}

// Scheduler returns the scheduler instance for lifecycle management.
func (s *HTTPServer) Scheduler() *scheduler.Scheduler {
	return s.scheduler
}

// taskProvider adapts the ProbeTaskRepo and InspectionTaskRepo to scheduler.TaskProvider.
type taskProvider struct {
	taskRepo           biz.ProbeTaskRepo
	configRepo         biz.ProbeConfigRepo
	inspectionTaskRepo inspectionmgmtdata.TaskRepository
}

func (p *taskProvider) GetEnabledTasks(ctx context.Context) ([]scheduler.Task, error) {
	// 获取拨测任务
	probeTasks, err := p.taskRepo.GetEnabled(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]scheduler.Task, 0, len(probeTasks)+10)
	for _, t := range probeTasks {
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

	// 获取新表中的巡检任务和拨测任务
	if p.inspectionTaskRepo != nil {
		inspectionTasks, err := p.inspectionTaskRepo.GetEnabledTasks(ctx)
		if err != nil {
			appLogger.Error("get enabled inspection tasks failed", zap.Error(err))
		} else {
			for _, t := range inspectionTasks {
				if t.TaskType == "inspection" {
					// 巡检任务
					payload, _ := json.Marshal(map[string]uint{"task_id": t.ID})
					result = append(result, scheduler.Task{
						ID:       t.ID + 100000, // 偏移ID避免与旧拨测任务冲突
						Name:     t.Name,
						Type:     "inspection_task",
						CronExpr: t.CronExpr,
						Payload:  string(payload),
						Enabled:  t.Enabled,
					})
				} else if t.TaskType == "probe" {
					// 新表中的拨测任务
					payload, _ := json.Marshal(map[string]uint{"task_id": t.ID})
					result = append(result, scheduler.Task{
						ID:       t.ID + 200000, // 偏移ID避免与旧拨测任务和巡检任务冲突
						Name:     t.Name,
						Type:     "network_probe_v2",
						CronExpr: t.CronExpr,
						Payload:  string(payload),
						Enabled:  t.Enabled,
					})
				}
			}
		}
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

// pushgatewayAdapter 适配器，将拨测的 Pushgateway 配置转换为巡检执行器需要的格式
type pushgatewayAdapter struct {
	repo biz.PushgatewayConfigRepo
}

func (a *pushgatewayAdapter) GetByID(ctx context.Context, id uint) (*inspectionmgmtsvc.PushgatewayConfig, error) {
	pgw, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &inspectionmgmtsvc.PushgatewayConfig{
		ID:       pgw.ID,
		Name:     pgw.Name,
		URL:      pgw.URL,
		Username: pgw.Username,
		Password: pgw.Password,
		Status:   int(pgw.Status),
	}, nil
}

// NewInspectionServices is the factory function that assembles the full dependency chain.
// Returns the HTTPServer and starts the scheduler.
func NewInspectionServices(db *gorm.DB, redisClient *redis.Client, hostRepo assetbiz.HostRepo, credentialRepo assetbiz.CredentialRepo, agentHub *agent.AgentHub) *HTTPServer {
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

	// 初始化巡检管理的仓库（必须在 scheduler 启动前初始化）
	inspectionTaskRepo := inspectionmgmtdata.NewTaskRepository(db)
	inspectionGroupRepo := inspectionmgmtdata.NewGroupRepository(db)
	inspectionItemRepo := inspectionmgmtdata.NewItemRepository(db)
	inspectionRecordRepo := inspectionmgmtdata.NewRecordRepository(db)

	// Scheduler（包含 inspectionTaskRepo 以便加载新表任务）
	provider := &taskProvider{
		taskRepo:           taskRepo,
		configRepo:         configRepo,
		inspectionTaskRepo: inspectionTaskRepo,
	}
	sched := scheduler.New(redisClient, provider)

	// Executor
	executor := biz.NewNetworkProbeExecutor(taskRepo, resultRepo, pgwRepo, groupLookup)
	variableResolver := biz.NewVariableResolver(variableRepo)
	executor.SetVariableResolver(variableResolver)
	sched.RegisterExecutor(executor)

	// Host health check executor
	healthExecutor := assetbiz.NewHostHealthExecutor(hostRepo, credentialRepo)
	sched.RegisterExecutor(healthExecutor)

	// 初始化巡检管理服务
	inspectionGroupService, inspectionItemService, inspectionRecordService, inspectionTaskService, executionRecordService, inspectionGroupRepo, inspectionItemRepo, inspectionRecordRepo, _, execRecordRepo := InitInspectionMgmtServices(db, hostRepo, credentialRepo, agentHub)

	// 创建 Pushgateway 适配器
	pgwAdapter := &pushgatewayAdapter{repo: pgwRepo}

	// 创建巡检执行器
	inspectionExecutor := inspectionmgmtsvc.NewInspectionExecutor(
		inspectionTaskRepo,
		inspectionGroupRepo,
		inspectionItemRepo,
		execRecordRepo,
		pgwAdapter,
		inspectionItemService,
	)
	sched.RegisterExecutor(inspectionExecutor)

	// 创建新表拨测任务执行器（使用 inspection_tasks 表的 probe 类型任务）
	inspectionTaskRepoAdapter := biz.NewInspectionTaskRepoAdapter(inspectionTaskRepo)
	probeV2Executor := biz.NewNetworkProbeV2Executor(
		inspectionTaskRepoAdapter,
		configRepo,
		resultRepo,
		pgwRepo,
		groupLookup,
	)
	probeV2Executor.SetVariableResolver(variableResolver)
	sched.RegisterExecutor(probeV2Executor)

	// 初始化 Redis Counter 并注入各执行器
	redisCounter := metrics.NewRedisCounter(redisClient, "srehub:counter")
	executor.SetRedisCounter(redisCounter)
	probeV2Executor.SetRedisCounter(redisCounter)
	inspectionExecutor.SetRedisCounter(redisCounter)

	// Start scheduler（所有执行器注册完成后启动）
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

	return NewHTTPServer(probeConfigSvc, probeTaskSvc, pgwSvc, variableSvc, sched, executor, probeV2Executor, healthExecutor, inspectionGroupService, inspectionItemService, inspectionRecordService, inspectionTaskService, executionRecordService, inspectionExecutor, hostRepo, inspectionGroupRepo, inspectionItemRepo, inspectionRecordRepo, execRecordRepo)
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
