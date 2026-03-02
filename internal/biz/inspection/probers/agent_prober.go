package probers

import (
	"fmt"
	"strings"
	"time"

	"github.com/ydcloud-dy/opshub/pkg/collector"
)

// AgentPingProber executes ping via an Agent and parses the output.
type AgentPingProber struct {
	Executor collector.CommandExecutor
}

func (p *AgentPingProber) Probe(target string, port, timeout, count, packetSize int) *Result {
	if count <= 0 {
		count = 4
	}
	if timeout <= 0 {
		timeout = 5
	}

	cmd := fmt.Sprintf("ping -c %d -W %d -s %d %s", count, timeout, packetSize, target)
	start := time.Now()
	output, err := p.Executor.Execute(cmd)
	elapsed := float64(time.Since(start).Microseconds()) / 1000.0

	result := &Result{Latency: elapsed}
	ParsePingOutput(output, result)

	if result.PingPacketsSent == 0 && err != nil {
		result.Success = false
		errMsg := strings.TrimSpace(output)
		if errMsg == "" {
			errMsg = fmt.Sprintf("ping failed: %v", err)
		}
		result.Error = errMsg
	}

	return result
}

// AgentTCPProber executes TCP probe via an Agent using bash /dev/tcp.
type AgentTCPProber struct {
	Executor collector.CommandExecutor
}

func (p *AgentTCPProber) Probe(target string, port, timeout, count, packetSize int) *Result {
	if timeout <= 0 {
		timeout = 5
	}
	cmd := fmt.Sprintf("timeout %d bash -c 'cat < /dev/null > /dev/tcp/%s/%d' 2>&1; echo \"EXIT:$?\"", timeout, target, port)
	start := time.Now()
	output, _ := p.Executor.Execute(cmd)
	connectTime := float64(time.Since(start).Microseconds()) / 1000.0

	result := &Result{
		Latency:        connectTime,
		TCPConnectTime: connectTime,
	}

	// Parse exit code from output
	if strings.Contains(output, "EXIT:0") {
		result.Success = true
	} else {
		result.Success = false
		result.Error = strings.TrimSpace(strings.Replace(output, "EXIT:1", "", 1))
		if result.Error == "" {
			result.Error = fmt.Sprintf("TCP connect to %s:%d failed", target, port)
		}
	}

	return result
}

// AgentUDPProber executes UDP probe via an Agent using nc.
type AgentUDPProber struct {
	Executor collector.CommandExecutor
}

func (p *AgentUDPProber) Probe(target string, port, timeout, count, packetSize int) *Result {
	if timeout <= 0 {
		timeout = 5
	}
	cmd := fmt.Sprintf("nc -u -z -w %d %s %d 2>&1; echo \"EXIT:$?\"", timeout, target, port)
	start := time.Now()
	output, _ := p.Executor.Execute(cmd)
	elapsed := float64(time.Since(start).Microseconds()) / 1000.0

	result := &Result{
		Latency:      elapsed,
		UDPWriteTime: elapsed,
	}

	if strings.Contains(output, "EXIT:0") {
		result.Success = true
	} else {
		result.Success = false
		result.Error = strings.TrimSpace(strings.Replace(output, "EXIT:1", "", 1))
		if result.Error == "" {
			result.Error = fmt.Sprintf("UDP probe to %s:%d failed", target, port)
		}
	}

	return result
}

// GetAgentProber returns the appropriate agent-based Prober for the given type.
func GetAgentProber(probeType string, executor collector.CommandExecutor) (Prober, error) {
	switch probeType {
	case "ping":
		return &AgentPingProber{Executor: executor}, nil
	case "tcp":
		return &AgentTCPProber{Executor: executor}, nil
	case "udp":
		return &AgentUDPProber{Executor: executor}, nil
	default:
		return nil, fmt.Errorf("unknown probe type for agent: %s", probeType)
	}
}
