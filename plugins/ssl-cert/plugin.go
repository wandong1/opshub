package sslcert

import (
	"context"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ydcloud-dy/opshub/internal/plugin"
	"github.com/ydcloud-dy/opshub/pkg/utils"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/deployer"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/model"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/server"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/service"
)

// Plugin SSL证书管理插件实现
type Plugin struct {
	db        *gorm.DB
	scheduler *service.Scheduler

	// 配置
	acmeEmail   string
	acmeStaging bool

	ctx       context.Context
	cancelCtx context.CancelFunc
}

// New 创建插件实例
func New() *Plugin {
	// 从环境变量读取ACME邮箱，如果未设置则使用空字符串（将在申请时报错提示用户配置）
	acmeEmail := os.Getenv("OPSHUB_ACME_EMAIL")
	if acmeEmail == "" {
		// 尝试其他常见的环境变量名
		acmeEmail = os.Getenv("ACME_EMAIL")
	}
	if acmeEmail == "" {
		acmeEmail = os.Getenv("LETSENCRYPT_EMAIL")
	}

	// 是否使用Let's Encrypt测试环境
	acmeStaging := os.Getenv("OPSHUB_ACME_STAGING") == "true"

	return &Plugin{
		acmeEmail:   acmeEmail,
		acmeStaging: acmeStaging,
	}
}

// NewWithConfig 创建带配置的插件实例
func NewWithConfig(acmeEmail string, acmeStaging bool) *Plugin {
	return &Plugin{
		acmeEmail:   acmeEmail,
		acmeStaging: acmeStaging,
	}
}

// Name 返回插件名称
func (p *Plugin) Name() string {
	return "ssl-cert"
}

// Description 返回插件描述
func (p *Plugin) Description() string {
	return "SSL证书自动续期插件 - 支持Let's Encrypt和云厂商证书服务,自动部署到Nginx和K8s"
}

// Version 返回插件版本
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Author 返回插件作者
func (p *Plugin) Author() string {
	return "J"
}

// Enable 启用插件
func (p *Plugin) Enable(db *gorm.DB) error {
	p.db = db

	// 自动迁移所有插件相关的表
	models := []interface{}{
		&model.SSLCertificate{},
		&model.DNSProvider{},
		&model.DeployConfig{},
		&model.RenewTask{},
	}

	for _, m := range models {
		if err := db.AutoMigrate(m); err != nil {
			return err
		}
	}

	// 创建上下文
	p.ctx, p.cancelCtx = context.WithCancel(context.Background())

	// 创建部署器依赖
	deployerDeps := &deployer.Dependencies{
		HostGetter:    NewHostGetter(db),
		ClusterGetter: NewClusterGetter(db),
	}

	// 创建并启动调度器
	p.scheduler = service.NewScheduler(db, deployerDeps, p.acmeEmail, p.acmeStaging, time.Hour)
	p.scheduler.Start()

	// 初始化插件按钮权限
	p.initPermissions(db)

	return nil
}

