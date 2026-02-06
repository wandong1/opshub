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

package nginx

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ydcloud-dy/opshub/internal/plugin"
	"github.com/ydcloud-dy/opshub/pkg/utils"
	"github.com/ydcloud-dy/opshub/plugins/nginx/model"
	"github.com/ydcloud-dy/opshub/plugins/nginx/server"
)

// Plugin Nginx统计插件实现
type Plugin struct {
	db        *gorm.DB
	name      string
	ctx       context.Context
	cancelCtx context.CancelFunc
}

// New 创建插件实例
func New() *Plugin {
	return &Plugin{
		name: "nginx",
	}
}

// Name 返回插件名称
func (p *Plugin) Name() string {
	return "nginx"
}

// Description 返回插件描述
func (p *Plugin) Description() string {
	return "Nginx统计插件 - 支持主机Nginx和K8s Ingress-Nginx的访问日志分析和统计"
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
		// 数据源
		&model.NginxSource{},

		// 旧版表（兼容）
		&model.NginxAccessLog{},
		&model.NginxDailyStats{},
		&model.NginxHourlyStats{},

		// 维度表
		&model.NginxDimIP{},
		&model.NginxDimURL{},
		&model.NginxDimReferer{},
		&model.NginxDimUserAgent{},

		// 事实表
		&model.NginxFactAccessLog{},

		// 新版聚合表
		&model.NginxAggHourly{},
		&model.NginxAggDaily{},
	}

	for _, m := range models {
		if err := db.AutoMigrate(m); err != nil {
			return err
		}
	}

	// 启动上下文
	p.ctx, p.cancelCtx = context.WithCancel(context.Background())

	// 启动日志采集调度器
	go p.startLogCollectorScheduler()

	// 初始化插件按钮权限
	p.initPermissions(db)

	return nil
}

// initPermissions 初始化Nginx插件按钮权限
func (p *Plugin) initPermissions(db *gorm.DB) {
	utils.EnsureMenu(db, "nginx", "Nginx统计", 1, "", "/nginx", "", "Odometer", 36)
	utils.EnsureMenu(db, "nginx-config", "功能配置", 2, "nginx", "/nginx/config", "nginx/Config", "Setting", 1)

	utils.EnsureMenuPermissions(db, "nginx-config", []utils.MenuPermission{
		{Code: "nginx-sources:list", Name: "查看数据源", ApiMethod: "GET", ApiPath: "/api/v1/plugins/nginx/sources", Sort: 1},
		{Code: "nginx-sources:create", Name: "创建数据源", ApiMethod: "POST", ApiPath: "/api/v1/plugins/nginx/sources", Sort: 2},
		{Code: "nginx-sources:update", Name: "编辑数据源", ApiMethod: "PUT", ApiPath: "/api/v1/plugins/nginx/sources/:id", Sort: 3},
		{Code: "nginx-sources:delete", Name: "删除数据源", ApiMethod: "DELETE", ApiPath: "/api/v1/plugins/nginx/sources/:id", Sort: 4},
	})

	utils.EnsureMenuPermissions(db, "nginx", []utils.MenuPermission{
		{Code: "nginx:overview", Name: "查看概况", ApiMethod: "GET", ApiPath: "/api/v1/plugins/nginx/overview", Sort: 5},
		{Code: "nginx:logs", Name: "查看访问日志", ApiMethod: "GET", ApiPath: "/api/v1/plugins/nginx/access-logs", Sort: 6},
		{Code: "nginx:stats", Name: "查看统计数据", ApiMethod: "GET", ApiPath: "/api/v1/plugins/nginx/stats/timeseries", Sort: 7},
		{Code: "nginx:collect", Name: "手动采集日志", ApiMethod: "POST", ApiPath: "/api/v1/plugins/nginx/collect", Sort: 8},
	})

	utils.AssignMenusToAdminRole(db)
}

// Disable 禁用插件
func (p *Plugin) Disable(db *gorm.DB) error {
	// 停止定时任务
	if p.cancelCtx != nil {
		p.cancelCtx()
	}
	return nil
}

// startLogCollectorScheduler 启动日志采集调度器
func (p *Plugin) startLogCollectorScheduler() {
	// 每小时执行一次
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	handler := server.NewHandler(p.db)

	// 首次启动立即执行一次（可选）
	// p.collectActiveSources(handler)

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.collectActiveSources(handler)
		}
	}
}

// collectActiveSources 采集所有活跃数据源的日志
func (p *Plugin) collectActiveSources(handler *server.Handler) {
	// 获取所有活跃的数据源
	var sources []model.NginxSource
	if err := p.db.Where("status = ?", 1).Find(&sources).Error; err != nil {
		fmt.Printf("获取活跃数据源失败: %v\n", err)
		return
	}

	if len(sources) == 0 {
		return
	}

	fmt.Printf("[Nginx插件] 开始定时采集日志，共 %d 个数据源\n", len(sources))

	// 并发采集各个数据源的日志
	for _, source := range sources {
		// 在后台执行采集，避免阻塞
		go func(s model.NginxSource) {
			if err := handler.CollectSourceLogs(&s); err != nil {
				fmt.Printf("[Nginx插件] 采集数据源 %s (ID:%d) 失败: %v\n", s.Name, s.ID, err)
			} else {
				fmt.Printf("[Nginx插件] 采集数据源 %s (ID:%d) 成功\n", s.Name, s.ID)
			}
		}(source)
	}
}

// RegisterRoutes 注册路由
func (p *Plugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	server.RegisterRoutes(router, db)
}

// GetMenus 获取插件菜单配置
func (p *Plugin) GetMenus() []plugin.MenuConfig {
	parentPath := "/nginx"

	return []plugin.MenuConfig{
		{
			Name:       "Nginx统计",
			Path:       parentPath,
			Icon:       "DataLine",
			Sort:       50,
			Hidden:     false,
			ParentPath: "",
		},
		{
			Name:       "概况",
			Path:       "/nginx/overview",
			Icon:       "PieChart",
			Sort:       1,
			Hidden:     false,
			ParentPath: parentPath,
		},
		{
			Name:       "数据日报",
			Path:       "/nginx/daily-report",
			Icon:       "Calendar",
			Sort:       2,
			Hidden:     false,
			ParentPath: parentPath,
		},
		{
			Name:       "实时",
			Path:       "/nginx/realtime",
			Icon:       "Timer",
			Sort:       3,
			Hidden:     false,
			ParentPath: parentPath,
		},
		{
			Name:       "访问明细",
			Path:       "/nginx/access-logs",
			Icon:       "List",
			Sort:       4,
			Hidden:     false,
			ParentPath: parentPath,
		},
		{
			Name:       "功能配置",
			Path:       "/nginx/config",
			Icon:       "Setting",
			Sort:       5,
			Hidden:     false,
			ParentPath: parentPath,
		},
	}
}
