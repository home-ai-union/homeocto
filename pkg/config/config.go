// Package config provides HomeOcto-specific configuration, loaded independently
// from PicoClaw's main config.json to avoid upstream coupling.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	picoconfig "github.com/sipeed/picoclaw/pkg/config"
)

const defaultConfigFileName = "home.json"

// Build-time variables injected via ldflags during build process.
// These are set by the Makefile or .goreleaser.yaml using the -X flag:
//
//	-X github.com/home-ai-union/homeocto/pkg/config.Version=<version>
//	-X github.com/home-ai-union/homeocto/pkg/config.GitCommit=<commit>
//	-X github.com/home-ai-union/homeocto/pkg/config.BuildTime=<timestamp>
//	-X github.com/home-ai-union/homeocto/pkg/config.GoVersion=<go-version>
var (
	// Version is the current version of HomeOcto.
	Version = "dev"
	// GitCommit is the Git commit SHA (short).
	GitCommit string
	// BuildTime is the build timestamp in RFC3339 format.
	BuildTime string
	// GoVersion is the Go version used for building.
	GoVersion string
)

// SyncVersionToPicoclaw synchronizes HomeOcto version info to PicoClaw's config package.
// This should be called during application initialization to ensure version commands
// display consistent information across both packages.
func SyncVersionToPicoclaw() {
	picoconfig.Version = Version
	picoconfig.GitCommit = GitCommit
	picoconfig.BuildTime = BuildTime
	picoconfig.GoVersion = GoVersion
}

// HomeConfig is the top-level HomeOcto configuration.
// It is stored in a standalone home.json file and loaded independently
// from PicoClaw's config.json so that upstream changes to PicoClaw do not
// affect HomeOcto configuration.
type HomeConfig struct {
	// Enabled controls whether HomeOcto intent processing is active.
	// When false (or home.json is absent), handleIntent is a no-op.
	Enabled bool `json:"enabled"`

	// IntentEnabled controls whether the intent classification and dispatching
	// logic (RunIntent) should be executed. When false, RunIntent will skip
	// processing and return immediately, falling through to the large model.
	IntentEnabled bool `json:"intent_enabled"`

	// SmallModel specifies the model_name (from PicoClaw's model_list) used for
	// intent classification and other small tasks. If empty, falls back to the
	// default model from PicoClaw config.
	SmallModel string `json:"small_model,omitempty"`

	// BigModel specifies the model_name (from PicoClaw's model_list) used for
	// complex tasks. If empty, falls back to the default model from PicoClaw config.
	BigModel string `json:"big_model,omitempty"`

	// ToolWSPort specifies the port for the standalone tool WebSocket server.
	// When set to 0 or negative, the standalone server is disabled and tools
	// can only be accessed through the PicoClaw channel injection method.
	// Default: 0 (disabled)
	Port int `json:"port,omitempty"`
}

// Load reads and parses a homeocto.json file from the given path.
func load(path string) (*HomeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("home config: read %s: %w", path, err)
	}

	var cfg HomeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("home config: parse %s: %w", path, err)
	}
	return &cfg, nil
}

// LoadFromFile reads a home.json file from the given absolute path.
// Returns os.IsNotExist error if the file is absent (no auto-creation).
func LoadFromFile(path string) (*HomeConfig, error) {
	return load(path)
}

// DefaultConfigFileName returns the default home config filename.
func DefaultConfigFileName() string {
	return defaultConfigFileName
}

// LoadFromDir looks for homeocto.json inside dir and loads it.
// If the file does not exist, it creates a default config and saves it.
func LoadHomeConfig() (*HomeConfig, error) {
	// Ensure the directory exists first
	homeDir := GetPicoclawHome()
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		return nil, fmt.Errorf("home config: create directory %s: %w", homeDir, err)
	}

	path := filepath.Join(homeDir, defaultConfigFileName)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		// Create default config and save it
		cfg := DefaultHomeConfig()
		if err := SaveHomeConfig(path, cfg); err != nil {
			return nil, fmt.Errorf("home config: create default %s: %w", path, err)
		}
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("home config: stat %s: %w", path, err)
	}
	return load(path)
}

// DefaultHomeConfig returns a default HomeConfig with sensible defaults.
func DefaultHomeConfig() *HomeConfig {
	return &HomeConfig{
		Enabled:       true,
		IntentEnabled: false,
		SmallModel:    "",
		BigModel:      "",
		Port:          18801,
	}
}

// SaveHomeConfig saves the HomeConfig to the specified path.
func SaveHomeConfig(path string, cfg *HomeConfig) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
