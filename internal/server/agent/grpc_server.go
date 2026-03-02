package agent

import (
	"fmt"
	"net"

	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/internal/conf"
	agentrepo "github.com/ydcloud-dy/opshub/internal/data/agent"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gorm.io/gorm"
)

// GRPCServer Agent gRPC服务器
type GRPCServer struct {
	server     *grpc.Server
	hub        *AgentHub
	service    *AgentService
	tlsMgr     *TLSManager
	conf       *conf.Config
	db         *gorm.DB
	agentRepo  *agentrepo.Repository
}

// NewGRPCServer 创建gRPC服务器
func NewGRPCServer(cfg *conf.Config, db *gorm.DB) *GRPCServer {
	hub := NewAgentHub()
	repo := agentrepo.NewRepository(db)
	tlsMgr := NewTLSManager(cfg.Agent.CertDir)

	svc := &AgentService{
		hub:       hub,
		agentRepo: repo,
		db:        db,
		cfg:       cfg,
	}

	return &GRPCServer{
		hub:       hub,
		service:   svc,
		tlsMgr:    tlsMgr,
		conf:      cfg,
		db:        db,
		agentRepo: repo,
	}
}

// Hub 返回AgentHub
func (s *GRPCServer) Hub() *AgentHub {
	return s.hub
}

// TLSManager 返回TLS管理器
func (s *GRPCServer) TLSManager() *TLSManager {
	return s.tlsMgr
}

// AgentRepo 返回Agent仓库
func (s *GRPCServer) AgentRepo() *agentrepo.Repository {
	return s.agentRepo
}

// SetServiceLabelRepo 注入服务标签仓库
func (s *GRPCServer) SetServiceLabelRepo(repo asset.ServiceLabelRepo) {
	s.service.serviceLabelRepo = repo
}

// SetHostRepo 注入主机UseCase（用于自动注册）
func (s *GRPCServer) SetHostRepo(hostUseCase interface{ GetHostRepo() asset.HostRepo }) {
	s.service.hostRepo = hostUseCase.GetHostRepo()
}

// Start 启动gRPC服务器
func (s *GRPCServer) Start() error {
	if !s.conf.Agent.Enabled {
		appLogger.Info("Agent功能未启用，跳过gRPC服务器启动")
		return nil
	}

	// 初始化CA
	if err := s.tlsMgr.InitCA(); err != nil {
		return fmt.Errorf("初始化CA失败: %w", err)
	}

	// 加载TLS配置
	tlsConfig, err := s.tlsMgr.LoadServerTLSConfig()
	if err != nil {
		return fmt.Errorf("加载TLS配置失败: %w", err)
	}

	// 创建gRPC服务器
	s.server = grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	pb.RegisterAgentHubServer(s.server, s.service)

	addr := fmt.Sprintf(":%d", s.conf.Server.RPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("gRPC监听失败: %w", err)
	}

	appLogger.Info("Agent gRPC服务器启动", zap.String("addr", addr))
	return s.server.Serve(lis)
}

// Stop 停止gRPC服务器
func (s *GRPCServer) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
		appLogger.Info("Agent gRPC服务器已停止")
	}
}
