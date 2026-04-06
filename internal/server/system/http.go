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

package system

import (
	"github.com/gin-gonic/gin"
	systembiz "github.com/ydcloud-dy/opshub/internal/biz/system"
	systemdata "github.com/ydcloud-dy/opshub/internal/data/system"
	systemservice "github.com/ydcloud-dy/opshub/internal/service/system"
	"gorm.io/gorm"
)

// HTTPServer 系统配置HTTP服务器
type HTTPServer struct {
	configService  *systemservice.ConfigService
	grafanaProxy   *GrafanaProxyHandler
}

// NewHTTPServer 创建系统配置HTTP服务器
func NewHTTPServer(configService *systemservice.ConfigService) *HTTPServer {
	// 创建 Grafana 代理（复用 configUseCase）
	grafanaProxy := NewGrafanaProxyHandler(configService.GetConfigUseCase())
	return &HTTPServer{
		configService: configService,
		grafanaProxy:  grafanaProxy,
	}
}

// RegisterRoutes 注册路由
func (s *HTTPServer) RegisterRoutes(auth *gin.RouterGroup, public *gin.RouterGroup) {
	// 需要认证的路由
	system := auth.Group("/system")
	{
		config := system.Group("/config")
		{
			config.GET("", s.configService.GetAllConfig)
			config.GET("/basic", s.configService.GetBasicConfig)
			config.PUT("/basic", s.configService.SaveBasicConfig)
			config.GET("/security", s.configService.GetSecurityConfig)
			config.PUT("/security", s.configService.SaveSecurityConfig)
			config.GET("/data-retention", s.configService.GetDataRetentionConfig)
			config.PUT("/data-retention", s.configService.SaveDataRetentionConfig)
			config.POST("/logo", s.configService.UploadLogo)
		}

		// 集成管理路由
		integration := system.Group("/integrations")
		{
			integration.GET("", s.configService.GetIntegrationConfig)
			integration.PUT("", s.configService.SaveIntegrationConfig)
		}

		// 定制功能配置路由
		custom := system.Group("/custom")
		{
			custom.GET("", s.configService.GetCustomConfig)
			custom.PUT("", s.configService.SaveCustomConfig)
		}
	}

	// 公开路由（无需认证）
	public.GET("/config", s.configService.GetPublicConfig)
}

// RegisterGrafanaProxy 在根路由注册 Grafana 代理（路径与 Grafana sub_path 一致，无需认证）
func (s *HTTPServer) RegisterGrafanaProxy(router *gin.Engine) {
	s.grafanaProxy.RegisterProxyRoute(router)
}

// NewSystemServices 创建系统服务依赖
func NewSystemServices(db *gorm.DB, uploadDir string) (*systemservice.ConfigService, *systembiz.ConfigUseCase) {
	// 初始化Repository
	configRepo := systemdata.NewConfigRepo(db)
	loginAttemptRepo := systemdata.NewLoginAttemptRepo(db)

	// 初始化UseCase
	configUseCase := systembiz.NewConfigUseCase(configRepo, loginAttemptRepo)

	// 初始化Service
	configService := systemservice.NewConfigService(configUseCase, uploadDir)

	return configService, configUseCase
}
