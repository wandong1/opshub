package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/ydcloud-dy/opshub/internal/conf"
	"github.com/ydcloud-dy/opshub/internal/plugin"
	k8splugin "github.com/ydcloud-dy/opshub/internal/plugins/kubernetes"
	"github.com/ydcloud-dy/opshub/internal/server/rbac"
	"github.com/ydcloud-dy/opshub/internal/service"
	"github.com/ydcloud-dy/opshub/pkg/middleware"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HTTPServer HTTP服务器
type HTTPServer struct {
	server     *http.Server
	conf       *conf.Config
	svc        *service.Service
	db         *gorm.DB
	pluginMgr  *plugin.Manager
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(conf *conf.Config, svc *service.Service, db *gorm.DB) *HTTPServer {
	// 设置Gin模式
	gin.SetMode(conf.Server.Mode)

	// 创建路由
	router := gin.New()

	// 使用中间件
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())

	// 创建插件管理器
	pluginMgr := plugin.NewManager(db)

	// 注册插件
	if err := pluginMgr.Register(k8splugin.New()); err != nil {
		appLogger.Error("注册Kubernetes插件失败", zap.Error(err))
	}

	// 注册路由
	s := &HTTPServer{
		conf:      conf,
		svc:       svc,
		db:        db,
		pluginMgr: pluginMgr,
	}

	s.registerRoutes(router, conf.Server.JWTSecret)

	// 启用所有插件
	s.enablePlugins()

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
	// Swagger 文档
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 健康检查
	router.GET("/health", s.svc.Health)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// 示例接口
		v1.GET("/example", s.svc.Example)

		// 在这里添加更多路由
		// v1.POST("/users", s.svc.CreateUser)
		// v1.GET("/users/:id", s.svc.GetUser)
	}

	// RBAC 路由
	rbacServer := rbac.NewHTTPServer(rbac.NewRBACServices(s.db, jwtSecret))
	rbacServer.RegisterRoutes(router)

	// 插件路由
	pluginsGroup := v1.Group("/plugins")
	s.pluginMgr.RegisterAllRoutes(pluginsGroup)

	// 插件管理接口
	pluginInfoGroup := v1.Group("/plugins")
	{
		pluginInfoGroup.GET("", s.listPlugins)
		pluginInfoGroup.GET("/:name", s.getPlugin)
		pluginInfoGroup.GET("/:name/menus", s.getPluginMenus)
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
func (s *HTTPServer) listPlugins(c *gin.Context) {
	plugins := s.pluginMgr.GetAllPlugins()
	result := make([]map[string]interface{}, 0, len(plugins))

	for _, p := range plugins {
		result = append(result, map[string]interface{}{
			"name":        p.Name(),
			"description": p.Description(),
			"version":     p.Version(),
			"author":      p.Author(),
		})
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// getPlugin 获取插件详情
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
		},
	})
}

// getPluginMenus 获取插件的菜单配置
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

// Start 启动服务器
func (s *HTTPServer) Start() error {
	appLogger.Info("HTTP服务器启动",
		zap.String("addr", s.server.Addr),
		zap.String("mode", s.conf.Server.Mode),
	)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP服务器启动失败: %w", err)
	}

	return nil
}

// Stop 停止服务器
func (s *HTTPServer) Stop(ctx context.Context) error {
	appLogger.Info("HTTP服务器停止中...")
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP服务器停止失败: %w", err)
	}
	appLogger.Info("HTTP服务器已停止")
	return nil
}
