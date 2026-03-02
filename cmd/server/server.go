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
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ydcloud-dy/opshub/cmd/root"
	"github.com/ydcloud-dy/opshub/internal/biz"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	auditmodel "github.com/ydcloud-dy/opshub/internal/biz/audit"
	inspectionbiz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	rbacmodel "github.com/ydcloud-dy/opshub/internal/biz/rbac"
	systemmodel "github.com/ydcloud-dy/opshub/internal/biz/system"
	"github.com/ydcloud-dy/opshub/internal/conf"
	dataPkg "github.com/ydcloud-dy/opshub/internal/data"
	systemdata "github.com/ydcloud-dy/opshub/internal/data/system"
	agentmodel "github.com/ydcloud-dy/opshub/internal/agent"
	"github.com/ydcloud-dy/opshub/internal/server"
	agentserver "github.com/ydcloud-dy/opshub/internal/server/agent"
	"github.com/ydcloud-dy/opshub/internal/service"
	rbacservice "github.com/ydcloud-dy/opshub/internal/service/rbac"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
	k8smodel "github.com/ydcloud-dy/opshub/plugins/kubernetes/model"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 全局变量，用于在服务器生命周期内保持连接
var (
	globalData       *dataPkg.Data
	globalRedis      *dataPkg.Redis
	globalHTTPServer *server.HTTPServer
	globalGRPCServer *agentserver.GRPCServer
)

