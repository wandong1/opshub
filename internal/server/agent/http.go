package agent

import (
	"github.com/gin-gonic/gin"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	rbacService "github.com/ydcloud-dy/opshub/internal/service/rbac"
	"gorm.io/gorm"
)

// HTTPServer Agent HTTP服务
type HTTPServer struct {
	hub            *AgentHub
	hostUseCase    *assetbiz.HostUseCase
	db             *gorm.DB
	authMiddleware *rbacService.AuthMiddleware
	tlsMgr         *TLSManager
	grpcServer     *GRPCServer
}

// NewHTTPServer 创建Agent HTTP服务
func NewHTTPServer(
	grpcServer *GRPCServer,
	hostUseCase *assetbiz.HostUseCase,
	db *gorm.DB,
	authMiddleware *rbacService.AuthMiddleware,
) *HTTPServer {
	return &HTTPServer{
		hub:            grpcServer.Hub(),
		hostUseCase:    hostUseCase,
		db:             db,
		authMiddleware: authMiddleware,
		tlsMgr:         grpcServer.TLSManager(),
		grpcServer:     grpcServer,
	}
}

// RegisterRoutes 注册Agent相关路由
func (s *HTTPServer) RegisterRoutes(r *gin.RouterGroup) {
	agents := r.Group("/agents")
	{
		agents.GET("/statuses", s.GetAllStatuses)
		agents.GET("/:hostId/status", s.GetAgentStatus)
		agents.POST("/:hostId/deploy", s.DeployAgent)
		agents.PUT("/:hostId/update", s.UpdateAgent)
		agents.DELETE("/:hostId/uninstall", s.UninstallAgent)
		agents.POST("/batch-deploy", s.BatchDeployAgent)
		agents.GET("/:hostId/terminal", s.HandleTerminal)
		agents.GET("/:hostId/files", s.ListFiles)
		agents.POST("/:hostId/files/upload", s.UploadFile)
		agents.GET("/:hostId/files/download", s.DownloadFile)
		agents.DELETE("/:hostId/files", s.DeleteFile)
		agents.POST("/:hostId/execute", s.ExecuteCommand)
		agents.POST("/generate-install", s.GenerateInstallPackage)
	}
}
