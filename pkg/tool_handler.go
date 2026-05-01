package homeocto

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// -----------------------------------------------------------------------------
// Tool execution (home.ToolExecutor implementation)
// -----------------------------------------------------------------------------

// ExecuteTool implements home.ToolExecutor interface.
// It executes a tool by name with the given command JSON.
func (hc *HomeOcto) ExecuteTool(ctx context.Context, toolName, commandJSON string) (string, bool) {
	if hc == nil || hc.toolRegistry == nil {
		return "", true
	}

	// Get tool from registry
	tool, found := hc.toolRegistry.Get(toolName)
	if !found {
		logger.ErrorCF("homeocto", "Tool not found in registry", map[string]any{
			"tool_name":       toolName,
			"available_tools": hc.toolRegistry.List(),
		})
		return fmt.Sprintf("tool '%s' not registered", toolName), true
	}

	// Execute tool
	toolArgs := map[string]any{"commandJson": commandJSON}
	result := tool.Execute(ctx, toolArgs)

	if result.IsError {
		logger.ErrorCF("homeocto", "Tool execution failed", map[string]any{
			"tool_name": toolName,
			"error":     result.ForLLM,
		})
		return result.ForLLM, true
	}

	logger.InfoCF("homeocto", "Tool executed successfully", map[string]any{
		"tool_name":     toolName,
		"result_length": len(result.ForLLM),
	})

	return result.ForLLM, false
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