var Cmd = &cobra.Command{
	Use:   "server",
	Short: "启动服务",
	Long:  `启动 OpsHub HTTP 服务器`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// 从命令行参数覆盖配置
		if mode := viper.GetString("mode"); mode != "" {
			viper.Set("server.mode", mode)
		}
		if logLevel := viper.GetString("log-level"); logLevel != "" {
			viper.Set("log.level", logLevel)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// 加载配置
		cfg, err := runServer()
		if err != nil {
			fmt.Printf("启动服务失败: %v\n", err)
			os.Exit(1)
		}

		// 等待中断信号
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		fmt.Println("\n正在关闭服务...")
		ctx := context.Background()
		if err := stopServer(ctx, cfg); err != nil {
			fmt.Printf("关闭服务失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("服务已关闭")
	},
}

func init() {
	root.Cmd.AddCommand(Cmd)
}

func runServer() (*conf.Config, error) {
	// 加载配置
	cfg, err := conf.Load(root.GetConfigFile())
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 初始化日志
	logCfg := &appLogger.Config{
		Level:      cfg.Log.Level,
		Filename:   cfg.Log.Filename,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
		Console:    cfg.Log.Console,
	}
	if err := appLogger.Init(logCfg); err != nil {
		return nil, fmt.Errorf("初始化日志失败: %w", err)
	}
	defer appLogger.Sync()

	appLogger.Info("服务启动中...",
		zap.String("version", "1.0.0"),
		zap.String("mode", cfg.Server.Mode),
	)

	// 初始化数据层
	data, err := dataPkg.NewData(cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化数据层失败: %w", err)
	}
	globalData = data // 保存到全局变量，防止被垃圾回收

	// 初始化Redis
	redis, err := dataPkg.NewRedis(cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化Redis失败: %w", err)
	}
	globalRedis = redis // 保存到全局变量

	// 初始化验证码存储（使用 Redis）
	rbacservice.InitCaptchaStore(redis.Get())

	// 初始化业务层
	biz := biz.NewBiz(data, redis)

	// 初始化服务层
	svc := service.NewService(biz)

	// 自动迁移数据库表
	if err := autoMigrate(data.DB()); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 初始化默认数据
	if err := initDefaultData(data.DB()); err != nil {
		return nil, fmt.Errorf("初始化默认数据失败: %w", err)
	}

	// 启动Agent gRPC服务器（在HTTP服务器之前创建，以便注册路由）
	var grpcServer *agentserver.GRPCServer
	if cfg.Agent.Enabled {
		grpcServer = agentserver.NewGRPCServer(cfg, data.DB())
		globalGRPCServer = grpcServer
	}

	// 初始化HTTP服务器
	httpServer := server.NewHTTPServer(cfg, svc, data.DB(), redis.Get(), grpcServer)
	globalHTTPServer = httpServer // 保存到全局变量

	// 启动gRPC服务器
	if grpcServer != nil {
		go func() {
			if err := grpcServer.Start(); err != nil {
				appLogger.Fatal("Agent gRPC服务器启动失败", zap.Error(err))
			}
		}()
	}

	// 启动服务器
	go func() {
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("HTTP服务器启动失败", zap.Error(err))
		}
	}()

	// 打印启动信息
	printStartupInfo(cfg)

	return cfg, nil
}

// autoMigrate 自动迁移数据库表
func autoMigrate(db *gorm.DB) error {
	// 自动迁移表结构
	if err := db.AutoMigrate(
		&rbacmodel.SysUser{},
		&rbacmodel.SysRole{},
		&rbacmodel.SysDepartment{},
		&rbacmodel.SysMenu{},
		&rbacmodel.SysUserRole{},
		&rbacmodel.SysRoleMenu{},
		&rbacmodel.SysMenuAPI{},
		&rbacmodel.SysPosition{},
		&rbacmodel.SysUserPosition{},
		&rbacmodel.SysRoleAssetPermission{},
		&rbacmodel.SysRoleMiddlewarePermission{},
		// 中间件管理表
		&assetbiz.Middleware{},
		// 主机管理表（含Agent字段）
		&assetbiz.Host{},
		// 系统配置相关表
		&systemmodel.SysConfig{},
		&systemmodel.SysUserLoginAttempt{},
		// Kubernetes 集群相关表
		&models.Cluster{},
		&k8smodel.UserKubeConfig{},
		&k8smodel.K8sUserRoleBinding{},
		// 审计日志相关表
		&auditmodel.SysOperationLog{},
		&auditmodel.SysLoginLog{},
		&auditmodel.SysDataLog{},
		&auditmodel.SysMiddlewareAuditLog{},
		// 智能巡检相关表
		&inspectionbiz.ProbeConfig{},
		&inspectionbiz.ProbeTask{},
		&inspectionbiz.ProbeResult{},
		&inspectionbiz.PushgatewayConfig{},
		&inspectionbiz.ProbeTaskConfig{},
		&inspectionbiz.ProbeVariable{},
		// Agent相关表
		&agentmodel.AgentInfo{},
		// 终端会话审计表
		&assetbiz.TerminalSession{},
		// 服务标签表
		&assetbiz.ServiceLabel{},
	); err != nil {
		return err
	}

	// 为用户表创建虚拟列和唯一索引
	// 问题：MySQL 唯一索引中多个 NULL 值被认为是不同的，无法正确约束
	// 解决：使用虚拟列 is_deleted (0=未删除, 1=已删除) 来创建唯一索引

	// 1. 检查并添加虚拟列 is_deleted
	var columnExists bool
	db.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'sys_user' AND COLUMN_NAME = 'is_deleted'").Scan(&columnExists)

	if !columnExists {
		// 虚拟列不存在，添加它
		if err := db.Exec("ALTER TABLE sys_user ADD COLUMN is_deleted TINYINT(1) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN 0 ELSE 1 END) STORED").Error; err != nil {
			appLogger.Warn("添加虚拟列失败", zap.Error(err))
		} else {
			appLogger.Info("成功添加虚拟列 is_deleted")
		}
	}

	// 2. 删除旧的索引
	db.Exec("DROP INDEX idx_username_deleted_at ON sys_user")
	db.Exec("DROP INDEX idx_email_deleted_at ON sys_user")
	db.Exec("DROP INDEX idx_username_email_deleted_at ON sys_user")

	// 3. 创建新的唯一索引：用户名 + 邮箱 + is_deleted
	// 这样未删除的记录 (is_deleted=0) 中，username + email 的组合必须唯一
	// 已删除的记录 (is_deleted=1) 不会阻止新记录创建
	if err := db.Exec("CREATE UNIQUE INDEX idx_username_email_is_deleted ON sys_user(username, email, is_deleted)").Error; err != nil {
		appLogger.Warn("创建用户名邮箱唯一索引失败", zap.Error(err))
	} else {
		appLogger.Info("成功创建用户名邮箱联合唯一索引")
	}

	return nil
}

// initDefaultData 初始化默认数据
func initDefaultData(db *gorm.DB) error {
	// 检查是否已有管理员用户
	var count int64
	db.Model(&rbacmodel.SysUser{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		return nil // 已存在管理员，无需初始化
	}

	appLogger.Info("开始初始化默认数据...")

	// 创建默认部门
	dept := &rbacmodel.SysDepartment{
		Name:     "总公司",
		Code:     "HQ",
		ParentID: 0,
		Sort:     0,
		Status:   1,
	}
	if err := db.Create(dept).Error; err != nil {
		return fmt.Errorf("创建默认部门失败: %w", err)
	}

	// 创建管理员角色
	adminRole := &rbacmodel.SysRole{
		Name:        "超级管理员",
		Code:        "admin",
		Description: "系统超级管理员，拥有所有权限",
		Sort:        0,
		Status:      1,
	}
	if err := db.Create(adminRole).Error; err != nil {
		return fmt.Errorf("创建管理员角色失败: %w", err)
	}

	// 创建管理员用户（密码：123456）
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("加密密码失败: %w", err)
	}

	adminUser := &rbacmodel.SysUser{
		Username:     "admin",
		Password:     string(hashedPassword),
		RealName:     "系统管理员",
		Email:        "admin@opshub.com",
		Status:       1,
		DepartmentID: dept.ID,
	}
	if err := db.Create(adminUser).Error; err != nil {
		return fmt.Errorf("创建管理员用户失败: %w", err)
	}

	// 为管理员分配角色
	if err := db.Exec("INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)",
		adminUser.ID, adminRole.ID).Error; err != nil {
		return fmt.Errorf("分配管理员角色失败: %w", err)
	}

	appLogger.Info("默认数据初始化完成")
	appLogger.Info("默认管理员账号: admin")
	appLogger.Info("默认管理员密码: 123456")

	// 初始化系统默认配置
	configRepo := systemdata.NewConfigRepo(db)
	if err := configRepo.InitDefaultConfigs(context.Background()); err != nil {
		appLogger.Warn("初始化系统配置失败", zap.Error(err))
	} else {
		appLogger.Info("系统默认配置初始化完成")
	}

	// 初始化默认服务标签
	initDefaultServiceLabels(db)

	// 初始化服务标签菜单
	initServiceLabelMenus(db)

	return nil
}

// initDefaultServiceLabels 初始化默认服务标签
func initDefaultServiceLabels(db *gorm.DB) {
	var count int64
	db.Model(&assetbiz.ServiceLabel{}).Count(&count)
	if count > 0 {
		return
	}

	labels := []assetbiz.ServiceLabel{
		{Name: "mysqld", MatchProcesses: "mysqld", Description: "MySQL数据库", Status: 1},
		{Name: "redis-server", MatchProcesses: "redis-server", Description: "Redis缓存", Status: 1},
		{Name: "mongodb", MatchProcesses: "mongod,mongos", Description: "MongoDB数据库", Status: 1},
		{Name: "milvus", MatchProcesses: "milvus", Description: "Milvus向量数据库", Status: 1},
		{Name: "kafka", MatchProcesses: "kafka", Description: "Kafka消息队列", Status: 1},
		{Name: "zookeeper", MatchProcesses: "zookeeper,QuorumPeerMain", Description: "ZooKeeper", Status: 1},
		{Name: "nfs", MatchProcesses: "nfsd,rpc.nfsd", Description: "NFS文件服务", Status: 1},
		{Name: "harbor", MatchProcesses: "harbor-core,harbor-jobservice", Description: "Harbor镜像仓库", Status: 1},
		{Name: "docker", MatchProcesses: "dockerd,containerd", Description: "Docker容器引擎", Status: 1},
		{Name: "k8s-master", MatchProcesses: "kube-apiserver,kube-controller-manager,kube-scheduler", Description: "Kubernetes Master节点", Status: 1},
		{Name: "k8s-node", MatchProcesses: "kubelet,kube-proxy", Description: "Kubernetes Worker节点", Status: 1},
		{Name: "clickhouse-server", MatchProcesses: "clickhouse-server,clickhouse", Description: "ClickHouse数据库", Status: 1},
		{Name: "doris", MatchProcesses: "doris_be,doris_fe,PaloFe,PaloBe", Description: "Apache Doris", Status: 1},
	}

	for _, label := range labels {
		db.Create(&label)
	}
	appLogger.Info("默认服务标签初始化完成", zap.Int("count", len(labels)))
}

// initServiceLabelMenus 初始化服务标签菜单（幂等）
func initServiceLabelMenus(db *gorm.DB) {
	// 检查菜单是否已存在
	var count int64
	db.Model(&rbacmodel.SysMenu{}).Where("code = ?", "service-label-management").Count(&count)
	if count > 0 {
		return
	}

	// 查找资产管理菜单ID
	var assetMenu rbacmodel.SysMenu
	if err := db.Where("code = ?", "asset-management").First(&assetMenu).Error; err != nil {
		appLogger.Warn("未找到资产管理菜单，跳过服务标签菜单初始化")
		return
	}

	// 创建服务标签页面菜单
	slMenu := &rbacmodel.SysMenu{
		Name:      "服务标签",
		Code:      "service-label-management",
		Type:      2,
		ParentID:  assetMenu.ID,
		Path:      "/asset/service-labels",
		Component: "asset/ServiceLabels",
		Icon:      "PriceTag",
		Sort:      9,
		Visible:   1,
		Status:    1,
	}
	if err := db.Create(slMenu).Error; err != nil {
		appLogger.Warn("创建服务标签菜单失败", zap.Error(err))
		return
	}

	// 创建按钮权限
	buttons := []rbacmodel.SysMenu{
		{Name: "新增标签", Code: "service-labels:create", Type: 3, ParentID: slMenu.ID, Sort: 1, Visible: 1, Status: 1, ApiPath: "/api/v1/service-labels", ApiMethod: "POST"},
		{Name: "编辑标签", Code: "service-labels:update", Type: 3, ParentID: slMenu.ID, Sort: 2, Visible: 1, Status: 1, ApiPath: "/api/v1/service-labels/:id", ApiMethod: "PUT"},
		{Name: "删除标签", Code: "service-labels:delete", Type: 3, ParentID: slMenu.ID, Sort: 3, Visible: 1, Status: 1, ApiPath: "/api/v1/service-labels/:id", ApiMethod: "DELETE"},
	}
	for i := range buttons {
		db.Create(&buttons[i])
	}

	// 为管理员角色分配菜单权限
	var adminRole rbacmodel.SysRole
	if err := db.Where("code = ?", "admin").First(&adminRole).Error; err == nil {
		db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, slMenu.ID)
		for _, btn := range buttons {
			db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, btn.ID)
		}
	}

	// 创建菜单API关联
	for _, btn := range buttons {
		if btn.ApiPath != "" {
			db.Create(&rbacmodel.SysMenuAPI{
				MenuID:    btn.ID,
				ApiPath:   btn.ApiPath,
				ApiMethod: btn.ApiMethod,
			})
		}
	}

	appLogger.Info("服务标签菜单初始化完成")
}

