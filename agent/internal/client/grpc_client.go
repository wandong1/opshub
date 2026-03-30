package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ydcloud-dy/opshub/agent/internal/config"
	"github.com/ydcloud-dy/opshub/agent/internal/logger"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// TerminalHandler 终端操作回调
type TerminalHandler interface {
	Open(sessionID string, cols, rows uint32) error
	Input(sessionID string, data []byte) error
	Resize(sessionID string, cols, rows uint32) error
	Close(sessionID string) error
}

// FileHandler 文件操作回调
type FileHandler interface {
	HandleRequest(requestID, action, path, filename string, data []byte) (*pb.AgentMessage, error)
}

// CommandHandler 命令执行回调
type CommandHandler interface {
	Execute(requestID, command string, timeout int32) *pb.AgentMessage
}

// ProbeHandler 拨测回调
type ProbeHandler interface {
	Probe(req *pb.ProbeRequest) *pb.ProbeResult
}

// GRPCClient Agent gRPC客户端
type GRPCClient struct {
	cfg               *config.Config
	stream            pb.AgentHub_ConnectClient
	conn              *grpc.ClientConn
	mu                sync.Mutex
	termHandler       TerminalHandler
	fileHandler       FileHandler
	cmdHandler        CommandHandler
	probeHandler      ProbeHandler
	heartbeatInterval int32
	intervalMu        sync.RWMutex
	heartbeatCancel   context.CancelFunc
	heartbeatMu       sync.Mutex
}

// NewGRPCClient 创建gRPC客户端
func NewGRPCClient(cfg *config.Config) *GRPCClient {
	return &GRPCClient{cfg: cfg, heartbeatInterval: 30}
}

// SetHandlers 设置处理器
func (c *GRPCClient) SetHandlers(term TerminalHandler, file FileHandler, cmd CommandHandler) {
	c.termHandler = term
	c.fileHandler = file
	c.cmdHandler = cmd
}

// SetProbeHandler 设置拨测处理器
func (c *GRPCClient) SetProbeHandler(probe ProbeHandler) {
	c.probeHandler = probe
}

// SendMessage 发送消息到服务端
func (c *GRPCClient) SendMessage(msg *pb.AgentMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stream == nil {
		return fmt.Errorf("stream not connected")
	}
	return c.stream.Send(msg)
}

