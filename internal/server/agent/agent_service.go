package agent

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	agentrepo "github.com/ydcloud-dy/opshub/internal/data/agent"
	agentmodel "github.com/ydcloud-dy/opshub/internal/agent"
	"github.com/ydcloud-dy/opshub/internal/conf"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/peer"
	"gorm.io/gorm"
)

// AgentService 实现AgentHub gRPC服务
type AgentService struct {
	pb.UnimplementedAgentHubServer
	hub              *AgentHub
	agentRepo        *agentrepo.Repository
	db               *gorm.DB
	cfg              *conf.Config
	hostRepo         assetbiz.HostRepo
	serviceLabelRepo assetbiz.ServiceLabelRepo
}

// Connect 处理Agent双向流连接
func (s *AgentService) Connect(stream pb.AgentHub_ConnectServer) error {
	var as *AgentStream
	var agentID string

	defer func() {
		if agentID != "" {
			s.hub.Unregister(agentID)
			// 更新状态为offline
			s.agentRepo.UpdateStatus(context.Background(), agentID, "offline")
			s.db.Model(&struct{ AgentStatus string }{}).
				Exec("UPDATE hosts SET agent_status = 'offline' WHERE agent_id = ?", agentID)
		}
	}()

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			appLogger.Error("Agent流接收错误", zap.Error(err), zap.String("agentID", agentID))
			return err
		}

		switch payload := msg.Payload.(type) {
		case *pb.AgentMessage_Register:
			agentID = payload.Register.AgentId
			as = s.handleRegister(stream, payload.Register)

		case *pb.AgentMessage_Heartbeat:
			if as != nil {
				s.handleHeartbeat(as, payload.Heartbeat)
			}

		case *pb.AgentMessage_TermOutput:
			if as != nil {
				s.hub.HandleTerminalOutput(payload.TermOutput.SessionId, payload.TermOutput.Data)
			}

		case *pb.AgentMessage_FileList:
			if as != nil {
				as.ResolvePending(payload.FileList.RequestId, payload.FileList)
			}

		case *pb.AgentMessage_FileChunk:
			if as != nil {
				as.ResolvePending(payload.FileChunk.RequestId, payload.FileChunk)
			}

		case *pb.AgentMessage_CmdResult:
			if as != nil {
				as.ResolvePending(payload.CmdResult.RequestId, payload.CmdResult)
			}
		}
	}
}

// handleRegister 处理Agent注册
func (s *AgentService) handleRegister(stream pb.AgentHub_ConnectServer, req *pb.RegisterRequest) *AgentStream {
	// 查找Agent对应的HostID
	agentInfo, err := s.agentRepo.GetByAgentID(context.Background(), req.AgentId)
	if err != nil {
		// Agent记录不存在 → 尝试自动注册
		if s.hostRepo == nil {
			appLogger.Error("Agent注册失败: 未找到Agent记录且未配置自动注册", zap.String("agentID", req.AgentId))
			stream.Send(&pb.ServerMessage{
				Payload: &pb.ServerMessage_RegisterAck{
					RegisterAck: &pb.RegisterResponse{Success: false, Message: "Agent未注册"},
				},
			})
			return nil
		}

		// 验证agent_id格式
		if _, parseErr := uuid.Parse(req.AgentId); parseErr != nil {
			appLogger.Error("Agent自动注册失败: AgentID格式无效", zap.String("agentID", req.AgentId))
			stream.Send(&pb.ServerMessage{
				Payload: &pb.ServerMessage_RegisterAck{
					RegisterAck: &pb.RegisterResponse{Success: false, Message: "AgentID格式无效"},
				},
			})
			return nil
		}

		agentInfo, err = s.autoRegisterHost(stream, req)
		if err != nil {
			appLogger.Error("Agent自动注册失败", zap.String("agentID", req.AgentId), zap.Error(err))
			stream.Send(&pb.ServerMessage{
				Payload: &pb.ServerMessage_RegisterAck{
					RegisterAck: &pb.RegisterResponse{Success: false, Message: "自动注册失败: " + err.Error()},
				},
			})
			return nil
		}
		appLogger.Info("Agent自动注册成功", zap.String("agentID", req.AgentId), zap.Uint("hostID", agentInfo.HostID))
	}

	// 注册到Hub
	as := s.hub.Register(req.AgentId, agentInfo.HostID, stream)

	// 更新Agent信息
	now := time.Now()
	updates := map[string]any{
		"status":    "online",
		"version":   req.Version,
		"hostname":  req.Hostname,
		"os":        req.Os,
		"arch":      req.Arch,
		"last_seen": &now,
	}
	s.agentRepo.UpdateInfo(context.Background(), req.AgentId, updates)

	// 更新Host表状态
	s.db.Exec("UPDATE hosts SET agent_status = 'online' WHERE agent_id = ?", req.AgentId)

	// 发送注册确认
	stream.Send(&pb.ServerMessage{
		Payload: &pb.ServerMessage_RegisterAck{
			RegisterAck: &pb.RegisterResponse{
				Success:           true,
				Message:           "注册成功",
				HeartbeatInterval: int32(s.cfg.Agent.HeartbeatTimeout / 3),
			},
		},
	})

	// 启动发送协程
	go s.sendLoop(as)

	// 异步检测进程标签
	if s.serviceLabelRepo != nil {
		go s.detectAndApplyServiceLabels(as, agentInfo.HostID)
	}

	appLogger.Info("Agent注册成功",
		zap.String("agentID", req.AgentId),
		zap.Uint("hostID", agentInfo.HostID),
		zap.String("version", req.Version))
	return as
}

