package probers

import (
	"fmt"
	"net"
	"time"
)

// UDPProber implements UDP probing.
type UDPProber struct{}

func (p *UDPProber) Probe(target string, port, timeout, count, packetSize int) *Result {
	addr := fmt.Sprintf("%s:%d", target, port)
	deadline := time.Duration(timeout) * time.Second

	// Use a single deadline for the entire operation (dial + write + read).
	start := time.Now()
	conn, err := net.DialTimeout("udp", addr, deadline)
	if err != nil {
		return &Result{Error: err.Error()}
	}
	defer conn.Close()

	// Set absolute deadline so total time never exceeds timeout.
	conn.SetDeadline(start.Add(deadline))

	payload := make([]byte, packetSize)

	writeStart := time.Now()
	_, err = conn.Write(payload)
	writeTime := float64(time.Since(writeStart).Microseconds()) / 1000.0
	if err != nil {
		return &Result{
			Latency:      writeTime,
			UDPWriteTime: writeTime,
			Error:        err.Error(),
		}
	}

	buf := make([]byte, 1024)
	readStart := time.Now()
	_, readErr := conn.Read(buf)
	readTime := float64(time.Since(readStart).Microseconds()) / 1000.0

	totalLatency := float64(time.Since(start).Microseconds()) / 1000.0
	result := &Result{
		Latency:      totalLatency,
		UDPWriteTime: writeTime,
		UDPReadTime:  readTime,
	}

	if readErr != nil {
		if netErr, ok := readErr.(net.Error); ok && netErr.Timeout() {
			// Read timed out — mark as failure since the target didn't respond within timeout.
			result.Success = false
			result.Error = "read timeout: no response within deadline"
		} else {
			result.Success = false
			result.Error = readErr.Error()
		}
	} else {
		result.Success = true
	}

	return result
}
