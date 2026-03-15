package inspection

import (
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/internal/plugin"
	"github.com/ydcloud-dy/opshub/internal/server/agent"
	"github.com/ydcloud-dy/opshub/plugins/inspection/executor"
	"github.com/ydcloud-dy/opshub/plugins/inspection/model"
	"github.com/ydcloud-dy/opshub/plugins/inspection/repository"
	"github.com/ydcloud-dy/opshub/plugins/inspection/server"
	"github.com/ydcloud-dy/opshub/plugins/inspection/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Plugin struct {
	db       *gorm.DB
	handler  *server.Handler
	agentHub *agent.AgentHub
	hostRepo assetbiz.HostRepo
}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "inspection"
}

func (p *Plugin) Description() string {
	return "智能巡检系统，支持命令、脚本、PromQL 三种执行方式"
}

func (p *Plugin) Version() string {
	return "1.0.0"
}

func (p *Plugin) Author() string {
	return "OpsHub Team"
}

func (p *Plugin) Enable(db *gorm.DB) error {
	p.db = db

	// 自动迁移数据库表
	if err := db.AutoMigrate(
		&model.InspectionGroup{},
		&model.InspectionItem{},
		&model.InspectionTask{},
		&model.InspectionRecord{},
	); err != nil {
		return err
	}

	// 初始化 Repository
	groupRepo := repository.NewGroupRepository(db)
	itemRepo := repository.NewItemRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	recordRepo := repository.NewRecordRepository(db)

	// 初始化执行器
	cmdExecutor := executor.NewCommandExecutor(p.agentHub)

	// 初始化 Service
	groupService := service.NewGroupService(groupRepo)
	itemService := service.NewItemService(itemRepo, groupRepo, recordRepo, p.hostRepo, cmdExecutor)
	taskService := service.NewTaskService(taskRepo)
	recordService := service.NewRecordService(recordRepo, itemRepo)

	// 初始化 Handler
	p.handler = server.NewHandler(groupService, itemService, taskService, recordService)

	return nil
}

func (p *Plugin) Disable(db *gorm.DB) error {
	return nil
}

func (p *Plugin) RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	server.RegisterRoutes(r, p.handler)
}

func (p *Plugin) GetMenus() []plugin.MenuConfig {
	return []plugin.MenuConfig{
		{
			Name:       "智能巡检",
			Path:       "/inspection",
			Icon:       "icon-check-circle",
			Sort:       60,
			Hidden:     false,
			ParentPath: "",
		},
		{
			Name:       "巡检组管理",
			Path:       "/inspection/groups",
			Icon:       "icon-folder",
			Sort:       1,
			Hidden:     false,
			ParentPath: "/inspection",
		},
		{
			Name:       "巡检项管理",
			Path:       "/inspection/items",
			Icon:       "icon-list",
			Sort:       2,
			Hidden:     false,
			ParentPath: "/inspection",
		},
		{
			Name:       "定时任务",
			Path:       "/inspection/tasks",
			Icon:       "icon-schedule",
			Sort:       3,
			Hidden:     false,
			ParentPath: "/inspection",
		},
		{
			Name:       "执行记录",
			Path:       "/inspection/records",
			Icon:       "icon-history",
			Sort:       4,
			Hidden:     false,
			ParentPath: "/inspection",
		},
	}
}

// SetAgentHub 注入 AgentHub
func (p *Plugin) SetAgentHub(hub *agent.AgentHub) {
	p.agentHub = hub
}

// SetHostRepo 注入 HostRepo
func (p *Plugin) SetHostRepo(repo assetbiz.HostRepo) {
	p.hostRepo = repo
}
