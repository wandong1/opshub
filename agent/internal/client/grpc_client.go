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

// GRPCClient Agent gRPC客户端
type GRPCClient struct {
	cfg        *config.Config
	stream     pb.AgentHub_ConnectClient
	conn       *grpc.ClientConn
	mu         sync.Mutex
	termHandler    TerminalHandler
	fileHandler    FileHandler
	cmdHandler     CommandHandler
	heartbeatInterval int32
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
			fmt.Printf("连接断开: %v, 5秒后重连...\n", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

// connectAndServe 连接并处理消息
func (c *GRPCClient) connectAndServe(ctx context.Context) error {
	tlsConfig, err := c.loadTLS()
	if err != nil {
		return fmt.Errorf("加载TLS失败: %w", err)
	}

	conn, err := grpc.NewClient(c.cfg.ServerAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	)
	if err != nil {
		return fmt.Errorf("gRPC连接失败: %w", err)
	}
	defer conn.Close()
	c.conn = conn

	client := pb.NewAgentHubClient(conn)
	stream, err := client.Connect(ctx)
	if err != nil {
		return fmt.Errorf("建立流失败: %w", err)
	}
	c.mu.Lock()
	c.stream = stream
	c.mu.Unlock()

	// 发送注册
	hostname, _ := os.Hostname()
	ips := getLocalIPs()
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

	// 启动心跳
	go c.heartbeatLoop(ctx)

	// 接收消息循环
	for {
		msg, err := stream.Recv()
		if err != nil {
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
	ticker := time.NewTicker(time.Duration(c.heartbeatInterval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.SendMessage(&pb.AgentMessage{
				Payload: &pb.AgentMessage_Heartbeat{
					Heartbeat: &pb.HeartbeatRequest{
						AgentId: c.cfg.AgentID,
					},
				},
			})
		}
	}
}

// handleServerMessage 处理服务端消息
func (c *GRPCClient) handleServerMessage(msg *pb.ServerMessage) {
	switch payload := msg.Payload.(type) {
	case *pb.ServerMessage_RegisterAck:
		if payload.RegisterAck.Success {
			fmt.Printf("注册成功: %s\n", payload.RegisterAck.Message)
			if payload.RegisterAck.HeartbeatInterval > 0 {
				c.heartbeatInterval = payload.RegisterAck.HeartbeatInterval
			}
		} else {
			fmt.Printf("注册失败: %s\n", payload.RegisterAck.Message)
		}

	case *pb.ServerMessage_HeartbeatAck:
		// 心跳确认，无需处理

	case *pb.ServerMessage_TermOpen:
		if c.termHandler != nil {
			c.termHandler.Open(payload.TermOpen.SessionId, payload.TermOpen.Cols, payload.TermOpen.Rows)
		}
	case *pb.ServerMessage_TermInput:
		if c.termHandler != nil {
			c.termHandler.Input(payload.TermInput.SessionId, payload.TermInput.Data)
		}
	case *pb.ServerMessage_TermResize:
		if c.termHandler != nil {
			c.termHandler.Resize(payload.TermResize.SessionId, payload.TermResize.Cols, payload.TermResize.Rows)
		}
	case *pb.ServerMessage_TermClose:
		if c.termHandler != nil {
			c.termHandler.Close(payload.TermClose.SessionId)
		}

	case *pb.ServerMessage_FileRequest:
		if c.fileHandler != nil {
			req := payload.FileRequest
			resp, err := c.fileHandler.HandleRequest(req.RequestId, req.Action, req.Path, req.Filename, req.Data)
			if err != nil {
				resp = &pb.AgentMessage{
					Payload: &pb.AgentMessage_FileChunk{
						FileChunk: &pb.FileChunk{RequestId: req.RequestId, Error: err.Error()},
					},
				}
			}
			c.SendMessage(resp)
		}

	case *pb.ServerMessage_CmdRequest:
		if c.cmdHandler != nil {
			resp := c.cmdHandler.Execute(payload.CmdRequest.RequestId, payload.CmdRequest.Command, payload.CmdRequest.Timeout)
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
