// Package homeocto provides the HomeOcto subsystem for intent recognition
// and workflow dispatching.  The HomeOcto type is the single entry point
// consumed by the agent loop.
package homeocto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/home-ai-union/homeocto/pkg/homeocto/intent"
	"github.com/home-ai-union/homeocto/pkg/homeocto/ioc"
	third "github.com/home-ai-union/homeocto/pkg/homeocto/third/ioc"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"

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
	thirdf *third.ThirdFactory

	// WebSocket server for direct tool execution
	wsServer  *http.Server
	wsHandler *ToolWSHandler
}

// InjectToolWSHandler creates a ToolWSHandler and injects it into PicoChannel
// so that /pico/ws-tool requests can execute tools directly without the agent loop.
func (hc *HomeOcto) InjectToolWSHandler(
	cm *channels.Manager,
	toolRegistry *tools.ToolRegistry,
	picoCfg *config.Config,
) {
	if cm == nil {
		logger.Warnf("[HomeOcto.InjectToolWSHandler] channelManager is nil, skipping")
		return
	}

	handler := NewToolWSHandler(hc, toolRegistry, picoCfg)
	if handler == nil {
		logger.Warnf("[HomeOcto.InjectToolWSHandler] NewToolWSHandler returned nil")
		return
	}

	// Get PicoChannel and inject the handler directly
	ch := cm.ChannelByName("pico")
	if ch == nil {
		logger.Warnf("[HomeOcto.InjectToolWSHandler] PicoChannel not found")
		return
	}

	if setter, ok := ch.(channels.ToolHandlerSetter); ok {
		setter.SetToolHandler(handler)
		logger.InfoCF("homeocto", "ToolWSHandler injected into PicoChannel", nil)
	} else {
		logger.Warnf("[HomeOcto.InjectToolWSHandler] PicoChannel does not implement ToolHandlerSetter")
	}
}

// NewHomeOcto creates a HomeOcto instance from the given workspace directory,
// PicoClaw config, and message bus.
// workspace is the data root used for all HomeOcto data files (users, devices, workflows, etc.).
// Returns nil (no error) when HomeOcto is disabled or homeocto.json is absent -
// the caller should treat nil as "not configured".
func NewHomeOcto(workspace string, picolawerCfg *config.Config, msgBus *bus.MessageBus) (*HomeOcto, error) {
	// Create factory which handles all singleton object creation
	factory, err := ioc.NewFactory(workspace, picolawerCfg, msgBus)
	if err != nil {
		if errors.Is(err, ErrDisabled) {
			return nil, ErrDisabled
		}
		return nil, fmt.Errorf("HomeOcto factory creation failed: %w", err)
	}
	thirdf := third.NewThirdFactory(factory)
	return &HomeOcto{
		f:      factory,
		thirdf: thirdf,
	}, nil
}

// RunIntentInput contains all inputs needed for intent recognition.
type RunIntentInput struct {
	UserInput  string
	Channel    string
	ChatID     string
	SenderID   string
	SessionKey string
}

