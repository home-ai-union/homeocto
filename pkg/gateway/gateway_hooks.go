//go:build !nohomeocto

package gateway

import (
	"errors"
	"fmt"

	hcpkg "github.com/home-ai-union/homeocto/pkg"
	hcconfig "github.com/home-ai-union/homeocto/pkg/config"

	"github.com/sipeed/picoclaw/pkg/agent"
	picoconfig "github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// homeoctoContext stores HomeOcto-related state in services.extData
type homeoctoContext struct {
	homeOcto *hcpkg.HomeOcto
}

// homeoctoCtx holds the initialized HomeOcto context between init and service setup.
// runHomeOctoInit creates and stores it here; it is later placed into services.extData
// after setupAndStartServices creates the services instance.
var homeoctoCtx *homeoctoContext

// runHomeOctoInit initializes HomeOcto, creates tool registry, registers tools
// to agentLoop, starts WebSocket server, and initializes third-party clients.
func runHomeOctoInit(agentLoop *agent.AgentLoop) error {
	cfg, err := hcconfig.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	// Load home.json config
	hcfg, err := hcconfig.LoadHomeConfig()
	if err != nil {
		return fmt.Errorf("load home config: %w", err)
	}

	if !hcfg.Enabled {
		logger.InfoCF("homeocto", "HomeOcto disabled in config", nil)
		return nil
	}

	// Create HomeOcto instance
	ho, err := hcpkg.NewHomeOcto(hcconfig.GetWorkspace(), cfg, hcfg)
	if err != nil {
		if errors.Is(err, hcpkg.ErrDisabled) {
			logger.InfoCF("homeocto", "HomeOcto returned disabled", nil)
			return nil
		}
		return fmt.Errorf("create homeocto: %w", err)
	}

	// Initialize third-party device clients (Xiaomi, Tuya) FIRST
	// This must happen before tool registration so that tools have clients available
	thirdf := ho.ThirdFactory()
	if thirdf != nil {
		clients, err := thirdf.GetClients()
		if err != nil {
			logger.WarnCF("homeocto", "Failed to init third-party clients", map[string]any{
				"error": err.Error(),
			})
		} else if clients != nil {
			logger.InfoCF("homeocto", "Third-party clients initialized", map[string]any{
				"brands": clients.ListBrands(),
			})
			ho.Factory().SetClients(clients)
		}
	}

	// Create tool registry and register homeocto tools
	// Tools will have clients already injected via registerClientsSetter
	toolRegistry := ho.ToolRegistry()
	ho.RegisterTools(toolRegistry)

	logger.InfoCF("homeocto", "HomeOcto tools registered", map[string]any{
		"tools": toolRegistry.List(),
	})

	// Register homeocto tools to agentLoop (broadcasts to ALL agents)
	for _, toolName := range toolRegistry.List() {
		tool, ok := toolRegistry.Get(toolName)
		if !ok {
			continue
		}
		agentLoop.RegisterTool(tool)
		logger.InfoCF("homeocto", "Tool registered to agentLoop", map[string]any{
			"tool_name": toolName,
		})
	}

	// Start standalone WebSocket server
	if err := ho.StartWSServer(toolRegistry, cfg); err != nil {
		logger.ErrorCF("homeocto", "Failed to start HomeOcto WebSocket server", map[string]any{
			"error": err.Error(),
		})
		// Non-fatal: continue without WS server
	} else {
		logger.InfoCF("homeocto", "HomeOcto WebSocket server started", map[string]any{
			"port": hcfg.Port,
		})
	}

	// Store context for later placement into services.extData
	homeoctoCtx = &homeoctoContext{
		homeOcto: ho,
	}

	fmt.Printf("  • HomeOcto: enabled (port=%d, tools=%d)\n", hcfg.Port, len(toolRegistry.List()))

	return nil
}

// stopHomeOcto stops the HomeOcto WebSocket server during shutdown.
func stopHomeOcto(s *services) {
	if s == nil || s.extData == nil {
		return
	}

	ctx, ok := s.extData["homeocto"].(*homeoctoContext)
	if !ok || ctx == nil || ctx.homeOcto == nil {
		return
	}

	logger.InfoCF("homeocto", "Stopping HomeOcto WebSocket server", nil)
	ctx.homeOcto.StopWSServer()
}

// reinitHomeOcto re-registers HomeOcto tools after a config reload.
// This is necessary because ReloadProviderAndConfig creates a new AgentRegistry,
// losing all previously registered tools.
func reinitHomeOcto(al *agent.AgentLoop, s *services, newCfg *picoconfig.Config) {
	if s == nil || s.extData == nil {
		return
	}

	ctx, ok := s.extData["homeocto"].(*homeoctoContext)
	if !ok || ctx == nil || ctx.homeOcto == nil {
		logger.InfoCF("homeocto", "HomeOcto not initialized, skipping re-init", nil)
		return
	}

	logger.InfoCF("homeocto", "Re-initializing HomeOcto tools after config reload", nil)
	// Create tool registry and register homeocto tools
	toolRegistry := ctx.homeOcto.ToolRegistry()

	logger.InfoCF("homeocto", "HomeOcto tools registered", map[string]any{
		"tools": toolRegistry.List(),
	})

	// Register homeocto tools to agentLoop (broadcasts to ALL agents)
	for _, toolName := range toolRegistry.List() {
		tool, ok := toolRegistry.Get(toolName)
		if !ok {
			continue
		}
		al.RegisterTool(tool)
		logger.InfoCF("homeocto", "Tool registered to agentLoop", map[string]any{
			"tool_name": toolName,
		})
	}

}
