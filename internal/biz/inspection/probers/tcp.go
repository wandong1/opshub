package probers

import (
	"fmt"
	"net"
	"time"
)

// TCPProber implements TCP connection probing.
type TCPProber struct{}

func (p *TCPProber) Probe(target string, port, timeout, count, packetSize int) *Result {
	addr := fmt.Sprintf("%s:%d", target, port)
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, time.Duration(timeout)*time.Second)
	connectTime := float64(time.Since(start).Microseconds()) / 1000.0

	if err != nil {
		return &Result{
			Latency:        connectTime,
			TCPConnectTime: connectTime,
			Error:          err.Error(),
		}
	}
	conn.Close()

	return &Result{
		Success:        true,
		Latency:        connectTime,
		TCPConnectTime: connectTime,
	}
}
