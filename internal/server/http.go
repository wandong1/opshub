// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/ydcloud-dy/opshub/docs"
	"github.com/ydcloud-dy/opshub/internal/cache"
	"github.com/ydcloud-dy/opshub/internal/conf"
	agentrepo "github.com/ydcloud-dy/opshub/internal/data/agent"
	rbacdata "github.com/ydcloud-dy/opshub/internal/data/rbac"
	"github.com/ydcloud-dy/opshub/internal/plugin"
	assetserver "github.com/ydcloud-dy/opshub/internal/server/asset"
	agentserver "github.com/ydcloud-dy/opshub/internal/server/agent"
	auditserver "github.com/ydcloud-dy/opshub/internal/server/audit"
	identityserver "github.com/ydcloud-dy/opshub/internal/server/identity"
	alertserver "github.com/ydcloud-dy/opshub/internal/server/alert"
	inspectionserver "github.com/ydcloud-dy/opshub/internal/server/inspection"
	"github.com/ydcloud-dy/opshub/internal/server/rbac"
	systemserver "github.com/ydcloud-dy/opshub/internal/server/system"
	"github.com/ydcloud-dy/opshub/internal/service"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/middleware"
	inspectionplugin "github.com/ydcloud-dy/opshub/plugins/inspection"
	k8splugin "github.com/ydcloud-dy/opshub/plugins/kubernetes"
	monitorplugin "github.com/ydcloud-dy/opshub/plugins/monitor"
	nginxplugin "github.com/ydcloud-dy/opshub/plugins/nginx"
	sslcertplugin "github.com/ydcloud-dy/opshub/plugins/ssl-cert"
	taskplugin "github.com/ydcloud-dy/opshub/plugins/task"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HTTPServer HTTP服务器
type HTTPServer struct {
	server           *http.Server
	conf             *conf.Config
	svc              *service.Service
	db               *gorm.DB
	redisClient      *redis.Client
	pluginMgr        *plugin.Manager
	uploadSrv        *UploadServer
	inspectionServer *inspectionserver.HTTPServer
	alertServer      *alertserver.HTTPServer
	grpcServer       *agentserver.GRPCServer
	cacheManager     *cache.CacheManager
	scheduler        *cache.CacheSyncScheduler
	evalCancel       context.CancelFunc // 告警评估引擎的 cancel 函数
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(conf *conf.Config, svc *service.Service, db *gorm.DB, redisClient *redis.Client, grpcServer *agentserver.GRPCServer) *HTTPServer {
	// 设置Gin模式
	gin.SetMode(conf.Server.Mode)

	// 创建路由
	router := gin.New()

	// 使用中间件
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.AuditLogOperation(db))

	// 创建插件管理器
	pluginMgr := plugin.NewManager(db)

	// 创建上传服务
	uploadDir := "./web/public/uploads"
	uploadURL := "/uploads"
	uploadSrv := NewUploadServer(db, uploadDir, uploadURL)

	// 注册 Kubernetes 插件
	if err := pluginMgr.Register(k8splugin.New()); err != nil {
		appLogger.Error("注册Kubernetes插件失败", zap.Error(err))
	}

	// 注册 Task 插件
	taskPlugin := taskplugin.New()
	if grpcServer != nil {
		taskPlugin.SetAgentHub(grpcServer.Hub())
	}
	if err := pluginMgr.Register(taskPlugin); err != nil {
		appLogger.Error("注册Task插件失败", zap.Error(err))
	}

	// 注册 Monitor 插件
	if err := pluginMgr.Register(monitorplugin.New()); err != nil {
		appLogger.Error("注册Monitor插件失败", zap.Error(err))
	}

	// 注册 Nginx 插件
	if err := pluginMgr.Register(nginxplugin.New()); err != nil {
		appLogger.Error("注册Nginx插件失败", zap.Error(err))
	}

	// 注册 ssl-cert 插件
	if err := pluginMgr.Register(sslcertplugin.New()); err != nil {
		appLogger.Error("注册ssl-cert插件失败", zap.Error(err))
	}

	// ========== 初始化缓存系统 ==========
	agentRepo := agentrepo.NewRepository(db)

	// 转换配置
	cacheConfig := cache.ConvertConfigToCacheConfig(struct {
		BatchFlushInterval int
		BatchSize          int
		BatchQueueMaxSize  int
		RedisTTL           int
		LockTimeout        int
		OfflineThreshold   int
	}{
		BatchFlushInterval: conf.Cache.BatchFlushInterval,
		BatchSize:          conf.Cache.BatchSize,
		BatchQueueMaxSize:  conf.Cache.BatchQueueMaxSize,
		RedisTTL:           conf.Cache.RedisTTL,
		LockTimeout:        conf.Cache.LockTimeout,
		OfflineThreshold:   conf.Cache.OfflineThreshold,
	})

	cacheManager := cache.NewCacheManager(redisClient, agentRepo, db, cacheConfig)
	scheduler := cache.NewCacheSyncScheduler(cacheManager)

	// 预热缓存
	if err := scheduler.WarmupCache(); err != nil {
		appLogger.Warn("缓存预热失败", zap.Error(err))
	}

	// 启动定期任务
	scheduler.Start()

	// 启动批量同步 Worker
	cacheManager.StartBatchWorker()

	// 注入到 AgentService
	grpcServer.GetAgentService().SetCacheManager(cacheManager)
	// ========== 缓存系统初始化完成 ==========

	// 注册路由
	s := &HTTPServer{
		conf:         conf,
		svc:          svc,
		db:           db,
		redisClient:  redisClient,
		pluginMgr:    pluginMgr,
		uploadSrv:    uploadSrv,
		grpcServer:   grpcServer,
		cacheManager: cacheManager,
		scheduler:    scheduler,
	}

	// 先启用所有插件（在注册路由之前）
	s.enablePlugins()

	// 注册路由（插件启用后才能注册路由）
	s.registerRoutes(router, conf.Server.JWTSecret)

	// 创建HTTP服务器
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", conf.Server.HttpPort),
		Handler:      router,
		ReadTimeout:  time.Duration(conf.Server.ReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(conf.Server.WriteTimeout) * time.Millisecond,
	}

	return s
}

