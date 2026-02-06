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
	auditmodel "github.com/ydcloud-dy/opshub/internal/biz/audit"
	rbacmodel "github.com/ydcloud-dy/opshub/internal/biz/rbac"
	systemmodel "github.com/ydcloud-dy/opshub/internal/biz/system"
	"github.com/ydcloud-dy/opshub/internal/conf"
	dataPkg "github.com/ydcloud-dy/opshub/internal/data"
	systemdata "github.com/ydcloud-dy/opshub/internal/data/system"
	"github.com/ydcloud-dy/opshub/internal/server"
	"github.com/ydcloud-dy/opshub/internal/service"
	rbacservice "github.com/ydcloud-dy/opshub/internal/service/rbac"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/utils"
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

	// 初始化按钮级权限数据（幂等，每次启动执行）
	initButtonPermissions(data.DB())

	// 初始化HTTP服务器
	httpServer := server.NewHTTPServer(cfg, svc, data.DB(), redis.Get())
	globalHTTPServer = httpServer // 保存到全局变量

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

	// 创建默认菜单 - 先创建顶级菜单
	dashboardMenu := &rbacmodel.SysMenu{Name: "仪表盘", Code: "dashboard", Type: 2, ParentID: 0, Path: "/dashboard", Component: "Dashboard", Icon: "HomeFilled", Sort: 0, Visible: 1, Status: 1}
	if err := db.Create(dashboardMenu).Error; err != nil {
		return fmt.Errorf("创建首页菜单失败: %w", err)
	}
	db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, dashboardMenu.ID)

	// 创建系统管理顶级菜单
	systemMenu := &rbacmodel.SysMenu{Name: "系统管理", Code: "system", Type: 1, ParentID: 0, Path: "/system", Icon: "Setting", Sort: 100, Visible: 1, Status: 1}
	if err := db.Create(systemMenu).Error; err != nil {
		return fmt.Errorf("创建系统管理菜单失败: %w", err)
	}
	db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, systemMenu.ID)

	// 创建系统管理子菜单（使用 systemMenu.ID 作为父级）
	systemSubMenus := []*rbacmodel.SysMenu{
		{Name: "用户管理", Code: "users", Type: 2, ParentID: systemMenu.ID, Path: "/system/users", Component: "system/Users", Icon: "User", Sort: 1, Visible: 1, Status: 1},
		{Name: "角色管理", Code: "roles", Type: 2, ParentID: systemMenu.ID, Path: "/system/roles", Component: "system/Roles", Icon: "UserFilled", Sort: 2, Visible: 1, Status: 1},
		{Name: "菜单管理", Code: "menus", Type: 2, ParentID: systemMenu.ID, Path: "/system/menus", Component: "system/Menus", Icon: "Menu", Sort: 3, Visible: 1, Status: 1},
		{Name: "部门信息", Code: "departments", Type: 2, ParentID: systemMenu.ID, Path: "/system/departments", Component: "system/Departments", Icon: "OfficeBuilding", Sort: 4, Visible: 1, Status: 1},
		{Name: "岗位信息", Code: "positions", Type: 2, ParentID: systemMenu.ID, Path: "/system/positions", Component: "system/Positions", Icon: "Avatar", Sort: 5, Visible: 1, Status: 1},
		{Name: "系统配置", Code: "settings", Type: 2, ParentID: systemMenu.ID, Path: "/system/settings", Component: "system/Settings", Icon: "Tools", Sort: 6, Visible: 1, Status: 1},
	}

	for _, menu := range systemSubMenus {
		if err := db.Create(menu).Error; err != nil {
			return fmt.Errorf("创建系统管理子菜单失败: %w", err)
		}
		db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, menu.ID)
	}

	// 创建操作审计顶级菜单
	auditMenu := &rbacmodel.SysMenu{Name: "操作审计", Code: "audit", Type: 1, ParentID: 0, Path: "/audit", Icon: "Document", Sort: 50, Visible: 1, Status: 1}
	if err := db.Create(auditMenu).Error; err != nil {
		return fmt.Errorf("创建操作审计菜单失败: %w", err)
	}
	db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, auditMenu.ID)

	// 创建操作审计子菜单（不包含数据日志）
	auditSubMenus := []*rbacmodel.SysMenu{
		{Name: "操作日志", Code: "operation-logs", Type: 2, ParentID: auditMenu.ID, Path: "/audit/operation-logs", Component: "audit/OperationLogs", Icon: "Document", Sort: 1, Visible: 1, Status: 1},
		{Name: "登录日志", Code: "login-logs", Type: 2, ParentID: auditMenu.ID, Path: "/audit/login-logs", Component: "audit/LoginLogs", Icon: "CircleCheck", Sort: 2, Visible: 1, Status: 1},
	}

	for _, menu := range auditSubMenus {
		if err := db.Create(menu).Error; err != nil {
			return fmt.Errorf("创建操作审计子菜单失败: %w", err)
		}
		db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, menu.ID)
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

	return nil
}