// handleHeartbeat 处理心跳
func (s *AgentService) handleHeartbeat(as *AgentStream, req *pb.HeartbeatRequest) {
	now := time.Now()
	s.agentRepo.UpdateInfo(context.Background(), req.AgentId, map[string]any{
		"status":    "online",
		"last_seen": &now,
	})

	as.Send(&pb.ServerMessage{
		Payload: &pb.ServerMessage_HeartbeatAck{
			HeartbeatAck: &pb.HeartbeatResponse{Success: true},
		},
	})
}

// autoRegisterHost 自动注册主机和AgentInfo
func (s *AgentService) autoRegisterHost(stream pb.AgentHub_ConnectServer, req *pb.RegisterRequest) (*agentmodel.AgentInfo, error) {
	// 选取第一个非loopback IP
	ip := ""
	for _, addr := range req.Ips {
		if addr != "" && addr != "127.0.0.1" && addr != "::1" {
			ip = addr
			break
		}
	}
	if ip == "" && len(req.Ips) > 0 {
		ip = req.Ips[0]
	}
	// 兜底：从 gRPC 连接的远端地址提取 IP
	if ip == "" {
		if p, ok := peer.FromContext(stream.Context()); ok {
			if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
				ip = tcpAddr.IP.String()
			} else {
				host, _, err := net.SplitHostPort(p.Addr.String())
				if err == nil {
					ip = host
				}
			}
		}
	}
	if ip == "" {
		ip = "unknown"
	}

	hostname := req.Hostname
	if hostname == "" {
		hostname = ip
	}

	// 查找或创建"未分组"默认分组
	var defaultGroup assetbiz.AssetGroup
	if err := s.db.Where("code = ?", "agent-auto-register").First(&defaultGroup).Error; err != nil {
		defaultGroup = assetbiz.AssetGroup{
			Name:        "Agent自动注册",
			Code:        "agent-auto-register",
			ParentID:    0,
			Description: "Agent手动部署自动注册的主机",
			Sort:        999,
			Status:      1,
		}
		if err := s.db.Create(&defaultGroup).Error; err != nil {
			return nil, fmt.Errorf("创建默认分组失败: %w", err)
		}
	}

	// 创建Host记录
	host := &assetbiz.Host{
		Name:           hostname,
		IP:             ip,
		Port:           22,
		Type:           "self",
		GroupID:        defaultGroup.ID,
		Status:         1,
		AgentID:        req.AgentId,
		AgentStatus:    "online",
		ConnectionMode: "agent",
		OS:             req.Os,
		Arch:           req.Arch,
		Hostname:       hostname,
		SSHUser:        "root",
	}

	if err := s.hostRepo.Create(context.Background(), host); err != nil {
		return nil, err
	}

	// 创建AgentInfo记录
	now := time.Now()
	agentInfo := &agentmodel.AgentInfo{
		AgentID:  req.AgentId,
		HostID:   host.ID,
		Version:  req.Version,
		Hostname: hostname,
		OS:       req.Os,
		Arch:     req.Arch,
		Status:   "online",
		LastSeen: &now,
	}
	if err := s.agentRepo.Create(context.Background(), agentInfo); err != nil {
		return nil, err
	}

	return agentInfo, nil
}

// detectAndApplyServiceLabels 异步检测进程并打标签
func (s *AgentService) detectAndApplyServiceLabels(as *AgentStream, hostID uint) {
	// 等待stream就绪
	time.Sleep(2 * time.Second)

	// 通过Agent执行命令获取进程列表
	requestID := uuid.New().String()
	as.RegisterPending(requestID)
	as.Send(&pb.ServerMessage{
		Payload: &pb.ServerMessage_CmdRequest{
			CmdRequest: &pb.CommandRequest{
				RequestId: requestID,
				Command:   "ps -eo comm= | sort -u",
				Timeout:   10,
			},
		},
	})

	result, err := s.hub.WaitResponse(as, requestID, 15*time.Second)
	if err != nil {
		appLogger.Warn("检测服务标签失败: 执行命令超时", zap.Uint("hostID", hostID), zap.Error(err))
		return
	}

	cmdResult, ok := result.(*pb.CommandResult)
	if !ok || cmdResult.ExitCode != 0 {
		appLogger.Warn("检测服务标签失败: 命令执行失败", zap.Uint("hostID", hostID))
		return
	}

	// 获取所有启用的标签
	labels, err := s.serviceLabelRepo.GetAllEnabled(context.Background())
	if err != nil || len(labels) == 0 {
		return
	}

	// 解析进程列表
	processes := make(map[string]bool)
	for _, line := range strings.Split(cmdResult.Stdout, "\n") {
		proc := strings.TrimSpace(line)
		if proc != "" {
			processes[proc] = true
		}
	}

	// 匹配标签
	var matchedTags []string
	for _, label := range labels {
		for _, proc := range strings.Split(label.MatchProcesses, ",") {
			proc = strings.TrimSpace(proc)
			if proc != "" && processes[proc] {
				matchedTags = append(matchedTags, label.Name)
				break
			}
		}
	}

	// 更新Host的Tags字段
	if len(matchedTags) > 0 {
		tags := strings.Join(matchedTags, ",")
		s.db.Exec("UPDATE hosts SET tags = ? WHERE id = ?", tags, hostID)
		appLogger.Info("服务标签检测完成", zap.Uint("hostID", hostID), zap.String("tags", tags))
	}
}

// sendLoop 发送消息循环
func (s *AgentService) sendLoop(as *AgentStream) {
	for {
		select {
		case msg := <-as.SendCh:
			if err := as.Stream.Send(msg); err != nil {
				appLogger.Error("发送消息到Agent失败",
					zap.String("agentID", as.AgentID), zap.Error(err))
				return
			}
		case <-as.DoneCh:
			return
		}
	}
}
