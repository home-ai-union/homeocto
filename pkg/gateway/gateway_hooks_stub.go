//go:build nohomeocto

package gateway

import (
	"github.com/sipeed/picoclaw/pkg/agent"
	picoconfig "github.com/sipeed/picoclaw/pkg/config"
)

// runHomeOctoInit is a no-op when HomeOcto is disabled.
func runHomeOctoInit(homePath string, cfg *picoconfig.Config, agentLoop *agent.AgentLoop) error {
	return nil
}

// stopHomeOcto is a no-op when HomeOcto is disabled.
func stopHomeOcto(s *services) {
}

// reinitHomeOcto is a no-op when HomeOcto is disabled.
func reinitHomeOcto(al *agent.AgentLoop, s *services, newCfg *picoconfig.Config) {
}
