package inspection

import "github.com/ydcloud-dy/opshub/pkg/collector"

// AgentCommandFactory abstracts Agent capability so biz layer doesn't depend on server layer.
type AgentCommandFactory interface {
	// IsOnline checks whether the Agent on the given host is online.
	IsOnline(hostID uint) bool
	// NewExecutor creates a CommandExecutor that runs commands via the Agent on the given host.
	NewExecutor(hostID uint) (collector.CommandExecutor, error)
}
