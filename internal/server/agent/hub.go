package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
)

// AgentStream 封装一个Agent的gRPC双向流
type AgentStream struct {
	AgentID  string
	HostID   uint
	Stream   pb.AgentHub_ConnectServer
	SendCh   chan *pb.ServerMessage
	DoneCh   chan struct{}
	mu       sync.Mutex
	// 请求响应回调
	pending  map[string]chan any
	pendMu   sync.Mutex
}

// Send 线程安全地发送消息
func (s *AgentStream) Send(msg *pb.ServerMessage) error {
	select {
	case s.SendCh <- msg:
		return nil
	case <-s.DoneCh:
		return fmt.Errorf("agent stream closed")
	}
}

// RegisterPending 注册一个等待响应的请求
func (s *AgentStream) RegisterPending(requestID string) chan any {
	s.pendMu.Lock()
	defer s.pendMu.Unlock()
	ch := make(chan any, 1)
	s.pending[requestID] = ch
	return ch
}

// ResolvePending 完成一个等待中的请求
func (s *AgentStream) ResolvePending(requestID string, result any) {
	s.pendMu.Lock()
	ch, ok := s.pending[requestID]
	if ok {
		delete(s.pending, requestID)
	}
	s.pendMu.Unlock()
	if ok {
		ch <- result
	}
}

// AgentHub 管理所有Agent连接
type AgentHub struct {
	mu            sync.RWMutex
	byAgentID     map[string]*AgentStream
	byHostID      map[uint]*AgentStream
	// 终端输出回调: sessionID -> callback
	termCallbacks map[string]func(data []byte)
	termMu        sync.RWMutex
}

// NewAgentHub 创建AgentHub
func NewAgentHub() *AgentHub {
	return &AgentHub{
		byAgentID:     make(map[string]*AgentStream),
		byHostID:      make(map[uint]*AgentStream),
		termCallbacks: make(map[string]func(data []byte)),
	}
}

// Register 注册Agent连接
func (h *AgentHub) Register(agentID string, hostID uint, stream pb.AgentHub_ConnectServer) *AgentStream {
	as := &AgentStream{
		AgentID: agentID,
		HostID:  hostID,
		Stream:  stream,
		SendCh:  make(chan *pb.ServerMessage, 64),
		DoneCh:  make(chan struct{}),
		pending: make(map[string]chan any),
	}

	h.mu.Lock()
	// 检查旧连接
	if old, ok := h.byAgentID[agentID]; ok {
		appLogger.Warn("Agent重复连接，替换旧连接",
			zap.String("agent_id", agentID),
			zap.Uint("old_host_id", old.HostID),
			zap.Uint("new_host_id", hostID),
		)
		// 不立即关闭 DoneCh，让正在进行的请求自然完成或超时
		// 只是从映射表中移除，避免新请求使用旧连接
	}
	h.byAgentID[agentID] = as
	h.byHostID[hostID] = as
	totalAgents := len(h.byHostID)
	h.mu.Unlock()

	appLogger.Info("Agent已注册",
		zap.String("agent_id", agentID),
		zap.Uint("host_id", hostID),
		zap.Int("total_agents", totalAgents),
	)
	return as
}

// Unregister 注销Agent连接
func (h *AgentHub) Unregister(agentID string) {
	h.mu.Lock()
	var hostID uint
	if as, ok := h.byAgentID[agentID]; ok {
		hostID = as.HostID
		delete(h.byAgentID, agentID)
		delete(h.byHostID, as.HostID)
		select {
		case <-as.DoneCh:
		default:
			close(as.DoneCh)
		}
	}
	totalAgents := len(h.byHostID)
	h.mu.Unlock()

	appLogger.Info("Agent已注销",
		zap.String("agent_id", agentID),
		zap.Uint("host_id", hostID),
		zap.Int("total_agents", totalAgents),
	)
}

// GetByHostID 根据HostID获取Agent流
func (h *AgentHub) GetByHostID(hostID uint) (*AgentStream, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	as, ok := h.byHostID[hostID]
	return as, ok
}

// GetByAgentID 根据AgentID获取Agent流
func (h *AgentHub) GetByAgentID(agentID string) (*AgentStream, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	as, ok := h.byAgentID[agentID]
	return as, ok
}

// IsOnline 检查Agent是否在线
func (h *AgentHub) IsOnline(hostID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.byHostID[hostID]

	appLogger.Info("Checking agent online status",
		zap.Uint("host_id", hostID),
		zap.Bool("is_online", ok),
		zap.Int("total_agents", len(h.byHostID)),
	)

	return ok
}

// RegisterTerminalCallback 注册终端输出回调
func (h *AgentHub) RegisterTerminalCallback(sessionID string, cb func(data []byte)) {
	h.termMu.Lock()
	defer h.termMu.Unlock()
	h.termCallbacks[sessionID] = cb
}

// UnregisterTerminalCallback 注销终端输出回调
func (h *AgentHub) UnregisterTerminalCallback(sessionID string) {
	h.termMu.Lock()
	defer h.termMu.Unlock()
	delete(h.termCallbacks, sessionID)
}

// HandleTerminalOutput 处理终端输出
func (h *AgentHub) HandleTerminalOutput(sessionID string, data []byte) {
	h.termMu.RLock()
	cb, ok := h.termCallbacks[sessionID]
	h.termMu.RUnlock()
	if ok {
		cb(data)
	}
}

// WaitResponse 等待Agent响应，带超时
func (h *AgentHub) WaitResponse(as *AgentStream, requestID string, timeout time.Duration) (any, error) {
	ch := as.RegisterPending(requestID)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		as.pendMu.Lock()
		delete(as.pending, requestID)
		as.pendMu.Unlock()
		return nil, fmt.Errorf("等待Agent响应超时")
	case <-as.DoneCh:
		return nil, fmt.Errorf("Agent连接已断开")
	}
}

// SendProbeRequest 发送拨测请求到Agent并等待响应
func (h *AgentHub) SendProbeRequest(hostID uint, req *pb.ProbeRequest) (*pb.ProbeResult, error) {
	as, ok := h.GetByHostID(hostID)
	if !ok {
		return nil, fmt.Errorf("Agent未连接")
	}

	// 发送拨测请求
	msg := &pb.ServerMessage{
		Payload: &pb.ServerMessage_ProbeRequest{
			ProbeRequest: req,
		},
	}

	if err := as.Send(msg); err != nil {
		return nil, fmt.Errorf("发送拨测请求失败: %w", err)
	}

	// 等待响应
	result, err := h.WaitResponse(as, req.RequestId, 35*time.Second)
	if err != nil {
		return nil, err
	}

	probeResult, ok := result.(*pb.ProbeResult)
	if !ok {
		return nil, fmt.Errorf("响应类型错误")
	}

	return probeResult, nil
}

// CloseAll 关闭所有Agent连接
func (h *AgentHub) CloseAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	appLogger.Info("正在关闭所有Agent连接", zap.Int("count", len(h.byAgentID)))

	for agentID, as := range h.byAgentID {
		select {
		case <-as.DoneCh:
			// 已经关闭
		default:
			close(as.DoneCh)
			appLogger.Debug("已关闭Agent连接", zap.String("agentID", agentID))
		}
	}

	// 清空所有连接
	h.byAgentID = make(map[string]*AgentStream)
	h.byHostID = make(map[uint]*AgentStream)

	appLogger.Info("所有Agent连接已关闭")
}