// RunIntent performs intent classification and dispatching.
//
// Return semantics:
//   - (response, true,  false, nil) - fully handled by small model; send response to user.
//   - (context,  true,  true,  nil) - small model handled and produced context; forward
//     context to large model for further reasoning.
//   - ("",       false, false, nil) - not handled; fall through to large model with original input.
//
// A non-nil error is always accompanied by handled=false so the caller can fall
// through safely.
func (hc *HomeOcto) RunIntent(ctx context.Context, in RunIntentInput) (response string, handled bool, forwardToLLM bool, err error) {
	if hc == nil {
		return "", false, false, nil
	}

	// Skip intent processing if IntentEnabled is false
	hcfg := hc.f.GetHomeoctoConfig()
	if hcfg != nil && !hcfg.IntentEnabled {
		return "", false, false, nil
	}

	classifier, classifierErr := hc.f.GetIntentClassifier()
	if classifierErr != nil {
		return "", false, false, fmt.Errorf("intent classifier unavailable: %w", classifierErr)
	}

	result, classErr := classifier.Classify(ctx, in.UserInput)
	if classErr != nil {
		logger.WarnCF("homeocto", "intent classification error, falling through",
			map[string]any{"error": classErr.Error()})
	}
	if result.Type == intent.IntentUnknown {
		return "", false, false, nil
	}

	router, routerErr := hc.f.GetIntentRouter()
	if routerErr != nil {
		return "", false, false, fmt.Errorf("intent router unavailable: %w", routerErr)
	}

	handler, ok := router.Route(result)
	if !ok {
		return "", false, false, nil
	}

	ictx := intent.IntentContext{
		UserInput:  in.UserInput,
		Channel:    in.Channel,
		ChatID:     in.ChatID,
		SenderID:   in.SenderID,
		SessionKey: in.SessionKey,
		Result:     result,
		Workspace:  hc.f.Workspace,
	}

	resp := handler.Run(ctx, ictx)
	if resp.Error != nil {
		logger.ErrorCF("homeocto", "intent handler error",
			map[string]any{
				"intent": string(result.Type),
				"error":  resp.Error.Error(),
			})
	}
	return resp.Response, resp.Handled, resp.ForwardToLLM, resp.Error
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

	// Workflow tools
	registerTool(toolRegistry, f.GetListWorkflowsTool)
	registerTool(toolRegistry, f.GetGetWorkflowTool)
	registerTool(toolRegistry, f.GetSaveWorkflowTool)
	registerTool(toolRegistry, f.GetDeleteWorkflowTool)
	registerTool(toolRegistry, f.GetEnableWorkflowTool)
	registerTool(toolRegistry, f.GetDisableWorkflowTool)

	// Video / RTSP tools
	registerTool(toolRegistry, f.GetVideoTool)

	// LLM tools
	registerTool(toolRegistry, f.GetLLMTool)

	// Common tools
	registerTool(toolRegistry, f.GetCommonTool)

	// CLI tool for device control
	registerTool(toolRegistry, f.GetCLITool)
}

// SetMediaStore sets the media store for HomeOcto tools that need to send images to channels.
func (hc *HomeOcto) SetMediaStore(store media.MediaStore) {
	if hc == nil || hc.f == nil {
		return
	}
	hc.f.SetMediaStore(store)
}

// SetClients initializes and registers all third-party brand clients (Xiaomi, Tuya, etc.)
// and injects them into the CLI and LLM tools.
func (hc *HomeOcto) SetClients() error {
	if hc == nil || hc.thirdf == nil {
		return nil
	}

	// Set the refresh callback on CLITool before initializing clients
	if cliTool, err := hc.f.GetCLITool(); err == nil && cliTool != nil {
		cliTool.SetRefreshClients(func() error {
			return hc.thirdf.SetClients()
		})
	}

	return hc.thirdf.SetClients()
}

