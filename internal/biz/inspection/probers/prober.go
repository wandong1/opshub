package probers

import "fmt"

// Result holds the outcome of a probe execution.
type Result struct {
	Success         bool
	Latency         float64 // total ms
	PacketLoss      float64 // 0~1 (ping)
	PingRttAvg      float64 // ms
	PingRttMin      float64
	PingRttMax      float64
	PingStddev      float64
	PingPacketsSent int
	PingPacketsRecv int
	TCPConnectTime  float64 // ms
	UDPWriteTime    float64 // ms
	UDPReadTime     float64 // ms
	Error           string
}

// Prober defines the interface for network probing.
type Prober interface {
	Probe(target string, port, timeout, count, packetSize int) *Result
}

// GetProber returns the appropriate Prober for the given type.
func GetProber(probeType string) (Prober, error) {
	switch probeType {
	case "ping":
		return &PingProber{}, nil
	case "tcp":
		return &TCPProber{}, nil
	case "udp":
		return &UDPProber{}, nil
	default:
		return nil, fmt.Errorf("unknown probe type: %s", probeType)
	}
}
