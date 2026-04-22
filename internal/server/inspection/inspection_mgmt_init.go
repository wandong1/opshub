package inspection

import (
	"fmt"

	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	inspectionbiz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	inspectionmgmtbiz "github.com/ydcloud-dy/opshub/internal/biz/inspection_mgmt"
	systembiz "github.com/ydcloud-dy/opshub/internal/biz/system"
	inspectiondata "github.com/ydcloud-dy/opshub/internal/data/inspection"
	inspectionmgmtdata "github.com/ydcloud-dy/opshub/internal/data/inspection_mgmt"
	"github.com/ydcloud-dy/opshub/internal/server/agent"
	inspectionmgmtsvc "github.com/ydcloud-dy/opshub/internal/service/inspection_mgmt"
	"gorm.io/gorm"
)

// InitInspectionMgmtServices 初始化巡检管理服务
func InitInspectionMgmtServices(
	db *gorm.DB,
	hostRepo assetbiz.HostRepo,
	serviceLabelRepo assetbiz.ServiceLabelRepo,
	credentialRepo assetbiz.CredentialRepo,
	assetGroupRepo assetbiz.AssetGroupRepo,
	agentHub *agent.AgentHub,
	configUseCase *systembiz.ConfigUseCase,
) (
	*inspectionmgmtsvc.GroupService,
	*inspectionmgmtsvc.ItemService,
	*inspectionmgmtsvc.RecordService,
	*inspectionmgmtsvc.TaskService,
	*inspectionmgmtsvc.ExecutionRecordService,
	*inspectionmgmtsvc.CleanupService,
	inspectionmgmtdata.GroupRepository,
	inspectionmgmtdata.ItemRepository,
	inspectionmgmtdata.RecordRepository,
	inspectionmgmtdata.TaskRepository,
	inspectionmgmtdata.ExecutionRecordRepository,
) {
	fmt.Println("=========初始化task 数据库==========")
	// 自动迁移数据库表
	_ = db.AutoMigrate(
		&inspectionmgmtdata.InspectionGroup{},
		&inspectionmgmtdata.InspectionItem{},
		&inspectionmgmtdata.InspectionTask{},
		&inspectionmgmtdata.InspectionRecord{},
		&inspectionmgmtdata.InspectionExecutionRecord{},
		&inspectionmgmtdata.InspectionExecutionDetail{},
	)
	fmt.Println("=========初始化task 数据库==========")

	// 初始化 Repository
	groupRepo := inspectionmgmtdata.NewGroupRepository(db)
	itemRepo := inspectionmgmtdata.NewItemRepository(db)
	recordRepo := inspectionmgmtdata.NewRecordRepository(db)
	taskRepo := inspectionmgmtdata.NewTaskRepository(db)
	execRecordRepo := inspectionmgmtdata.NewExecutionRecordRepository(db)

	// 初始化执行器
	cmdExecutor := inspectionmgmtbiz.NewCommandExecutor(agentHub, credentialRepo)

	// 初始化拨测配置仓储和变量仓储
	probeConfigRepo := inspectiondata.NewProbeConfigRepo(db)
	variableRepo := inspectiondata.NewProbeVariableRepo(db)

	// 初始化拨测模块的变量解析器（用于解析拨测配置中的变量）
	probeVariableResolver := inspectionbiz.NewVariableResolver(variableRepo)

	// 初始化拨测执行器（传入变量解析器）
	probeExecutor := inspectionmgmtsvc.NewProbeExecutor(probeConfigRepo, probeVariableResolver)

	// 初始化巡检模块的变量解析器（用于解析巡检项中的变量）
	inspectionVariableResolver := inspectionmgmtsvc.NewVariableResolver(variableRepo, groupRepo)

	// 初始化 Service
	groupService := inspectionmgmtsvc.NewGroupService(groupRepo, itemRepo)
	itemService := inspectionmgmtsvc.NewItemService(itemRepo, groupRepo, recordRepo, hostRepo, assetGroupRepo, serviceLabelRepo, cmdExecutor, probeExecutor, inspectionVariableResolver)
	recordService := inspectionmgmtsvc.NewRecordService(recordRepo, itemRepo, groupRepo)
	recordService.SetHostRepo(hostRepo)
	taskService := inspectionmgmtsvc.NewTaskService(taskRepo)
	executionRecordService := inspectionmgmtsvc.NewExecutionRecordService(execRecordRepo)

	// 初始化清理服务
	cleanupService := inspectionmgmtsvc.NewCleanupService(recordRepo, configUseCase)

	return groupService, itemService, recordService, taskService, executionRecordService, cleanupService, groupRepo, itemRepo, recordRepo, taskRepo, execRecordRepo
}