// registerRoutes 注册路由
func (s *HTTPServer) registerRoutes(router *gin.Engine, jwtSecret string) {
	// Swagger 文档 — 动态设置，让 Swagger UI 使用当前页面地址
	docs.SwaggerInfo.Host = ""
	docs.SwaggerInfo.BasePath = "/"
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 健康检查
	router.GET("/health", s.svc.Health)

	// 静态文件服务 - 上传的文件
	router.Static("/uploads", "./web/public/uploads")

	// 创建 RBAC 服务
	userService, roleService, departmentService, menuService, positionService, captchaService, assetPermissionService, authMiddleware := rbac.NewRBACServices(s.db, jwtSecret, s.redisClient)

	// RBAC 路由
	rbacServer := rbac.NewHTTPServer(userService, roleService, departmentService, menuService, positionService, captchaService, assetPermissionService, authMiddleware)
	rbacServer.RegisterRoutes(router)

	// 创建 System 服务
	uploadDir := "./web/public/uploads"
	configService, apiKeyService, configUseCase, apiKeyUseCase := systemserver.NewSystemServices(s.db, s.redisClient, uploadDir)

	// 设置配置用例到用户服务（用于密码验证和登录锁定）
	userService.SetConfigUseCase(configUseCase)

	// 设置 API Key 用例到认证中间件
	authMiddleware.SetAPIKeyUseCase(apiKeyUseCase)

	// 创建 Audit 服务
	operationLogService, loginLogService, dataLogService, mwAuditLogService := auditserver.NewAuditServices(s.db)

	// 创建 System HTTP Server
	systemHTTPServer := systemserver.NewHTTPServer(configService, apiKeyService)

	// 创建 Asset 服务
	assetGroupService, hostService, middlewareService, mwPermissionService, serviceLabelService, websiteService, terminalManager, hostUseCase, websiteUseCase, serviceLabelRepo, hostRepo, credentialRepo := assetserver.NewAssetServices(s.db)

	// 设置Agent命令执行工厂，使主机采集支持Agent方式
	if s.grpcServer != nil {
		hostUseCase.SetAgentCommandFactory(agentserver.NewAgentCommandFactory(s.grpcServer.Hub()))
	}

	// 注册 Inspection 插件
	inspectionPlugin := inspectionplugin.New()
	if s.grpcServer != nil {
		inspectionPlugin.SetAgentHub(s.grpcServer.Hub())
	}
	inspectionPlugin.SetHostRepo(hostRepo)
	if err := s.pluginMgr.Register(inspectionPlugin); err != nil {
		appLogger.Error("注册Inspection插件失败", zap.Error(err))
	}

	// 创建 WebsiteProxyHandler
	var websiteProxyHandler *assetserver.WebsiteProxyHandler
	if s.grpcServer != nil {
		websiteProxyHandler = assetserver.NewWebsiteProxyHandler(websiteUseCase, s.grpcServer.Hub())
	}

	// 设置authMiddleware的assetPermissionRepo
	assetPermissionRepo := rbacdata.NewAssetPermissionRepo(s.db)
	authMiddleware.SetAssetPermissionRepo(assetPermissionRepo)

	// 设置authMiddleware的middlewarePermissionRepo
	mwPermissionRepo := rbacdata.NewMiddlewarePermissionRepo(s.db)
	authMiddleware.SetMiddlewarePermissionRepo(mwPermissionRepo)

	// Asset 路由
	assetServer := assetserver.NewHTTPServer(assetGroupService, hostService, middlewareService, mwPermissionService, serviceLabelService, websiteService, websiteProxyHandler, terminalManager, s.db, authMiddleware)

	// API v1 - 公开接口(不需要认证)
	public := router.Group("/api/v1/public")
	{
		public.GET("/example", s.svc.Example)
	}

	// Agent安装包下载（无需认证，包ID随机且30分钟过期）
	if s.grpcServer != nil {
		router.GET("/api/v1/agents/install-package/:id", func(c *gin.Context) {
			agentserver.DownloadInstallPackagePublic(c)
		})
	}

	// API v1 - 需要认证的接口
	v1 := router.Group("/api/v1")
	v1.Use(authMiddleware.AuthRequired())
	v1.Use(authMiddleware.RequirePermission())
	{
		// Audit 路由
		auditHTTPServer := auditserver.NewHTTPService(operationLogService, loginLogService, dataLogService, mwAuditLogService)
		auditHTTPServer.RegisterRoutes(v1)

		// 注册 Asset 路由
		assetServer.RegisterRoutes(v1)

		// 注册 Identity 路由
		identityServer, err := identityserver.NewIdentityServices(s.db)
		if err != nil {
			appLogger.Error("创建Identity服务失败", zap.Error(err))
		} else {
			identityServer.RegisterRoutes(v1)
		}

		// 注册 Inspection（智能巡检）路由
		var agentHub *agentserver.AgentHub
		if s.grpcServer != nil {
			agentHub = s.grpcServer.Hub()
		}
		s.inspectionServer = inspectionserver.NewInspectionServices(s.db, s.redisClient, hostRepo, credentialRepo, agentHub, configUseCase)
		s.inspectionServer.RegisterRoutes(v1)

		// 设置Agent命令执行工厂，使拨测支持Agent方式
		if s.grpcServer != nil {
			s.inspectionServer.SetAgentCommandFactory(agentserver.NewAgentCommandFactory(s.grpcServer.Hub()))
		}

		// 注入 TeleAI Authorization 自动填充全局开关（per-probe 的 appKey/region 存储在拨测配置中）
		if teleAICfg, err := configUseCase.GetTeleAIAuthConfig(context.Background()); err == nil && teleAICfg.Enabled {
			s.inspectionServer.SetTeleAIEnabled(true)
		}

		// 上传接口
		v1.POST("/upload/avatar", s.uploadSrv.UploadAvatar)
		v1.PUT("/profile/avatar", s.uploadSrv.UpdateUserAvatar)

		// 系统配置路由
		systemHTTPServer.RegisterRoutes(v1, public)
		// Grafana 代理：路由路径与 Grafana sub_path 一致，注册在根路由（无需认证）
		systemHTTPServer.RegisterGrafanaProxy(router)

		// Agent路由
		if s.grpcServer != nil {
			s.grpcServer.SetServiceLabelRepo(serviceLabelRepo)
			s.grpcServer.SetHostRepo(hostUseCase)
			agentHTTPServer := agentserver.NewHTTPServer(s.grpcServer, hostUseCase, s.db, authMiddleware)
			agentHTTPServer.RegisterRoutes(v1)
		}

		// 注册 Alert（告警管理）路由
		s.alertServer = alertserver.NewAlertServices(s.db, s.redisClient)
		s.alertServer.RegisterRoutes(v1)
	}

	// 插件路由
	pluginsGroup := router.Group("/api/v1/plugins")
	pluginsGroup.Use(authMiddleware.AuthRequired())
	pluginsGroup.Use(authMiddleware.RequirePermission())
	s.pluginMgr.RegisterAllRoutes(pluginsGroup)

	// 插件管理接口
	pluginInfoGroup := router.Group("/api/v1/plugins")
	pluginInfoGroup.Use(authMiddleware.AuthRequired())
	{
		pluginInfoGroup.GET("", s.listPlugins)
		pluginInfoGroup.GET("/:name", s.getPlugin)
		pluginInfoGroup.GET("/:name/menus", s.getPluginMenus)
		pluginInfoGroup.POST("/:name/enable", s.enablePlugin)
		pluginInfoGroup.POST("/:name/disable", s.disablePlugin)
		pluginInfoGroup.POST("/upload", s.uploadSrv.UploadPlugin)
		pluginInfoGroup.DELETE("/:name/uninstall", s.uploadSrv.UninstallPlugin)
	}

	// 前端静态文件服务（后面会用到）
	// router.Static("/assets", "./web/dist/assets")
	// router.NoRoute(func(c *gin.Context) {
	//     c.File("./web/dist/index.html")
	// })
}

