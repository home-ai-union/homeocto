package home

import "context"

// ToolExecutor defines the interface for tool parsing and execution.
// HomeChannel uses this to decouple from the HomeOcto type.
type ToolExecutor interface {
	// ParseToolCommand extracts tool name and command JSON from content.
	// Expected format: "tool:toolName {json_params}"
	// Returns (toolName, commandJSON, success)
	ParseToolCommand(content string) (toolName, commandJSON string, ok bool)

	// ExecuteTool runs a tool by name with the given command JSON.
	// Returns (result, isError)
	ExecuteTool(ctx context.Context, toolName, commandJSON string) (result string, isError bool)
}

var globalExecutor ToolExecutor

// SetToolExecutor sets the global tool executor instance.
func SetToolExecutor(e ToolExecutor) {
	globalExecutor = e
}

// GetToolExecutor returns the global tool executor instance.
func GetToolExecutor() ToolExecutor {
	return globalExecutor
}
