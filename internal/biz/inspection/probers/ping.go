package probers

import (
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PingProber implements ICMP ping by invoking the system ping command.
// This avoids raw socket permission issues (no root or sysctl needed).
type PingProber struct{}

var (
	// "4 packets transmitted, 3 received, 25% packet loss, time 3003ms"
	statsRe = regexp.MustCompile(`(\d+)\s+packets?\s+transmitted,\s+(\d+)\s+received.*?(\d+(?:\.\d+)?)%\s+packet\s+loss`)
	// "rtt min/avg/max/mdev = 0.123/0.456/0.789/0.012 ms"
	rttRe = regexp.MustCompile(`(?:rtt|round-trip)\s+min/avg/max/(?:mdev|stddev)\s*=\s*([\d.]+)/([\d.]+)/([\d.]+)/([\d.]+)\s*ms`)
)

func (p *PingProber) Probe(target string, port, timeout, count, packetSize int) *Result {
	if count <= 0 {
		count = 4
	}
	if timeout <= 0 {
		timeout = 5
	}

	// Build ping command: -c count, -W timeout (per-packet wait), -s packetSize
	args := []string{
		"-c", strconv.Itoa(count),
		"-W", strconv.Itoa(timeout),
		"-s", strconv.Itoa(packetSize),
		target,
	}

	start := time.Now()
	cmd := exec.Command("ping", args...)
	output, err := cmd.CombinedOutput()
	elapsed := float64(time.Since(start).Microseconds()) / 1000.0

	result := &Result{Latency: elapsed}
	ParsePingOutput(string(output), result)

	// If parsing found no stats and command failed, set error
	if result.PingPacketsSent == 0 && err != nil {
		result.Success = false
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			errMsg = fmt.Sprintf("ping failed: %v", err)
		}
		result.Error = errMsg
	}

	return result
}

// ParsePingOutput parses standard ping command output into a Result.
// Exported so it can be reused by AgentPingProber.
func ParsePingOutput(output string, result *Result) {
	// Parse packet stats
	if m := statsRe.FindStringSubmatch(output); len(m) == 4 {
		sent, _ := strconv.Atoi(m[1])
		recv, _ := strconv.Atoi(m[2])
		lossPercent, _ := strconv.ParseFloat(m[3], 64)

		result.PingPacketsSent = sent
		result.PingPacketsRecv = recv
		result.PacketLoss = lossPercent / 100.0
		result.Success = recv > 0
	}

	// Parse RTT min/avg/max/mdev
	if m := rttRe.FindStringSubmatch(output); len(m) == 5 {
		result.PingRttMin, _ = strconv.ParseFloat(m[1], 64)
		result.PingRttAvg, _ = strconv.ParseFloat(m[2], 64)
		result.PingRttMax, _ = strconv.ParseFloat(m[3], 64)
		result.PingStddev, _ = strconv.ParseFloat(m[4], 64)
	}

	// If 100% loss, mark as failure
	if result.PingPacketsSent > 0 && result.PingPacketsRecv == 0 {
		result.Success = false
		if result.Error == "" {
			result.Error = "100% packet loss"
		}
	}

	// Sanitize NaN/Inf
	if math.IsNaN(result.PingStddev) || math.IsInf(result.PingStddev, 0) {
		result.PingStddev = 0
	}
}