// enablePlugins 启用所有已注册的插件
func (s *HTTPServer) enablePlugins() {
	for _, p := range s.pluginMgr.GetAllPlugins() {
		if err := s.pluginMgr.Enable(p.Name()); err != nil {
			appLogger.Error("启用插件失败",
				zap.String("plugin", p.Name()),
				zap.Error(err),
			)
		} else {
			appLogger.Info("插件启用成功",
				zap.String("plugin", p.Name()),
				zap.String("version", p.Version()),
			)
		}
	}
}

// listPlugins 获取所有插件列表
// @Summary 获取插件列表
// @Description 获取系统中所有已注册的插件列表
// @Tags 插件管理
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} map[string]interface{} "插件列表"
// @Router /api/v1/plugins [get]
func (s *HTTPServer) listPlugins(c *gin.Context) {
	plugins := s.pluginMgr.GetAllPlugins()
	result := make([]map[string]interface{}, 0, len(plugins))

	for _, p := range plugins {
		enabled := s.pluginMgr.IsEnabled(p.Name())
		appLogger.Info("获取插件状态",
			zap.String("plugin", p.Name()),
			zap.Bool("enabled", enabled),
		)
		result = append(result, map[string]interface{}{
			"name":        p.Name(),
			"description": p.Description(),
			"version":     p.Version(),
			"author":      p.Author(),
			"enabled":     enabled,
		})
	}

	appLogger.Info("返回插件列表", zap.Int("count", len(result)))
	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// getPlugin 获取插件详情
// @Summary 获取插件详情
// @Description 获取指定插件的详细信息
// @Tags 插件管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "插件名称"
// @Success 200 {object} map[string]interface{} "插件详情"
// @Failure 404 {object} map[string]interface{} "插件不存在"
// @Router /api/v1/plugins/{name} [get]
func (s *HTTPServer) getPlugin(c *gin.Context) {
	name := c.Param("name")
	plugin, exists := s.pluginMgr.GetPlugin(name)
	if !exists {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "plugin not found",
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": map[string]interface{}{
			"name":        plugin.Name(),
			"description": plugin.Description(),
			"version":     plugin.Version(),
			"author":      plugin.Author(),
			"enabled":     s.pluginMgr.IsEnabled(name),
		},
	})
}

