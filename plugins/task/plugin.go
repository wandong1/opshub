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

package task

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ydcloud-dy/opshub/internal/plugin"
	"github.com/ydcloud-dy/opshub/pkg/utils"
	"github.com/ydcloud-dy/opshub/plugins/task/model"
	"github.com/ydcloud-dy/opshub/plugins/task/server"
)

// Plugin 任务中心插件实现
type Plugin struct {
	db   *gorm.DB
	name string
}

// New 创建插件实例
func New() *Plugin {
	return &Plugin{
		name: "task",
	}
}

// Name 返回插件名称
func (p *Plugin) Name() string {
	return "task"
}

// Description 返回插件描述
func (p *Plugin) Description() string {
	return "任务中心插件 - 支持执行任务、模板管理和文件分发"
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
		&model.JobTask{},
		&model.JobTemplate{},
		&model.AnsibleTask{},
	}

	for _, m := range models {
		if !db.Migrator().HasTable(m) {
			if err := db.AutoMigrate(m); err != nil {
				return err
			}
		}
	}

	// 初始化插件按钮权限
	p.initPermissions(db)

	return nil
}

// initPermissions 初始化Task插件按钮权限
func (p *Plugin) initPermissions(db *gorm.DB) {
	utils.EnsureMenu(db, "task", "任务中心", 1, "", "/task", "", "Files", 25)
	utils.EnsureMenu(db, "task-execute", "执行任务", 2, "task", "/task/execute", "task/Execute", "Promotion", 1)
	utils.EnsureMenu(db, "task-templates", "模板管理", 2, "task", "/task/templates", "task/Templates", "Document", 2)
	utils.EnsureMenu(db, "task-history", "执行历史", 2, "task", "/task/history", "task/History", "Timer", 3)

	utils.EnsureMenuPermissions(db, "task-execute", []utils.MenuPermission{
		{Code: "task:execute", Name: "执行任务", ApiMethod: "POST", ApiPath: "/api/v1/plugins/task/execute", Sort: 1},
		{Code: "task:distribute", Name: "文件分发", ApiMethod: "POST", ApiPath: "/api/v1/plugins/task/distribute", Sort: 2},
	})

	utils.EnsureMenuPermissions(db, "task-templates", []utils.MenuPermission{
		{Code: "templates:list", Name: "查看模板列表", ApiMethod: "GET", ApiPath: "/api/v1/plugins/task/templates", Sort: 1},
		{Code: "templates:create", Name: "创建模板", ApiMethod: "POST", ApiPath: "/api/v1/plugins/task/templates", Sort: 2},
		{Code: "templates:update", Name: "编辑模板", ApiMethod: "PUT", ApiPath: "/api/v1/plugins/task/templates/:id", Sort: 3},
		{Code: "templates:delete", Name: "删除模板", ApiMethod: "DELETE", ApiPath: "/api/v1/plugins/task/templates/:id", Sort: 4},
	})

	utils.EnsureMenuPermissions(db, "task", []utils.MenuPermission{
		{Code: "jobs:list", Name: "查看任务列表", ApiMethod: "GET", ApiPath: "/api/v1/plugins/task/jobs", Sort: 5},
		{Code: "jobs:create", Name: "创建任务", ApiMethod: "POST", ApiPath: "/api/v1/plugins/task/jobs", Sort: 6},
		{Code: "jobs:update", Name: "编辑任务", ApiMethod: "PUT", ApiPath: "/api/v1/plugins/task/jobs/:id", Sort: 7},
		{Code: "jobs:delete", Name: "删除任务", ApiMethod: "DELETE", ApiPath: "/api/v1/plugins/task/jobs/:id", Sort: 8},
	})

	utils.EnsureMenuPermissions(db, "task-history", []utils.MenuPermission{
		{Code: "exec-history:list", Name: "查看执行历史", ApiMethod: "GET", ApiPath: "/api/v1/plugins/task/execution-history", Sort: 1},
		{Code: "exec-history:delete", Name: "删除执行历史", ApiMethod: "DELETE", ApiPath: "/api/v1/plugins/task/execution-history/:id", Sort: 2},
		{Code: "exec-history:batch-delete", Name: "批量删除历史", ApiMethod: "POST", ApiPath: "/api/v1/plugins/task/execution-history/batch-delete", Sort: 3},
	})

	utils.AssignMenusToAdminRole(db)
}

// Disable 禁用插件
func (p *Plugin) Disable(db *gorm.DB) error {
	// 清理插件资源（如果需要）
	return nil
}

// RegisterRoutes 注册路由
func (p *Plugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	server.RegisterRoutes(router, db)
}

// GetMenus 获取插件菜单配置
func (p *Plugin) GetMenus() []plugin.MenuConfig {
	return []plugin.MenuConfig{
		{
			Name: "执行任务",
			Path: "/task/execute",
			Icon: "VideoPlay",
			Sort: 50,
		},
		{
			Name: "模板管理",
			Path: "/task/templates",
			Icon: "Document",
			Sort: 51,
		},
		{
			Name: "文件分发",
			Path: "/task/file-distribution",
			Icon: "FolderOpened",
			Sort: 52,
		},
	}
}
