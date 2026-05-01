// Package homeocto provides the HomeOcto subsystem for intent recognition
// and workflow dispatching.  The HomeOcto type is the single entry point
// consumed by the agent loop.
package homeocto

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/home-ai-union/homeocto/pkg/channels/home"
	homeconfig "github.com/home-ai-union/homeocto/pkg/config"
	"github.com/home-ai-union/homeocto/pkg/ioc"
	thirdioc "github.com/home-ai-union/homeocto/pkg/third/ioc"
	"github.com/sipeed/picoclaw/pkg/config"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// ErrDisabled is returned by New when HomeOcto is explicitly disabled or
// homeocto.json is absent. Callers can use errors.Is(err, ErrDisabled) to
// distinguish a deliberate no-op from a real initialisation failure.
var ErrDisabled = ioc.ErrDisabled

// HomeOcto holds all HomeOcto subsystem objects and exposes a single
// RunIntent method that the agent loop calls from processMessage.
type HomeOcto struct {
	f      *ioc.Factory
	thirdf *thirdioc.ThirdFactory

	// WebSocket server for direct tool execution (HomeChannel)
	wsServer     *http.Server
	homeChannel  *home.HomeChannel
	toolRegistry *tools.ToolRegistry
}

// Factory returns the IOC factory. Nil-safe.
func (hc *HomeOcto) Factory() *ioc.Factory {
	if hc == nil {
		return nil
	}
	return hc.f
}

// ThirdFactory returns the third-party device factory. Nil-safe.
func (hc *HomeOcto) ThirdFactory() *thirdioc.ThirdFactory {
	if hc == nil {
		return nil
	}
	return hc.thirdf
}

// WSServer returns the WebSocket HTTP server. Nil-safe.
func (hc *HomeOcto) WSServer() *http.Server {
	if hc == nil {
		return nil
	}
	return hc.wsServer
}

// HomeChannel returns the HomeChannel instance. Nil-safe.
func (hc *HomeOcto) HomeChannel() *home.HomeChannel {
	if hc == nil {
		return nil
	}
	return hc.homeChannel
}

// ToolRegistry returns the tool registry. Nil-safe.
func (hc *HomeOcto) ToolRegistry() *tools.ToolRegistry {
	if hc == nil {
		return nil
	}
	return hc.toolRegistry
}

// NewHomeOcto creates a HomeOcto instance from the given workspace directory,
// PicoClaw config, and message bus.
// workspace is the data root used for all HomeOcto data files (users, devices, workflows, etc.).
// Returns nil (no error) when HomeOcto is disabled or homeocto.json is absent -
// the caller should treat nil as "not configured".
func NewHomeOcto(workspace string, cfg *config.Config, hcfg *homeconfig.HomeConfig) (*HomeOcto, error) {
	// Create factory which handles all singleton object creation
	factory, err := ioc.NewFactory(workspace, cfg, hcfg)
	if err != nil {
		if errors.Is(err, ErrDisabled) {
			return nil, ErrDisabled
		}
		return nil, fmt.Errorf("HomeOcto factory creation failed: %w", err)
	}
	thirdf := thirdioc.NewThirdFactory(factory)
	return &HomeOcto{
		f:            factory,
		thirdf:       thirdf,
		toolRegistry: factory.GetToolRegistry(),
	}, nil
}

// -----------------------------------------------------------------------------
// HomeOcto tool registration
// -----------------------------------------------------------------------------

// registerTool is a helper that calls a factory method, logs on error, and
// registers the resulting tool when successful.
func registerTool[T tools.Tool](toolRegistry *tools.ToolRegistry, create func() (T, error)) {
	t, err := create()
	if err != nil {
		logger.WarnCF("homeocto", "tool creation failed, skipping",
			map[string]any{"error": err.Error()})
		return
	}
	toolRegistry.Register(t)
}

// RegisterTools registers all HomeOcto tools (device, space, workflow)
// into the given tool registry.
// It is safe to call when hc is nil - the method becomes a no-op.
func (hc *HomeOcto) RegisterTools(toolRegistry *tools.ToolRegistry) {
	if hc == nil || toolRegistry == nil {
		return
	}

	f := hc.f

	// Video / RTSP tools
	registerTool(toolRegistry, f.GetVideoTool)

	// LLM tools
	registerTool(toolRegistry, f.GetLLMTool)

	// Common tools
	registerTool(toolRegistry, f.GetCommonTool)

	// CLI tool for device control
	registerTool(toolRegistry, f.GetCLITool)
}

// StartWSServer starts a standalone WebSocket server for direct tool execution.
// This is used when PicoClaw's channel injection is not available.
// The server listens on the port specified in the Homeocto config (ToolWSPort).
// Returns nil if the standalone server is disabled (ToolWSPort <= 0).
func (hc *HomeOcto) StartWSServer(toolRegistry *tools.ToolRegistry, picoConfig *config.Config) error {
	if hc == nil || hc.f == nil {
		return nil
	}

	// Check if standalone server is enabled
	hcfg := hc.f.GetHomeoctoConfig()
	if hcfg == nil || hcfg.Port <= 0 {
		logger.InfoCF("homeocto", "Standalone tool WS server disabled (port <= 0)", nil)
		return nil
	}

	if toolRegistry == nil {
		return fmt.Errorf("toolRegistry is required")
	}

	// Store tool registry for ExecuteTool
	hc.toolRegistry = toolRegistry

	// Merge configuration from picoclaw and homeocto
	chCfg := home.BuildHomeChannelConfig(picoConfig, hcfg)

	// Set HomeOcto as the tool executor
	home.SetToolExecutor(hc)

	// Create HomeChannel
	hc.homeChannel = home.NewHomeChannel(chCfg, hc)

	// Start the channel
	ctx := context.Background()
	if err := hc.homeChannel.Start(ctx); err != nil {
		return fmt.Errorf("failed to start HomeChannel: %w", err)
	}

	return nil
}

// StopToolWSServer stops the standalone WebSocket server if it's running.
func (hc *HomeOcto) StopWSServer() {
	if hc == nil || hc.homeChannel == nil {
		return
	}

	logger.InfoCF("homeocto", "Stopping HomeChannel WebSocket server", nil)

	// Stop the HomeChannel
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := hc.homeChannel.Stop(ctx); err != nil {
		logger.ErrorCF("homeocto", "Error stopping HomeChannel", map[string]any{
			"error": err.Error(),
		})
	}

	hc.homeChannel = nil
	hc.toolRegistry = nil
}
