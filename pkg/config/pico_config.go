package config

import (
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// GetPicoclawHome returns the picoclaw home directory.
// Priority: $PICOCLAW_HOME > ~/.picoclaw
func GetPicoclawHome() string {
	return config.GetHome()
}

func GetConfigPath() string {
	if configPath := os.Getenv(config.EnvConfig); configPath != "" {
		return configPath
	}
	return filepath.Join(GetPicoclawHome(), "config.json")
}

func LoadConfig() (*config.Config, error) {
	cfg, err := config.LoadConfig(GetConfigPath())
	if err != nil {
		return nil, err
	}
	logger.SetLevelFromString(cfg.Gateway.LogLevel)
	return cfg, nil
}

func GetWorkspace() string {
	if cfg, err := LoadConfig(); err == nil && cfg.Agents.Defaults.Workspace != "" {
		return cfg.Agents.Defaults.Workspace
	}
	return filepath.Join(GetPicoclawHome(), "workspace")
}

func GetWorkspaceImgDir() string {
	return filepath.Join(GetWorkspace(), "img")
}
