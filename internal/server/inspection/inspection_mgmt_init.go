package inspection

import (
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	inspectionmgmtbiz "github.com/ydcloud-dy/opshub/internal/biz/inspection_mgmt"
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
	credentialRepo assetbiz.CredentialRepo,
	agentHub *agent.AgentHub,
) (
	*inspectionmgmtsvc.GroupService,
	*inspectionmgmtsvc.ItemService,
	*inspectionmgmtsvc.RecordService,
	*inspectionmgmtsvc.TaskService,
	*inspectionmgmtsvc.ExecutionRecordService,
	inspectionmgmtdata.GroupRepository,
	inspectionmgmtdata.ItemRepository,
	inspectionmgmtdata.RecordRepository,
	inspectionmgmtdata.TaskRepository,
	inspectionmgmtdata.ExecutionRecordRepository,
) {
	// 自动迁移数据库表
	_ = db.AutoMigrate(
		&inspectionmgmtdata.InspectionGroup{},
		&inspectionmgmtdata.InspectionItem{},
		&inspectionmgmtdata.InspectionTask{},
		&inspectionmgmtdata.InspectionRecord{},
		&inspectionmgmtdata.InspectionExecutionRecord{},
		&inspectionmgmtdata.InspectionExecutionDetail{},
	)

	// 初始化 Repository
	groupRepo := inspectionmgmtdata.NewGroupRepository(db)
	itemRepo := inspectionmgmtdata.NewItemRepository(db)
	recordRepo := inspectionmgmtdata.NewRecordRepository(db)
	taskRepo := inspectionmgmtdata.NewTaskRepository(db)
	execRecordRepo := inspectionmgmtdata.NewExecutionRecordRepository(db)

	// 初始化执行器
	cmdExecutor := inspectionmgmtbiz.NewCommandExecutor(agentHub, credentialRepo)

	// 初始化拨测配置仓储和拨测执行器
	probeConfigRepo := inspectiondata.NewProbeConfigRepo(db)
	probeExecutor := inspectionmgmtsvc.NewProbeExecutor(probeConfigRepo)

	// 初始化变量仓储和变量解析器
	variableRepo := inspectiondata.NewProbeVariableRepo(db)
	variableResolver := inspectionmgmtsvc.NewVariableResolver(variableRepo, groupRepo)

	// 初始化 Service
	groupService := inspectionmgmtsvc.NewGroupService(groupRepo, itemRepo)
	itemService := inspectionmgmtsvc.NewItemService(itemRepo, groupRepo, recordRepo, hostRepo, cmdExecutor, probeExecutor, variableResolver)
	recordService := inspectionmgmtsvc.NewRecordService(recordRepo, itemRepo, groupRepo)
	recordService.SetHostRepo(hostRepo)
	taskService := inspectionmgmtsvc.NewTaskService(taskRepo)
	executionRecordService := inspectionmgmtsvc.NewExecutionRecordService(execRecordRepo)

	return groupService, itemService, recordService, taskService, executionRecordService, groupRepo, itemRepo, recordRepo, taskRepo, execRecordRepo
}