func stopServer(ctx context.Context, cfg *conf.Config) error {
	appLogger.Info("服务正在关闭...")

	// 停止gRPC服务器
	if globalGRPCServer != nil {
		globalGRPCServer.Stop()
	}

	// 停止HTTP服务器
	if globalHTTPServer != nil {
		if err := globalHTTPServer.Stop(ctx); err != nil {
			appLogger.Error("停止HTTP服务器失败", zap.Error(err))
		}
	}

	// 关闭数据库连接
	if globalData != nil {
		if err := globalData.Close(); err != nil {
			appLogger.Error("关闭数据库连接失败", zap.Error(err))
		}
	}

	// 关闭Redis连接
	if globalRedis != nil {
		if err := globalRedis.Close(); err != nil {
			appLogger.Error("关闭Redis连接失败", zap.Error(err))
		}
	}

	return nil
}

func printStartupInfo(cfg *conf.Config) {
	addr := fmt.Sprintf("%s:%d", "0.0.0.0", cfg.Server.HttpPort)

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("       OpsHub 运维管理平台启动成功")
	fmt.Println("========================================")
	fmt.Printf("版本:     1.0.0\n")
	fmt.Printf("模式:     %s\n", cfg.Server.Mode)
	fmt.Printf("监听地址: http://%s\n", addr)
	fmt.Printf("健康检查: http://%s/health\n", addr)
	fmt.Printf("API文档:  http://%s/swagger/index.html\n", addr)
	fmt.Println("========================================")
	fmt.Println()
}
