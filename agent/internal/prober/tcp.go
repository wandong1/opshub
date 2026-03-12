package prober

import (
	"context"
	"fmt"
	"net"
	"time"

	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
)

// TCPProber TCP 拨测器
type TCPProber struct{}

func (p *TCPProber) Probe(ctx context.Context, req *pb.ProbeRequest) *pb.ProbeResult {
	result := &pb.ProbeResult{}
	start := time.Now()

	addr := fmt.Sprintf("%s:%d", req.Target, req.Port)

	// 创建 dialer
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)

	connectTime := float64(time.Since(start).Milliseconds())
	result.Latency = connectTime
	result.TcpConnectTime = connectTime

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("TCP connect failed: %v", err)
		return result
	}

	conn.Close()
	result.Success = true

	return result
}