// initButtonPermissions 初始化按钮级权限数据（幂等）
func initButtonPermissions(db *gorm.DB) {
	appLogger.Info("检查并初始化按钮级权限数据...")

	// 用户管理
	utils.EnsureMenuPermissions(db, "users", []utils.MenuPermission{
		{Code: "users:list", Name: "查看用户列表", ApiMethod: "GET", ApiPath: "/api/v1/users", Sort: 1},
		{Code: "users:detail", Name: "查看用户详情", ApiMethod: "GET", ApiPath: "/api/v1/users/:id", Sort: 2},
		{Code: "users:create", Name: "创建用户", ApiMethod: "POST", ApiPath: "/api/v1/users", Sort: 3},
		{Code: "users:update", Name: "编辑用户", ApiMethod: "PUT", ApiPath: "/api/v1/users/:id", Sort: 4},
		{Code: "users:delete", Name: "删除用户", ApiMethod: "DELETE", ApiPath: "/api/v1/users/:id", Sort: 5},
		{Code: "users:assign-roles", Name: "分配角色", ApiMethod: "POST", ApiPath: "/api/v1/users/:id/roles", Sort: 6},
		{Code: "users:assign-positions", Name: "分配岗位", ApiMethod: "POST", ApiPath: "/api/v1/users/:id/positions", Sort: 7},
		{Code: "users:reset-pwd", Name: "重置密码", ApiMethod: "PUT", ApiPath: "/api/v1/users/:id/reset-password", Sort: 8},
		{Code: "users:unlock", Name: "解锁用户", ApiMethod: "POST", ApiPath: "/api/v1/users/:id/unlock", Sort: 9},
	})

	// 角色管理
	utils.EnsureMenuPermissions(db, "roles", []utils.MenuPermission{
		{Code: "roles:list", Name: "查看角色列表", ApiMethod: "GET", ApiPath: "/api/v1/roles", Sort: 1},
		{Code: "roles:all", Name: "获取全部角色", ApiMethod: "GET", ApiPath: "/api/v1/roles/all", Sort: 2},
		{Code: "roles:detail", Name: "查看角色详情", ApiMethod: "GET", ApiPath: "/api/v1/roles/:id", Sort: 3},
		{Code: "roles:create", Name: "创建角色", ApiMethod: "POST", ApiPath: "/api/v1/roles", Sort: 4},
		{Code: "roles:update", Name: "编辑角色", ApiMethod: "PUT", ApiPath: "/api/v1/roles/:id", Sort: 5},
		{Code: "roles:delete", Name: "删除角色", ApiMethod: "DELETE", ApiPath: "/api/v1/roles/:id", Sort: 6},
		{Code: "roles:assign-menus", Name: "分配菜单权限", ApiMethod: "POST", ApiPath: "/api/v1/roles/:id/menus", Sort: 7},
	})

	// 菜单管理
	utils.EnsureMenuPermissions(db, "menus", []utils.MenuPermission{
		{Code: "menus:tree", Name: "查看菜单树", ApiMethod: "GET", ApiPath: "/api/v1/menus/tree", Sort: 1},
		{Code: "menus:detail", Name: "查看菜单详情", ApiMethod: "GET", ApiPath: "/api/v1/menus/:id", Sort: 2},
		{Code: "menus:create", Name: "创建菜单", ApiMethod: "POST", ApiPath: "/api/v1/menus", Sort: 3},
		{Code: "menus:update", Name: "编辑菜单", ApiMethod: "PUT", ApiPath: "/api/v1/menus/:id", Sort: 4},
		{Code: "menus:delete", Name: "删除菜单", ApiMethod: "DELETE", ApiPath: "/api/v1/menus/:id", Sort: 5},
	})

	// 部门管理
	utils.EnsureMenuPermissions(db, "departments", []utils.MenuPermission{
		{Code: "depts:tree", Name: "查看部门树", ApiMethod: "GET", ApiPath: "/api/v1/departments/tree", Sort: 1},
		{Code: "depts:parent-options", Name: "获取上级部门选项", ApiMethod: "GET", ApiPath: "/api/v1/departments/parent-options", Sort: 2},
		{Code: "depts:detail", Name: "查看部门详情", ApiMethod: "GET", ApiPath: "/api/v1/departments/:id", Sort: 3},
		{Code: "depts:create", Name: "创建部门", ApiMethod: "POST", ApiPath: "/api/v1/departments", Sort: 4},
		{Code: "depts:update", Name: "编辑部门", ApiMethod: "PUT", ApiPath: "/api/v1/departments/:id", Sort: 5},
		{Code: "depts:delete", Name: "删除部门", ApiMethod: "DELETE", ApiPath: "/api/v1/departments/:id", Sort: 6},
	})

	// 岗位管理
	utils.EnsureMenuPermissions(db, "positions", []utils.MenuPermission{
		{Code: "positions:list", Name: "查看岗位列表", ApiMethod: "GET", ApiPath: "/api/v1/positions", Sort: 1},
		{Code: "positions:detail", Name: "查看岗位详情", ApiMethod: "GET", ApiPath: "/api/v1/positions/:id", Sort: 2},
		{Code: "positions:create", Name: "创建岗位", ApiMethod: "POST", ApiPath: "/api/v1/positions", Sort: 3},
		{Code: "positions:update", Name: "编辑岗位", ApiMethod: "PUT", ApiPath: "/api/v1/positions/:id", Sort: 4},
		{Code: "positions:delete", Name: "删除岗位", ApiMethod: "DELETE", ApiPath: "/api/v1/positions/:id", Sort: 5},
		{Code: "positions:users", Name: "查看岗位用户", ApiMethod: "GET", ApiPath: "/api/v1/positions/:id/users", Sort: 6},
		{Code: "positions:assign-users", Name: "分配岗位用户", ApiMethod: "POST", ApiPath: "/api/v1/positions/:id/users", Sort: 7},
		{Code: "positions:remove-user", Name: "移除岗位用户", ApiMethod: "DELETE", ApiPath: "/api/v1/positions/:id/users/:userId", Sort: 8},
	})

	// 系统配置
	utils.EnsureMenuPermissions(db, "settings", []utils.MenuPermission{
		{Code: "settings:view", Name: "查看系统配置", ApiMethod: "GET", ApiPath: "/api/v1/system/config", Sort: 1},
		{Code: "settings:basic", Name: "保存基础配置", ApiMethod: "PUT", ApiPath: "/api/v1/system/config/basic", Sort: 2},
		{Code: "settings:security", Name: "保存安全配置", ApiMethod: "PUT", ApiPath: "/api/v1/system/config/security", Sort: 3},
		{Code: "settings:logo", Name: "上传系统Logo", ApiMethod: "POST", ApiPath: "/api/v1/system/config/logo", Sort: 4},
	})

	// 操作日志
	utils.EnsureMenuPermissions(db, "operation-logs", []utils.MenuPermission{
		{Code: "op-logs:list", Name: "查看操作日志", ApiMethod: "GET", ApiPath: "/api/v1/audit/operation-logs", Sort: 1},
		{Code: "op-logs:detail", Name: "查看日志详情", ApiMethod: "GET", ApiPath: "/api/v1/audit/operation-logs/:id", Sort: 2},
		{Code: "op-logs:delete", Name: "删除操作日志", ApiMethod: "DELETE", ApiPath: "/api/v1/audit/operation-logs/:id", Sort: 3},
		{Code: "op-logs:batch-delete", Name: "批量删除操作日志", ApiMethod: "POST", ApiPath: "/api/v1/audit/operation-logs/batch-delete", Sort: 4},
	})

	// 登录日志
	utils.EnsureMenuPermissions(db, "login-logs", []utils.MenuPermission{
		{Code: "login-logs:list", Name: "查看登录日志", ApiMethod: "GET", ApiPath: "/api/v1/audit/login-logs", Sort: 1},
		{Code: "login-logs:detail", Name: "查看日志详情", ApiMethod: "GET", ApiPath: "/api/v1/audit/login-logs/:id", Sort: 2},
		{Code: "login-logs:delete", Name: "删除登录日志", ApiMethod: "DELETE", ApiPath: "/api/v1/audit/login-logs/:id", Sort: 3},
		{Code: "login-logs:batch-delete", Name: "批量删除登录日志", ApiMethod: "POST", ApiPath: "/api/v1/audit/login-logs/batch-delete", Sort: 4},
	})

	// === 资产管理模块（确保父菜单存在） ===
	utils.EnsureMenu(db, "assets", "资产管理", 1, "", "/assets", "", "FolderOpened", 30)
	utils.EnsureMenu(db, "asset-groups", "资产分组", 2, "assets", "/assets/groups", "asset/Groups", "Connection", 1)
	utils.EnsureMenu(db, "hosts", "主机管理", 2, "assets", "/assets/hosts", "asset/Hosts", "Monitor", 2)
	utils.EnsureMenu(db, "credentials", "凭证管理", 2, "assets", "/assets/credentials", "asset/Credentials", "Lock", 3)
	utils.EnsureMenu(db, "terminal-sessions", "终端审计", 2, "assets", "/assets/terminal-sessions", "asset/TerminalSessions", "View", 4)

	// 资产分组
	utils.EnsureMenuPermissions(db, "asset-groups", []utils.MenuPermission{
		{Code: "asset-groups:tree", Name: "查看资产分组", ApiMethod: "GET", ApiPath: "/api/v1/asset-groups/tree", Sort: 1},
		{Code: "asset-groups:create", Name: "创建资产分组", ApiMethod: "POST", ApiPath: "/api/v1/asset-groups", Sort: 2},
		{Code: "asset-groups:update", Name: "编辑资产分组", ApiMethod: "PUT", ApiPath: "/api/v1/asset-groups/:id", Sort: 3},
		{Code: "asset-groups:delete", Name: "删除资产分组", ApiMethod: "DELETE", ApiPath: "/api/v1/asset-groups/:id", Sort: 4},
	})

	// 主机管理
	utils.EnsureMenuPermissions(db, "hosts", []utils.MenuPermission{
		{Code: "hosts:list", Name: "查看主机列表", ApiMethod: "GET", ApiPath: "/api/v1/hosts", Sort: 1},
		{Code: "hosts:detail", Name: "查看主机详情", ApiMethod: "GET", ApiPath: "/api/v1/hosts/:id", Sort: 2},
		{Code: "hosts:create", Name: "创建主机", ApiMethod: "POST", ApiPath: "/api/v1/hosts", Sort: 3},
		{Code: "hosts:update", Name: "编辑主机", ApiMethod: "PUT", ApiPath: "/api/v1/hosts/:id", Sort: 4},
		{Code: "hosts:delete", Name: "删除主机", ApiMethod: "DELETE", ApiPath: "/api/v1/hosts/:id", Sort: 5},
		{Code: "hosts:batch-delete", Name: "批量删除主机", ApiMethod: "POST", ApiPath: "/api/v1/hosts/batch-delete", Sort: 6},
		{Code: "hosts:test", Name: "测试主机连接", ApiMethod: "POST", ApiPath: "/api/v1/hosts/:id/test", Sort: 7},
		{Code: "hosts:collect", Name: "采集主机信息", ApiMethod: "POST", ApiPath: "/api/v1/hosts/:id/collect", Sort: 8},
		{Code: "hosts:batch-collect", Name: "批量采集信息", ApiMethod: "POST", ApiPath: "/api/v1/hosts/batch-collect", Sort: 9},
		{Code: "hosts:import", Name: "导入主机", ApiMethod: "POST", ApiPath: "/api/v1/hosts/import", Sort: 10},
		{Code: "hosts:template", Name: "下载导入模板", ApiMethod: "GET", ApiPath: "/api/v1/hosts/template/download", Sort: 11},
		{Code: "hosts:files", Name: "文件管理", ApiMethod: "GET", ApiPath: "/api/v1/hosts/:id/files", Sort: 12},
		{Code: "hosts:file-upload", Name: "上传文件", ApiMethod: "POST", ApiPath: "/api/v1/hosts/:id/files/upload", Sort: 13},
		{Code: "hosts:file-download", Name: "下载文件", ApiMethod: "GET", ApiPath: "/api/v1/hosts/:id/files/download", Sort: 14},
		{Code: "hosts:file-delete", Name: "删除文件", ApiMethod: "DELETE", ApiPath: "/api/v1/hosts/:id/files", Sort: 15},
	})

	// 凭证管理
	utils.EnsureMenuPermissions(db, "credentials", []utils.MenuPermission{
		{Code: "credentials:list", Name: "查看凭证列表", ApiMethod: "GET", ApiPath: "/api/v1/credentials", Sort: 1},
		{Code: "credentials:create", Name: "创建凭证", ApiMethod: "POST", ApiPath: "/api/v1/credentials", Sort: 2},
		{Code: "credentials:update", Name: "编辑凭证", ApiMethod: "PUT", ApiPath: "/api/v1/credentials/:id", Sort: 3},
		{Code: "credentials:delete", Name: "删除凭证", ApiMethod: "DELETE", ApiPath: "/api/v1/credentials/:id", Sort: 4},
	})

	// 终端
	utils.EnsureMenuPermissions(db, "terminal-sessions", []utils.MenuPermission{
		{Code: "terminal:connect", Name: "SSH终端连接", ApiMethod: "GET", ApiPath: "/api/v1/asset/terminal/:id", Sort: 1},
		{Code: "terminal-sessions:list", Name: "查看终端会话", ApiMethod: "GET", ApiPath: "/api/v1/terminal-sessions", Sort: 2},
		{Code: "terminal-sessions:play", Name: "回放终端会话", ApiMethod: "GET", ApiPath: "/api/v1/terminal-sessions/:id/play", Sort: 3},
		{Code: "terminal-sessions:delete", Name: "删除终端会话", ApiMethod: "DELETE", ApiPath: "/api/v1/terminal-sessions/:id", Sort: 4},
	})

	// === 身份认证模块 ===
	utils.EnsureMenu(db, "identity", "身份认证", 1, "", "/identity", "", "Lock", 40)
	utils.EnsureMenu(db, "identity-sources", "身份源管理", 2, "identity", "/identity/sources", "identity/Sources", "Connection", 1)
	utils.EnsureMenu(db, "identity-apps", "应用管理", 2, "identity", "/identity/apps", "identity/Apps", "Platform", 2)
	utils.EnsureMenu(db, "identity-logs", "认证日志", 2, "identity", "/identity/logs", "identity/Logs", "Document", 3)

	// 身份源
	utils.EnsureMenuPermissions(db, "identity-sources", []utils.MenuPermission{
		{Code: "identity-sources:list", Name: "查看身份源", ApiMethod: "GET", ApiPath: "/api/v1/identity/sources", Sort: 1},
		{Code: "identity-sources:create", Name: "创建身份源", ApiMethod: "POST", ApiPath: "/api/v1/identity/sources", Sort: 2},
		{Code: "identity-sources:update", Name: "编辑身份源", ApiMethod: "PUT", ApiPath: "/api/v1/identity/sources/:id", Sort: 3},
		{Code: "identity-sources:delete", Name: "删除身份源", ApiMethod: "DELETE", ApiPath: "/api/v1/identity/sources/:id", Sort: 4},
	})

	// 应用管理
	utils.EnsureMenuPermissions(db, "identity-apps", []utils.MenuPermission{
		{Code: "identity-apps:list", Name: "查看应用列表", ApiMethod: "GET", ApiPath: "/api/v1/identity/apps", Sort: 1},
		{Code: "identity-apps:create", Name: "创建应用", ApiMethod: "POST", ApiPath: "/api/v1/identity/apps", Sort: 2},
		{Code: "identity-apps:update", Name: "编辑应用", ApiMethod: "PUT", ApiPath: "/api/v1/identity/apps/:id", Sort: 3},
		{Code: "identity-apps:delete", Name: "删除应用", ApiMethod: "DELETE", ApiPath: "/api/v1/identity/apps/:id", Sort: 4},
	})

	// 认证日志
	utils.EnsureMenuPermissions(db, "identity-logs", []utils.MenuPermission{
		{Code: "identity-logs:list", Name: "查看认证日志", ApiMethod: "GET", ApiPath: "/api/v1/identity/logs", Sort: 1},
		{Code: "identity-logs:stats", Name: "查看认证统计", ApiMethod: "GET", ApiPath: "/api/v1/identity/logs/stats", Sort: 2},
	})

	// === 资产权限管理 ===
	utils.EnsureMenu(db, "asset-perms", "资产权限", 2, "assets", "/assets/permissions", "asset/Permissions", "Key", 5)

	utils.EnsureMenuPermissions(db, "asset-perms", []utils.MenuPermission{
		{Code: "asset-perms:list", Name: "查看资产权限", ApiMethod: "GET", ApiPath: "/api/v1/asset-permissions", Sort: 1},
		{Code: "asset-perms:create", Name: "创建资产权限", ApiMethod: "POST", ApiPath: "/api/v1/asset-permissions", Sort: 2},
		{Code: "asset-perms:update", Name: "编辑资产权限", ApiMethod: "PUT", ApiPath: "/api/v1/asset-permissions/:id", Sort: 3},
		{Code: "asset-perms:delete", Name: "删除资产权限", ApiMethod: "DELETE", ApiPath: "/api/v1/asset-permissions/:id", Sort: 4},
	})

	// 将所有菜单分配给admin角色
	utils.AssignMenusToAdminRole(db)

	appLogger.Info("按钮级权限数据初始化完成")
}

func stopServer(ctx context.Context, cfg *conf.Config) error {
	appLogger.Info("服务正在关闭...")

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
