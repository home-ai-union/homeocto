package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/AlexxIT/go2rtc/pkg/yaml"
)

const defaultGo2RtcFileName = "go2rtc.yaml"

var configMu sync.Mutex

// LoadYamlConfig loads and parses a YAML configuration file.
func LoadYamlConfig(filepath string, v any) error {
	if filepath == "" {
		return errors.New("config file path is empty")
	}

	b, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(b, v)
}

// PatchConfig patches a YAML configuration file at the given path.
// If the file does not exist, it creates a new one with the patched value.
func PatchConfig(fpath string, path []string, value any) error {
	if fpath == "" {
		return errors.New("config file disabled")
	}

	configMu.Lock()
	defer configMu.Unlock()

	// empty config is OK - if file doesn't exist, start with empty bytes
	b, _ := os.ReadFile(fpath)

	b, err := yaml.Patch(b, path, value)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(fpath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return os.WriteFile(fpath, b, 0o644)
}

// GetGo2RTCPath returns the full path to the go2rtc.yaml config file.
func GetGo2RTCPath() string {
	return filepath.Join(GetPicoclawHome(), defaultGo2RtcFileName)
}

// LoadGo2RTCConfig loads the go2rtc.yaml configuration.
// If the file does not exist, it returns nil without error (empty config).
func LoadGo2RTCConfig(v any) error {
	path := GetGo2RTCPath()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, return nil without error
		// The v parameter will remain at its zero value
		return nil
	}

	return LoadYamlConfig(path, v)
}

// PatchGo2RTCConfig patches the go2rtc.yaml configuration.
func PatchGo2RTCConfig(path []string, value any) error {
	return PatchConfig(GetGo2RTCPath(), path, value)
}
