package inspection

import (
	"github.com/ydcloud-dy/opshub/pkg/collector"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
)

// AgentCommandFactory abstracts Agent capability so biz layer doesn't depend on server layer.
type AgentCommandFactory interface {
	// IsOnline checks whether the Agent on the given host is online.
	IsOnline(hostID uint) bool
	// NewExecutor creates a CommandExecutor that runs commands via the Agent on the given host.
	NewExecutor(hostID uint) (collector.CommandExecutor, error)
	// SendProbeRequest sends a probe request to the Agent and waits for the result.
	SendProbeRequest(hostID uint, req *pb.ProbeRequest) (*pb.ProbeResult, error)
	// SendWsSessionOpen opens a WebSocket session on the Agent.
	SendWsSessionOpen(hostID uint, sessionID, url string, headers, params map[string]string, timeout int32, skipVerify bool, proxyURL string) (*pb.WsSessionResult, error)
	// SendWsSessionAction sends a WebSocket action (send/receive) to the Agent.
	SendWsSessionAction(hostID uint, sessionID, actionID, actionType, message string, messageType, readTimeout int32, receiveMode string) (*pb.WsSessionResult, error)
	// SendWsSessionClose closes a WebSocket session on the Agent.
	SendWsSessionClose(hostID uint, sessionID string) (*pb.WsSessionResult, error)
}