// getPluginMenus 获取插件的菜单配置
// @Summary 获取插件菜单
// @Description 获取指定插件的菜单配置信息
// @Tags 插件管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "插件名称"
// @Success 200 {object} map[string]interface{} "菜单配置"
// @Failure 404 {object} map[string]interface{} "插件不存在"
// @Router /api/v1/plugins/{name}/menus [get]
func (s *HTTPServer) getPluginMenus(c *gin.Context) {
	name := c.Param("name")
	plugin, exists := s.pluginMgr.GetPlugin(name)
	if !exists {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "plugin not found",
		})
		return
	}

	menus := plugin.GetMenus()
	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    menus,
	})
}

// enablePlugin 启用插件
// @Summary 启用插件
// @Description 启用指定的插件
// @Tags 插件管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "插件名称"
// @Success 200 {object} map[string]interface{} "启用成功"
// @Failure 500 {object} map[string]interface{} "启用失败"
// @Router /api/v1/plugins/{name}/enable [post]
func (s *HTTPServer) enablePlugin(c *gin.Context) {
	name := c.Param("name")

	if err := s.pluginMgr.Enable(name); err != nil {
		appLogger.Error("启用插件失败",
			zap.String("plugin", name),
			zap.Error(err),
		)
		c.JSON(500, gin.H{
			"code":    500,
			"message": fmt.Sprintf("启用插件失败: %v", err),
		})
		return
	}

	appLogger.Info("插件启用成功", zap.String("plugin", name))
	c.JSON(200, gin.H{
		"code":    0,
		"message": "插件启用成功，请刷新页面以生效",
	})
}

