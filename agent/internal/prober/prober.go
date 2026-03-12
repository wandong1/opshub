package prober

import (
	"context"
	"time"

	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
)

// Prober 拨测器接口
type Prober interface {
	Probe(ctx context.Context, req *pb.ProbeRequest) *pb.ProbeResult
}

// Manager 拨测管理器
type Manager struct {
	probers map[string]Prober
}

// NewManager 创建拨测管理器
func NewManager() *Manager {
	m := &Manager{
		probers: make(map[string]Prober),
	}

	// 注册各类拨测器
	m.probers["ping"] = &PingProber{}
	m.probers["tcp"] = &TCPProber{}
	m.probers["udp"] = &UDPProber{}
	m.probers["http"] = &HTTPProber{}
	m.probers["https"] = &HTTPProber{}
	m.probers["websocket"] = &WebSocketProber{}

	return m
}

// Probe 执行拨测
func (m *Manager) Probe(req *pb.ProbeRequest) *pb.ProbeResult {
	prober, ok := m.probers[req.ProbeType]
	if !ok {
		return &pb.ProbeResult{
			RequestId: req.RequestId,
			Success:   false,
			Error:     "unsupported probe type: " + req.ProbeType,
		}
	}

	// 设置超时
	timeout := time.Duration(req.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 执行拨测
	result := prober.Probe(ctx, req)
	result.RequestId = req.RequestId

	return result
}
