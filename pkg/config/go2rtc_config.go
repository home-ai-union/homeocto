package config

import (
	"errors"
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
func PatchConfig(filepath string, path []string, value any) error {
	if filepath == "" {
		return errors.New("config file disabled")
	}

	configMu.Lock()
	defer configMu.Unlock()

	// empty config is OK
	b, _ := os.ReadFile(filepath)

	b, err := yaml.Patch(b, path, value)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, b, 0644)
}

// GetGo2RTCPath returns the full path to the go2rtc.yaml config file.
func GetGo2RTCPath() string {
	return filepath.Join(GetPicoclawHome(), defaultGo2RtcFileName)
}

// LoadGo2RTCConfig loads the go2rtc.yaml configuration.
func LoadGo2RTCConfig(v any) error {
	return LoadYamlConfig(GetGo2RTCPath(), v)
}

// PatchGo2RTCConfig patches the go2rtc.yaml configuration.
func PatchGo2RTCConfig(path []string, value any) error {
	return PatchConfig(GetGo2RTCPath(), path, value)
}
