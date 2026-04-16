package agent

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/internal/cache"
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
	cacheManager     *cache.CacheManager // 缓存管理器
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

			// 更新缓存中的状态为offline
			if s.cacheManager != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				updates := map[string]any{
					"status": "offline",
				}

				if err := s.cacheManager.UpdateAgentStatus(ctx, agentID, updates); err != nil {
					appLogger.Warn("更新 Agent 离线状态到缓存失败",
						zap.String("agentID", agentID),
						zap.Error(err))
				}
			}
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

		case *pb.AgentMessage_ProbeResult:
			if as != nil {
				as.ResolvePending(payload.ProbeResult.RequestId, payload.ProbeResult)
			}

		case *pb.AgentMessage_HttpProxyResponse:
			if as != nil {
				as.ResolvePending(payload.HttpProxyResponse.RequestId, payload.HttpProxyResponse)
			}
		}
	}
}

// handleRegister 处理Agent注册
func (s *AgentService) handleRegister(stream pb.AgentHub_ConnectServer, req *pb.RegisterRequest) *AgentStream {
	// 查找Agent对应的HostID
	agentInfo, err := s.agentRepo.GetByAgentID(context.Background(), req.AgentId)
	if err != nil {
		// Agent记录不存在 → 检查是否为通过平台部署的场景
		// 优先检查 Host 表中是否已存在该 agent_id（说明是通过部署接口创建的）
		var existingHost assetbiz.Host
		if err := s.db.Where("agent_id = ?", req.AgentId).First(&existingHost).Error; err == nil {
			// Host 已存在但 agent_info 缺失（部署后 agent_info 被意外删除），补充创建
			appLogger.Warn("发现已部署的Agent缺失agent_info记录，正在修复", zap.String("agentID", req.AgentId), zap.Uint("hostID", existingHost.ID))
			now := time.Now()
			agentInfo = &agentmodel.AgentInfo{
				AgentID:  req.AgentId,
				HostID:   existingHost.ID,
				Version:  req.Version,
				Hostname: req.Hostname,
				OS:       req.Os,
				Arch:     req.Arch,
				Status:   "online",
				LastSeen: &now,
			}
			if createErr := s.agentRepo.Create(context.Background(), agentInfo); createErr != nil {
				// 创建失败，可能是 host_id 唯一索引冲突，尝试查询现有记录
				appLogger.Warn("创建agent_info失败，尝试查询现有记录", zap.Error(createErr))
				existingAgentInfo, queryErr := s.agentRepo.GetByHostID(context.Background(), existingHost.ID)
				if queryErr != nil {
					appLogger.Error("查询现有agent_info记录失败", zap.Error(queryErr))
					stream.Send(&pb.ServerMessage{
						Payload: &pb.ServerMessage_RegisterAck{
							RegisterAck: &pb.RegisterResponse{Success: false, Message: "Agent记录异常"},
						},
					})
					return nil
				}
				// 找到了现有记录，但 agent_id 不匹配，说明是重复部署导致的
				// 删除旧记录，使用新的 agent_id
				appLogger.Info("检测到重复部署，删除旧agent_info记录",
					zap.String("oldAgentID", existingAgentInfo.AgentID),
					zap.String("newAgentID", req.AgentId))
				s.agentRepo.Delete(context.Background(), existingAgentInfo.AgentID)
				// 重新创建
				if retryErr := s.agentRepo.Create(context.Background(), agentInfo); retryErr != nil {
					appLogger.Error("重新创建agent_info记录失败", zap.Error(retryErr))
					stream.Send(&pb.ServerMessage{
						Payload: &pb.ServerMessage_RegisterAck{
							RegisterAck: &pb.RegisterResponse{Success: false, Message: "Agent记录创建失败"},
						},
					})
					return nil
				}
			}
		} else {
			// Host 不存在 → 可能是手动安装场景，也可能是 SSH 部署但 IP 匹配失败
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

			// 关键修复：检查 IP 是否已存在（SSH 部署场景）
			// 从 Agent 上报的 IP 列表中提取主 IP
			var detectedIP string
			for _, addr := range req.Ips {
				if addr != "" && addr != "127.0.0.1" && addr != "::1" {
					detectedIP = addr
					break
				}
			}
			if detectedIP == "" && len(req.Ips) > 0 {
				detectedIP = req.Ips[0]
			}
			// 兜底：从 gRPC 连接提取 IP
			if detectedIP == "" {
				if p, ok := peer.FromContext(stream.Context()); ok {
					if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
						detectedIP = tcpAddr.IP.String()
					} else {
						host, _, splitErr := net.SplitHostPort(p.Addr.String())
						if splitErr == nil {
							detectedIP = host
						}
					}
				}
			}

			// 通过 IP 查找是否已有主机记录（SSH 部署后 Agent 首次连接的场景）
			var hostByIP assetbiz.Host
			if detectedIP != "" && detectedIP != "unknown" {
				if err := s.db.Where("ip = ?", detectedIP).First(&hostByIP).Error; err == nil {
					// IP 已存在，更新该主机的 agent_id 和 hostname
					appLogger.Info("检测到 IP 已存在的主机，更新 agent_id 和 hostname",
						zap.String("ip", detectedIP),
						zap.Uint("hostID", hostByIP.ID),
						zap.String("oldAgentID", hostByIP.AgentID),
						zap.String("newAgentID", req.AgentId),
						zap.String("oldHostname", hostByIP.Hostname),
						zap.String("newHostname", req.Hostname))

					// 删除旧的 agent_info 记录（如果存在）
					if hostByIP.AgentID != "" {
						s.agentRepo.Delete(context.Background(), hostByIP.AgentID)
					}

					// 更新 Host 记录
					s.db.Model(&assetbiz.Host{}).Where("id = ?", hostByIP.ID).Updates(map[string]any{
						"agent_id":        req.AgentId,
						"agent_status":    "online",
						"connection_mode": "agent",
						"hostname":        req.Hostname,
						"os":              req.Os,
						"arch":            req.Arch,
					})

					// 创建新的 agent_info 记录
					now := time.Now()
					agentInfo = &agentmodel.AgentInfo{
						AgentID:  req.AgentId,
						HostID:   hostByIP.ID,
						Version:  req.Version,
						Hostname: req.Hostname,
						OS:       req.Os,
						Arch:     req.Arch,
						Status:   "online",
						LastSeen: &now,
					}
					if createErr := s.agentRepo.Create(context.Background(), agentInfo); createErr != nil {
						appLogger.Error("创建 agent_info 记录失败", zap.Error(createErr))
						stream.Send(&pb.ServerMessage{
							Payload: &pb.ServerMessage_RegisterAck{
								RegisterAck: &pb.RegisterResponse{Success: false, Message: "Agent记录创建失败"},
							},
						})
						return nil
					}
				} else {
					// IP 不存在，真正的手动安装场景，创建新主机
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
			} else {
				// 无法获取 IP，回退到自动注册
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
		}
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

	// Debug 日志：记录收到心跳
	appLogger.Debug("收到 Agent 心跳",
		zap.String("agentID", req.AgentId),
		zap.Time("time", now))

	// 使用缓存管理器更新状态（同步执行，但内部会快速返回）
	if s.cacheManager != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		updates := map[string]any{
			"status":    "online",
			"last_seen": now,
		}

		if err := s.cacheManager.UpdateAgentStatus(ctx, req.AgentId, updates); err != nil {
			appLogger.Warn("更新 Agent 状态失败",
				zap.String("agentID", req.AgentId),
				zap.Error(err))
		}
	} else {
		// 降级：直接更新数据库
		s.agentRepo.UpdateInfo(context.Background(), req.AgentId, map[string]any{
			"status":    "online",
			"last_seen": now,
		})
	}

	// 立即响应心跳
	as.Send(&pb.ServerMessage{
		Payload: &pb.ServerMessage_HeartbeatAck{
			HeartbeatAck: &pb.HeartbeatResponse{Success: true},
		},
	})
}

// SetCacheManager 设置缓存管理器
func (s *AgentService) SetCacheManager(manager *cache.CacheManager) {
	s.cacheManager = manager
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
