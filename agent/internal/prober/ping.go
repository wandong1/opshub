package prober

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
)

// PingProber Ping 拨测器
type PingProber struct{}

func (p *PingProber) Probe(ctx context.Context, req *pb.ProbeRequest) *pb.ProbeResult {
	result := &pb.ProbeResult{}
	start := time.Now()

	count := req.Count
	if count <= 0 {
		count = 4
	}

	packetSize := req.PacketSize
	if packetSize <= 0 {
		packetSize = 56
	}

	// 构建 ping 命令
	args := []string{"-c", fmt.Sprintf("%d", count), "-s", fmt.Sprintf("%d", packetSize)}
	if req.Timeout > 0 {
		args = append(args, "-W", fmt.Sprintf("%d", req.Timeout))
	}
	args = append(args, req.Target)

	cmd := exec.CommandContext(ctx, "ping", args...)
	output, err := cmd.CombinedOutput()

	result.Latency = float64(time.Since(start).Milliseconds())

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("ping failed: %v", err)
		return result
	}

	// 解析 ping 输出
	p.parsePingOutput(string(output), result)

	return result
}

func (p *PingProber) parsePingOutput(output string, result *pb.ProbeResult) {
	// 解析统计信息: "4 packets transmitted, 4 received, 0% packet loss"
	statsRe := regexp.MustCompile(`(\d+) packets transmitted, (\d+) received, ([\d.]+)% packet loss`)
	if matches := statsRe.FindStringSubmatch(output); len(matches) == 4 {
		sent, _ := strconv.Atoi(matches[1])
		recv, _ := strconv.Atoi(matches[2])
		loss, _ := strconv.ParseFloat(matches[3], 64)

		result.PingPacketsSent = int32(sent)
		result.PingPacketsRecv = int32(recv)
		result.PacketLoss = loss / 100.0
	}

	// 解析 RTT: "rtt min/avg/max/mdev = 0.123/0.456/0.789/0.012 ms"
	rttRe := regexp.MustCompile(`rtt min/avg/max/(?:mdev|stddev) = ([\d.]+)/([\d.]+)/([\d.]+)/([\d.]+) ms`)
	if matches := rttRe.FindStringSubmatch(output); len(matches) == 5 {
		min, _ := strconv.ParseFloat(matches[1], 64)
		avg, _ := strconv.ParseFloat(matches[2], 64)
		max, _ := strconv.ParseFloat(matches[3], 64)
		stddev, _ := strconv.ParseFloat(matches[4], 64)

		result.PingRttMin = min
		result.PingRttAvg = avg
		result.PingRttMax = max
		result.PingStddev = stddev
	}

	// 判断成功
	result.Success = result.PingPacketsRecv > 0
	if !result.Success && result.Error == "" {
		result.Error = "no packets received"
	}
}
