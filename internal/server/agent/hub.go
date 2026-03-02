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
	// 关闭旧连接
	if old, ok := h.byAgentID[agentID]; ok {
		close(old.DoneCh)
	}
	h.byAgentID[agentID] = as
	h.byHostID[hostID] = as
	h.mu.Unlock()

	appLogger.Info("Agent已注册", zap.String("agentID", agentID), zap.Uint("hostID", hostID))
	return as
}

// Unregister 注销Agent连接
func (h *AgentHub) Unregister(agentID string) {
	h.mu.Lock()
	if as, ok := h.byAgentID[agentID]; ok {
		delete(h.byAgentID, agentID)
		delete(h.byHostID, as.HostID)
		select {
		case <-as.DoneCh:
		default:
			close(as.DoneCh)
		}
	}
	h.mu.Unlock()
	appLogger.Info("Agent已注销", zap.String("agentID", agentID))
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