// Run 启动客户端，自动重连
func (c *GRPCClient) Run(ctx context.Context) error {
	for {
		if err := c.connectAndServe(ctx); err != nil {
			logger.Warn("连接断开: %v, 5秒后重连...", err)
		}
		select {
		case <-ctx.Done():
			logger.Info("收到退出信号，停止重连")
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

// connectAndServe 连接并处理消息
func (c *GRPCClient) connectAndServe(ctx context.Context) error {
	logger.Info("正在连接到服务器: %s", c.cfg.ServerAddr)

	tlsConfig, err := c.loadTLS()
	if err != nil {
		logger.Error("加载TLS失败: %v", err)
		return fmt.Errorf("加载TLS失败: %w", err)
	}

	conn, err := grpc.NewClient(c.cfg.ServerAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	)
	if err != nil {
		logger.Error("gRPC连接失败: %v", err)
		return fmt.Errorf("gRPC连接失败: %w", err)
	}
	defer conn.Close()
	c.conn = conn

	client := pb.NewAgentHubClient(conn)
	stream, err := client.Connect(ctx)
	if err != nil {
		logger.Error("建立流失败: %v", err)
		return fmt.Errorf("建立流失败: %w", err)
	}
	c.mu.Lock()
	c.stream = stream
	c.mu.Unlock()

	// 发送注册
	hostname, _ := os.Hostname()
	ips := getLocalIPs()
	logger.Info("发送注册请求 - AgentID: %s, Hostname: %s, OS: %s, Arch: %s",
		c.cfg.AgentID, hostname, runtime.GOOS, runtime.GOARCH)

	c.SendMessage(&pb.AgentMessage{
		Payload: &pb.AgentMessage_Register{
			Register: &pb.RegisterRequest{
				AgentId:  c.cfg.AgentID,
				Hostname: hostname,
				Os:       runtime.GOOS,
				Arch:     runtime.GOARCH,
				Version:  "1.0.0",
				Ips:      ips,
			},
		},
	})

	// 停止旧的心跳循环（如果存在）
	c.heartbeatMu.Lock()
	if c.heartbeatCancel != nil {
		c.heartbeatCancel()
	}
	// 创建新的心跳上下文
	heartbeatCtx, cancel := context.WithCancel(ctx)
	c.heartbeatCancel = cancel
	c.heartbeatMu.Unlock()

	// 启动心跳
	c.intervalMu.RLock()
	interval := c.heartbeatInterval
	c.intervalMu.RUnlock()
	logger.Info("启动心跳循环，间隔: %d秒", interval)
	go c.heartbeatLoop(heartbeatCtx)

	// 接收消息循环
	for {
		msg, err := stream.Recv()
		if err != nil {
			logger.Error("接收消息失败: %v", err)
			return fmt.Errorf("接收消息失败: %w", err)
		}
		go c.handleServerMessage(msg)
	}
}

// loadTLS 加载mTLS配置
func (c *GRPCClient) loadTLS() (*tls.Config, error) {
	certDir := c.cfg.CertDir
	caCert, err := os.ReadFile(filepath.Join(certDir, "ca.pem"))
	if err != nil {
		return nil, fmt.Errorf("读取CA证书失败: %w", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("解析CA证书失败")
	}

	cert, err := tls.LoadX509KeyPair(
		filepath.Join(certDir, "cert.pem"),
		filepath.Join(certDir, "key.pem"),
	)
	if err != nil {
		return nil, fmt.Errorf("加载客户端证书失败: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		ServerName:   "localhost",
	}, nil
}

// heartbeatLoop 心跳循环
func (c *GRPCClient) heartbeatLoop(ctx context.Context) {
	c.intervalMu.RLock()
	interval := time.Duration(c.heartbeatInterval) * time.Second
	c.intervalMu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Debug("心跳循环退出")
			return
		case <-ticker.C:
			logger.Debug("发送心跳 - AgentID: %s", c.cfg.AgentID)
			err := c.SendMessage(&pb.AgentMessage{
				Payload: &pb.AgentMessage_Heartbeat{
					Heartbeat: &pb.HeartbeatRequest{
						AgentId: c.cfg.AgentID,
					},
				},
			})
			if err != nil {
				logger.Error("发送心跳失败: %v", err)
			}
		}
	}
}

// handleServerMessage 处理服务端消息
func (c *GRPCClient) handleServerMessage(msg *pb.ServerMessage) {
	switch payload := msg.Payload.(type) {
	case *pb.ServerMessage_RegisterAck:
		if payload.RegisterAck.Success {
			logger.Info("注册成功: %s", payload.RegisterAck.Message)
			if payload.RegisterAck.HeartbeatInterval > 0 {
				c.intervalMu.Lock()
				c.heartbeatInterval = payload.RegisterAck.HeartbeatInterval
				c.intervalMu.Unlock()
				logger.Info("心跳间隔已更新为: %d 秒", payload.RegisterAck.HeartbeatInterval)
			}
		} else {
			logger.Warn("注册失败: %s", payload.RegisterAck.Message)
		}

	case *pb.ServerMessage_HeartbeatAck:
		logger.Debug("收到心跳确认")

	case *pb.ServerMessage_TermOpen:
		if c.termHandler != nil {
			logger.Info("打开终端会话: sessionID=%s, cols=%d, rows=%d", payload.TermOpen.SessionId, payload.TermOpen.Cols, payload.TermOpen.Rows)
			c.termHandler.Open(payload.TermOpen.SessionId, payload.TermOpen.Cols, payload.TermOpen.Rows)
		}
	case *pb.ServerMessage_TermInput:
		if c.termHandler != nil {
			logger.Debug("终端输入: sessionID=%s, len=%d", payload.TermInput.SessionId, len(payload.TermInput.Data))
			c.termHandler.Input(payload.TermInput.SessionId, payload.TermInput.Data)
		}
	case *pb.ServerMessage_TermResize:
		if c.termHandler != nil {
			logger.Debug("终端调整大小: sessionID=%s, cols=%d, rows=%d", payload.TermResize.SessionId, payload.TermResize.Cols, payload.TermResize.Rows)
			c.termHandler.Resize(payload.TermResize.SessionId, payload.TermResize.Cols, payload.TermResize.Rows)
		}
	case *pb.ServerMessage_TermClose:
		if c.termHandler != nil {
			logger.Info("关闭终端会话: sessionID=%s", payload.TermClose.SessionId)
			c.termHandler.Close(payload.TermClose.SessionId)
		}

	case *pb.ServerMessage_FileRequest:
		if c.fileHandler != nil {
			req := payload.FileRequest
			logger.Info("收到文件请求: action=%s, path=%s, filename=%s, requestID=%s", req.Action, req.Path, req.Filename, req.RequestId)
			resp, err := c.fileHandler.HandleRequest(req.RequestId, req.Action, req.Path, req.Filename, req.Data)
			if err != nil {
				logger.Error("文件操作失败: action=%s, path=%s, err=%v", req.Action, req.Path, err)
				resp = &pb.AgentMessage{
					Payload: &pb.AgentMessage_FileChunk{
						FileChunk: &pb.FileChunk{RequestId: req.RequestId, Error: err.Error()},
					},
				}
			} else {
				logger.Debug("文件操作完成: action=%s, path=%s", req.Action, req.Path)
			}
			c.SendMessage(resp)
		}

	case *pb.ServerMessage_CmdRequest:
		if c.cmdHandler != nil {
			logger.Info("收到命令请求: requestID=%s, command=%s", payload.CmdRequest.RequestId, payload.CmdRequest.Command)
			resp := c.cmdHandler.Execute(payload.CmdRequest.RequestId, payload.CmdRequest.Command, payload.CmdRequest.Timeout)
			logger.Debug("命令执行完成: requestID=%s", payload.CmdRequest.RequestId)
			c.SendMessage(resp)
		}

	case *pb.ServerMessage_ProbeRequest:
		if c.probeHandler != nil {
			logger.Info("收到拨测请求: type=%s, target=%s, url=%s, requestID=%s",
				payload.ProbeRequest.ProbeType, payload.ProbeRequest.Target,
				payload.ProbeRequest.Url, payload.ProbeRequest.RequestId)
			result := c.probeHandler.Probe(payload.ProbeRequest)
			if result.Success {
				logger.Info("拨测完成: type=%s, success=true, latency=%.2fms, requestID=%s",
					payload.ProbeRequest.ProbeType, result.Latency, payload.ProbeRequest.RequestId)
			} else {
				logger.Warn("拨测完成: type=%s, success=false, error=%s, requestID=%s",
					payload.ProbeRequest.ProbeType, result.Error, payload.ProbeRequest.RequestId)
			}
			resp := &pb.AgentMessage{
				Payload: &pb.AgentMessage_ProbeResult{
					ProbeResult: result,
				},
			}
			c.SendMessage(resp)
		}
	}
}

// getLocalIPs 获取本机非loopback IP列表
func getLocalIPs() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ips = append(ips, ipNet.IP.String())
			} else {
				// 跳过link-local IPv6
				if !strings.HasPrefix(ipNet.IP.String(), "fe80") {
					ips = append(ips, ipNet.IP.String())
				}
			}
		}
	}
	return ips
}