// StartToolWSServer starts a standalone WebSocket server for direct tool execution.
// This is used when PicoClaw's channel injection is not available.
// The server listens on the port specified in the Homeocto config (ToolWSPort).
// Returns nil if the standalone server is disabled (ToolWSPort <= 0).
func (hc *HomeOcto) StartToolWSServer(toolRegistry *tools.ToolRegistry, picoConfig *config.Config) error {
	if hc == nil || hc.f == nil {
		return nil
	}

	// Check if standalone server is enabled
	hcfg := hc.f.GetHomeoctoConfig()
	if hcfg == nil || hcfg.ToolWSPort <= 0 {
		logger.InfoCF("homeocto", "Standalone tool WS server disabled (tool_ws_port <= 0)", nil)
		return nil
	}

	if toolRegistry == nil {
		return fmt.Errorf("toolRegistry is required")
	}

	// Create the WebSocket handler
	hc.wsHandler = NewToolWSHandler(hc, toolRegistry, picoConfig)
	if hc.wsHandler == nil {
		return fmt.Errorf("failed to create ToolWSHandler")
	}

	// Create HTTP mux and register the WebSocket endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/ws-tool", hc.wsHandler.HandleToolWebSocket)

	// Create the HTTP server
	addr := fmt.Sprintf(":%d", hcfg.ToolWSPort)
	hc.wsServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start the server in a goroutine
	go func() {
		logger.InfoCF("homeocto", "Starting standalone tool WS server", map[string]any{
			"addr": addr,
		})
		if err := hc.wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("homeocto", "Standalone tool WS server error", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

// StopToolWSServer stops the standalone WebSocket server if it's running.
func (hc *HomeOcto) StopToolWSServer() {
	if hc == nil || hc.wsServer == nil {
		return
	}

	logger.InfoCF("homeocto", "Stopping standalone tool WS server", nil)

	// Shutdown the HTTP server with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := hc.wsServer.Shutdown(ctx); err != nil {
		logger.ErrorCF("homeocto", "Error shutting down tool WS server", map[string]any{
			"error": err.Error(),
		})
	}

	// Close the handler resources
	if hc.wsHandler != nil {
		hc.wsHandler.Close()
	}

	hc.wsServer = nil
	hc.wsHandler = nil
}

// NewToolWSHandler creates a ToolWSHandler for direct tool WebSocket execution.
// Returns nil if HomeOcto is nil or not fully configured.
func (hc *HomeOcto) NewToolWSHandler(toolRegistry *tools.ToolRegistry, picoConfig *config.Config) ToolCallHandler {
	if hc == nil || toolRegistry == nil {
		return nil
	}
	return NewToolWSHandler(hc, toolRegistry, picoConfig)
}

// -----------------------------------------------------------------------------
// Device command handling via hc_cli tool
// -----------------------------------------------------------------------------

// HandleToolCall checks if the message is a tool command (format: "tool:name" + JSON params)
// and executes it via the specified tool directly, bypassing the LLM.
// Returns (response, handled) where handled=true means the command was processed.
func (hc *HomeOcto) HandleToolCall(ctx context.Context, channel, chatID, content string, toolRegistry *tools.ToolRegistry) (string, bool) {
	if hc == nil || toolRegistry == nil {
		return "", false
	}

	// Parse tool name and command JSON from content
	toolName, commandJSON, ok := hc.ParseToolCommand(content)
	if !ok {
		return "", false
	}

	logger.InfoCF("homeocto", "Tool command detected, executing via tool",
		map[string]any{
			"channel":   channel,
			"chat_id":   chatID,
			"tool_name": toolName,
		})

	// Get the specified tool from registry
	tool, ok := toolRegistry.Get(toolName)
	if !ok {
		logger.ErrorCF("homeocto", "Tool not found",
			map[string]any{
				"tool_name":       toolName,
				"available_tools": toolRegistry.List(),
			})
		return fmt.Sprintf("Tool execution failed: tool '%s' not registered", toolName), true
	}

	// Execute the tool with the command JSON
	toolArgs := map[string]any{
		"commandJson": commandJSON,
	}

	logger.DebugCF("homeocto", "Executing tool",
		map[string]any{
			"tool_name":    toolName,
			"command_json": commandJSON,
		})

	result := tool.Execute(ctx, toolArgs)

	if result.IsError {
		logger.ErrorCF("homeocto", "Tool execution failed",
			map[string]any{
				"tool_name": toolName,
				"error":     result.ForLLM,
			})
		return fmt.Sprintf("Tool execution failed: %s", result.ForLLM), true
	}

	logger.InfoCF("homeocto", "Tool executed successfully",
		map[string]any{
			"tool_name":     toolName,
			"result_length": len(result.ForLLM),
		})

	return result.ForLLM, true
}

// ParseToolCommand parses the content to extract tool name and command JSON.
// Expected format: "tool:toolName {json_params}"
// Returns (toolName, commandJSON, success)
func (hc *HomeOcto) ParseToolCommand(content string) (string, string, bool) {
	content = strings.TrimSpace(content)

	// Check if content starts with "tool:"
	if !strings.HasPrefix(content, "tool:") {
		return "", "", false
	}

	// Remove "tool:" prefix
	content = content[5:]

	// Find the first space to separate tool name from JSON
	spaceIdx := strings.Index(content, " ")
	if spaceIdx == -1 {
		return "", "", false
	}

	toolName := strings.TrimSpace(content[:spaceIdx])
	if toolName == "" {
		return "", "", false
	}

	// Extract JSON part
	commandJSON := strings.TrimSpace(content[spaceIdx+1:])
	if commandJSON == "" {
		return "", "", false
	}

	// Validate JSON format
	var cmd map[string]interface{}
	if err := json.Unmarshal([]byte(commandJSON), &cmd); err != nil {
		return "", "", false
	}

	return toolName, commandJSON, true
}
