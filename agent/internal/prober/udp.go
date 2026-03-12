package prober

import (
	"context"
	"fmt"
	"net"
	"time"

	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
)

// UDPProber UDP 拨测器
type UDPProber struct{}

func (p *UDPProber) Probe(ctx context.Context, req *pb.ProbeRequest) *pb.ProbeResult {
	result := &pb.ProbeResult{}
	start := time.Now()

	addr := fmt.Sprintf("%s:%d", req.Target, req.Port)

	// UDP 连接
	conn, err := net.Dial("udp", addr)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("UDP dial failed: %v", err)
		result.Latency = float64(time.Since(start).Milliseconds())
		return result
	}
	defer conn.Close()

	// 发送测试数据
	writeStart := time.Now()
	_, err = conn.Write([]byte("ping"))
	writeTime := float64(time.Since(writeStart).Milliseconds())
	result.UdpWriteTime = writeTime

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("UDP write failed: %v", err)
		result.Latency = float64(time.Since(start).Milliseconds())
		return result
	}

	// 尝试读取响应（可能超时）
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1024)
	readStart := time.Now()
	_, err = conn.Read(buf)
	readTime := float64(time.Since(readStart).Milliseconds())
	result.UdpReadTime = readTime

	result.Latency = float64(time.Since(start).Milliseconds())
	result.Success = true // UDP 写入成功即认为成功

	return result
}