// disablePlugin 禁用插件
// @Summary 禁用插件
// @Description 禁用指定的插件
// @Tags 插件管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "插件名称"
// @Success 200 {object} map[string]interface{} "禁用成功"
// @Failure 500 {object} map[string]interface{} "禁用失败"
// @Router /api/v1/plugins/{name}/disable [post]
func (s *HTTPServer) disablePlugin(c *gin.Context) {
	name := c.Param("name")

	if err := s.pluginMgr.Disable(name); err != nil {
		appLogger.Error("禁用插件失败",
			zap.String("plugin", name),
			zap.Error(err),
		)
		c.JSON(500, gin.H{
			"code":    500,
			"message": fmt.Sprintf("禁用插件失败: %v", err),
		})
		return
	}

	appLogger.Info("插件禁用成功", zap.String("plugin", name))
	c.JSON(200, gin.H{
		"code":    0,
		"message": "插件禁用成功，请刷新页面以生效",
	})
}

// Start 启动服务器
func (s *HTTPServer) Start() error {
	appLogger.Info("HTTP服务器启动",
		zap.String("addr", s.server.Addr),
		zap.String("mode", s.conf.Server.Mode),
	)

	// 启动告警评估引擎
	if s.alertServer != nil {
		evalCtx, cancel := context.WithCancel(context.Background())
		s.evalCancel = cancel
		go s.alertServer.GetEvalEngine().Start(evalCtx)
		appLogger.Info("告警评估引擎已启动")
	}

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP服务器启动失败: %w", err)
	}

	return nil
}

// Stop 停止服务器
func (s *HTTPServer) Stop(ctx context.Context) error {
	// 停止告警评估引擎
	if s.evalCancel != nil {
		s.evalCancel()
		appLogger.Info("告警评估引擎已停止")
	}

	// 停止批量同步 Worker
	if s.cacheManager != nil {
		s.cacheManager.StopBatchWorker()
		appLogger.Info("批量同步 Worker 已停止")
	}

	// 停止缓存调度器
	if s.scheduler != nil {
		s.scheduler.Stop()
		appLogger.Info("缓存调度器已停止")
	}

	// 停止巡检调度器
	inspectionserver.StopScheduler(s.inspectionServer)

	appLogger.Info("HTTP服务器停止中...")
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP服务器停止失败: %w", err)
	}
	appLogger.Info("HTTP服务器已停止")
	return nil
}