// initPermissions 初始化SSL-Cert插件按钮权限
func (p *Plugin) initPermissions(db *gorm.DB) {
	utils.EnsureMenu(db, "ssl-cert", "SSL证书", 1, "", "/ssl-cert", "", "Lock", 37)
	utils.EnsureMenu(db, "ssl-certificates", "证书管理", 2, "ssl-cert", "/ssl-cert/certificates", "ssl-cert/Certificates", "Document", 1)
	utils.EnsureMenu(db, "ssl-dns-providers", "DNS配置", 2, "ssl-cert", "/ssl-cert/dns-providers", "ssl-cert/DnsProviders", "Connection", 2)
	utils.EnsureMenu(db, "ssl-deploy-configs", "部署配置", 2, "ssl-cert", "/ssl-cert/deploy-configs", "ssl-cert/DeployConfigs", "Upload", 3)

	// 证书管理
	utils.EnsureMenuPermissions(db, "ssl-certificates", []utils.MenuPermission{
		{Code: "certs:list", Name: "查看证书列表", ApiMethod: "GET", ApiPath: "/api/v1/plugins/ssl-cert/certificates", Sort: 1},
		{Code: "certs:create", Name: "创建证书", ApiMethod: "POST", ApiPath: "/api/v1/plugins/ssl-cert/certificates", Sort: 2},
		{Code: "certs:update", Name: "编辑证书", ApiMethod: "PUT", ApiPath: "/api/v1/plugins/ssl-cert/certificates/:id", Sort: 3},
		{Code: "certs:delete", Name: "删除证书", ApiMethod: "DELETE", ApiPath: "/api/v1/plugins/ssl-cert/certificates/:id", Sort: 4},
		{Code: "certs:renew", Name: "续期证书", ApiMethod: "POST", ApiPath: "/api/v1/plugins/ssl-cert/certificates/:id/renew", Sort: 5},
		{Code: "certs:download", Name: "下载证书", ApiMethod: "GET", ApiPath: "/api/v1/plugins/ssl-cert/certificates/:id/download", Sort: 6},
	})

	// DNS提供商
	utils.EnsureMenuPermissions(db, "ssl-dns-providers", []utils.MenuPermission{
		{Code: "dns-providers:list", Name: "查看DNS提供商", ApiMethod: "GET", ApiPath: "/api/v1/plugins/ssl-cert/dns-providers", Sort: 1},
		{Code: "dns-providers:create", Name: "创建DNS提供商", ApiMethod: "POST", ApiPath: "/api/v1/plugins/ssl-cert/dns-providers", Sort: 2},
		{Code: "dns-providers:update", Name: "编辑DNS提供商", ApiMethod: "PUT", ApiPath: "/api/v1/plugins/ssl-cert/dns-providers/:id", Sort: 3},
		{Code: "dns-providers:delete", Name: "删除DNS提供商", ApiMethod: "DELETE", ApiPath: "/api/v1/plugins/ssl-cert/dns-providers/:id", Sort: 4},
	})

	// 部署配置
	utils.EnsureMenuPermissions(db, "ssl-deploy-configs", []utils.MenuPermission{
		{Code: "deploy-configs:list", Name: "查看部署配置", ApiMethod: "GET", ApiPath: "/api/v1/plugins/ssl-cert/deploy-configs", Sort: 1},
		{Code: "deploy-configs:create", Name: "创建部署配置", ApiMethod: "POST", ApiPath: "/api/v1/plugins/ssl-cert/deploy-configs", Sort: 2},
		{Code: "deploy-configs:update", Name: "编辑部署配置", ApiMethod: "PUT", ApiPath: "/api/v1/plugins/ssl-cert/deploy-configs/:id", Sort: 3},
		{Code: "deploy-configs:delete", Name: "删除部署配置", ApiMethod: "DELETE", ApiPath: "/api/v1/plugins/ssl-cert/deploy-configs/:id", Sort: 4},
		{Code: "deploy-configs:deploy", Name: "执行部署", ApiMethod: "POST", ApiPath: "/api/v1/plugins/ssl-cert/deploy-configs/:id/deploy", Sort: 5},
	})

	utils.AssignMenusToAdminRole(db)
}

// Disable 禁用插件
func (p *Plugin) Disable(db *gorm.DB) error {
	// 停止调度器
	if p.scheduler != nil {
		p.scheduler.Stop()
	}

	// 取消上下文
	if p.cancelCtx != nil {
		p.cancelCtx()
	}

	return nil
}

// RegisterRoutes 注册路由
func (p *Plugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	// 创建部署器依赖
	deployerDeps := &deployer.Dependencies{
		HostGetter:    NewHostGetter(db),
		ClusterGetter: NewClusterGetter(db),
	}

	// 创建服务
	certSvc := service.NewCertificateService(db, deployerDeps, p.acmeEmail, p.acmeStaging)
	dnsSvc := service.NewDNSProviderService(db)
	deploySvc := service.NewDeployService(db, deployerDeps)
	taskSvc := service.NewTaskService(db)

	// 注册路由
	sslCertGroup := router.Group("/ssl-cert")
	server.RegisterRoutes(sslCertGroup, certSvc, dnsSvc, deploySvc, taskSvc)
}

// GetMenus 获取插件菜单配置
func (p *Plugin) GetMenus() []plugin.MenuConfig {
	return []plugin.MenuConfig{
		{
			Name: "SSL证书",
			Path: "/ssl-cert",
			Icon: "Key",
			Sort: 50,
		},
		{
			Name:       "证书管理",
			Path:       "/ssl-cert/certificates",
			Icon:       "Document",
			Sort:       1,
			ParentPath: "/ssl-cert",
		},
		{
			Name:       "DNS配置",
			Path:       "/ssl-cert/dns-providers",
			Icon:       "Connection",
			Sort:       2,
			ParentPath: "/ssl-cert",
		},
		{
			Name:       "部署配置",
			Path:       "/ssl-cert/deploy-configs",
			Icon:       "Upload",
			Sort:       3,
			ParentPath: "/ssl-cert",
		},
		{
			Name:       "任务记录",
			Path:       "/ssl-cert/tasks",
			Icon:       "List",
			Sort:       4,
			ParentPath: "/ssl-cert",
		},
	}
}
