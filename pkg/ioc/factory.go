// Package ioc provides the singleton management for all components.
package ioc

import (
	"fmt"
	"path/filepath"
	"sync"

	homeconfig "github.com/home-ai-union/homeocto/pkg/config"
	"github.com/home-ai-union/homeocto/pkg/data"
	"github.com/home-ai-union/homeocto/pkg/llm"
	"github.com/home-ai-union/homeocto/pkg/third"
	hometool "github.com/home-ai-union/homeocto/pkg/tool"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// ErrDisabled is returned by NewFactory when HomeOcto is explicitly disabled or
// homeocto.json is absent. Callers can use errors.Is(err, ErrDisabled) to
// distinguish a deliberate no-op from a real initialisation failure.
var ErrDisabled = fmt.Errorf("homeocto is disabled")

// Factory is the central factory for creating and managing all HomeOcto objects.
// It follows the singleton pattern for components that should exist only once
// per application lifecycle.
type Factory struct {
	Workspace string
	Cfg       *config.Config
	Hcfg      *homeconfig.HomeConfig

	// Singleton instances - lazy loaded
	jsonStore     *data.JSONStore
	storeOnce     sync.Once
	storeErr      error
	deviceStore   data.DeviceStore
	deviceOnce    sync.Once
	deviceErr     error
	spaceStore    data.SpaceStore
	spaceOnce     sync.Once
	spaceErr      error
	homeStore     data.HomeStore
	homeOnce      sync.Once
	homeErr       error
	deviceOpStore data.DeviceOpStore
	deviceOpOnce  sync.Once
	deviceOpErr   error
	toolRegistry  *tools.ToolRegistry
	toolRegOnce   sync.Once

	// LLM tool singleton - lazy loaded
	llmTool *hometool.LLMTool
	llmOnce sync.Once
	llmErr  error

	// Video tool singleton - lazy loaded
	videoTool *hometool.VideoTool
	videoOnce sync.Once
	videoErr  error

	// Common tool singleton - lazy loaded
	commonTool *hometool.CommonTool
	commonOnce sync.Once
	commonErr  error

	// CLI tool singleton - lazy loaded
	cliTool   *hometool.CLITool
	cliToolMu sync.Mutex

	// Auth store - lazy loaded
	authStore data.AuthStore
	authOnce  sync.Once
	authErr   error

	// clientsSetters stores all objects that need to receive clients updates
	clientsSetters []ClientsSetter
	clientsMu      sync.Mutex
	// initializedClients stores the clients that have been initialized
	// This allows late-registered tools to receive already-initialized clients
	initializedClients *third.ClientsManager
}

// ClientsSetter is the interface for objects that can receive clients updates
type ClientsSetter interface {
	SetClients(clients *third.ClientsManager)
}

// NewFactory creates a new Factory instance.
// workspace is the data root used for all HomeOcto data files.
// Returns error when HomeOcto is disabled or homeocto.json is absent.
func NewFactory(workspace string, cfg *config.Config, hcfg *homeconfig.HomeConfig) (*Factory, error) {
	if hcfg == nil || !hcfg.Enabled {
		return nil, ErrDisabled
	}

	return &Factory{
		Workspace:      workspace,
		Cfg:            cfg,
		Hcfg:           hcfg,
		clientsSetters: make([]ClientsSetter, 0),
	}, nil
}

// registerClientsSetter adds an object to the clients setters list.
// This is called automatically when constructing tools that need clients.
// If clients have already been initialized, they are immediately injected.
func (f *Factory) registerClientsSetter(setter ClientsSetter) {
	f.clientsMu.Lock()
	defer f.clientsMu.Unlock()
	f.clientsSetters = append(f.clientsSetters, setter)

	// If clients were already initialized, inject them immediately
	if f.initializedClients != nil {
		setter.SetClients(f.initializedClients)
	}
}

// GetHomeoctoConfig returns the HomeOcto configuration
func (f *Factory) GetHomeoctoConfig() *homeconfig.HomeConfig {
	return f.Hcfg
}

// GetJSONStore returns the singleton JSONStore instance (lazy initialized)
func (f *Factory) GetJSONStore() (*data.JSONStore, error) {
	f.storeOnce.Do(func() {
		f.jsonStore, f.storeErr = data.NewJSONStore(filepath.Join(f.Workspace, "data"))
	})
	return f.jsonStore, f.storeErr
}

// GetDeviceStore returns the singleton DeviceStore instance (lazy initialized)
func (f *Factory) GetDeviceStore() (data.DeviceStore, error) {
	f.deviceOnce.Do(func() {
		store, err := f.GetJSONStore()
		if err != nil {
			f.deviceErr = err
			return
		}

		f.deviceStore, f.deviceErr = data.NewDeviceStore(store)
		if f.deviceErr != nil {
			f.deviceErr = fmt.Errorf("device store init failed: %w", f.deviceErr)
		}
	})
	return f.deviceStore, f.deviceErr
}

// GetSpaceStore returns the singleton SpaceStore instance (lazy initialized)
func (f *Factory) GetSpaceStore() (data.SpaceStore, error) {
	f.spaceOnce.Do(func() {
		store, err := f.GetJSONStore()
		if err != nil {
			f.spaceErr = err
			return
		}

		f.spaceStore, f.spaceErr = data.NewSpaceStore(store)
		if f.spaceErr != nil {
			f.spaceErr = fmt.Errorf("space store init failed: %w", f.spaceErr)
		}
	})
	return f.spaceStore, f.spaceErr
}

// GetHomeStore returns the singleton HomeStore instance (lazy initialized)
func (f *Factory) GetHomeStore() (data.HomeStore, error) {
	f.homeOnce.Do(func() {
		store, err := f.GetJSONStore()
		if err != nil {
			f.homeErr = err
			return
		}

		f.homeStore, f.homeErr = data.NewHomeStore(store)
		if f.homeErr != nil {
			f.homeErr = fmt.Errorf("home store init failed: %w", f.homeErr)
		}
	})
	return f.homeStore, f.homeErr
}

// GetDeviceOpStore returns the singleton DeviceOpStore instance (lazy initialized)
func (f *Factory) GetDeviceOpStore() (data.DeviceOpStore, error) {
	f.deviceOpOnce.Do(func() {
		store, err := f.GetJSONStore()
		if err != nil {
			f.deviceOpErr = err
			return
		}

		// Get device store first
		deviceStore, err := f.GetDeviceStore()
		if err != nil {
			f.deviceOpErr = err
			return
		}

		f.deviceOpStore, f.deviceOpErr = data.NewDeviceOpStore(store, deviceStore)
		if f.deviceOpErr != nil {
			f.deviceOpErr = fmt.Errorf("device op store init failed: %w", f.deviceOpErr)
		}
	})
	return f.deviceOpStore, f.deviceOpErr
}

// GetAuthStore returns the singleton AuthStore instance (lazy initialized).
// It points to workspace/auth, which stores credentials for all brands.
func (f *Factory) GetAuthStore() (data.AuthStore, error) {
	f.authOnce.Do(func() {
		// Get workspace path from config
		workspace := ""
		if f.Cfg != nil {
			workspace = f.Cfg.WorkspacePath()
		}
		if workspace == "" {
			f.authErr = fmt.Errorf("workspace path not configured")
			return
		}

		authDir := filepath.Join(workspace, "auth")
		store, err := data.NewJSONStore(authDir)
		if err != nil {
			f.authErr = fmt.Errorf("create auth json store: %w", err)
			return
		}

		f.authStore, f.authErr = data.NewAuthStore(store)
	})
	return f.authStore, f.authErr
}

// GetToolRegistry returns the singleton ToolRegistry instance (lazy initialized)
func (f *Factory) GetToolRegistry() *tools.ToolRegistry {
	f.toolRegOnce.Do(func() {
		f.toolRegistry = tools.NewToolRegistry()
	})
	return f.toolRegistry
}

// getSmallProvider creates an LLM provider for the small model.
// It resolves the model dynamically based on SmallModel config or falls back to default.
func (f *Factory) getSmallProvider() (providers.LLMProvider, string, error) {
	modelName := f.Hcfg.SmallModel
	if modelName == "" {
		// Fall back to HomeOcto's default model
		modelName = f.Cfg.Agents.Defaults.ModelName
	}

	// Use GetModelConfig to support load balancing (round-robin)
	modelCfg, err := f.Cfg.GetModelConfig(modelName)
	if err != nil {
		return nil, "", fmt.Errorf("small model %q: %w", modelName, err)
	}

	p, modelID, err := providers.CreateProviderFromConfig(modelCfg)
	if err != nil {
		return nil, "", fmt.Errorf("small model %q: %w", modelName, err)
	}
	return p, modelID, nil
}

// GetSmallLLM returns an LLM struct for the small model.
// The provider is created dynamically on each call.
func (f *Factory) GetSmallLLM() (*llm.LLM, error) {
	provider, modelID, err := f.getSmallProvider()
	if err != nil {
		return nil, err
	}
	return &llm.LLM{
		Provider: provider,
		Model:    modelID,
	}, nil
}

// getBigProvider creates an LLM provider for the big model.
// It resolves the model dynamically based on BigModel config or falls back to default.
func (f *Factory) getBigProvider() (providers.LLMProvider, string, error) {
	modelName := f.Hcfg.BigModel
	if modelName == "" {
		// Fall back to HomeOcto's default model
		modelName = f.Cfg.Agents.Defaults.ModelName
	}

	// Use GetModelConfig to support load balancing (round-robin)
	modelCfg, err := f.Cfg.GetModelConfig(modelName)
	if err != nil {
		return nil, "", fmt.Errorf("big model %q: %w", modelName, err)
	}

	p, modelID, err := providers.CreateProviderFromConfig(modelCfg)
	if err != nil {
		return nil, "", fmt.Errorf("big model %q: %w", modelName, err)
	}
	return p, modelID, nil
}

// GetBigLLM returns an LLM struct for the big model.
// The provider is created dynamically on each call.
func (f *Factory) GetBigLLM() (*llm.LLM, error) {
	provider, modelID, err := f.getBigProvider()
	if err != nil {
		return nil, err
	}
	return &llm.LLM{
		Provider: provider,
		Model:    modelID,
	}, nil
}

// GetSmallModelName returns the model name used for small/intent tasks.
func (f *Factory) GetSmallModelName() string {
	if f.Hcfg.SmallModel != "" {
		return f.Hcfg.SmallModel
	}
	// Fall back to HomeOcto's default model
	return f.Cfg.Agents.Defaults.GetModelName()
}

// GetVideoTool returns the singleton VideoTool instance (lazy initialized).
// It provides unified video operations: capImage, capAnalyze.
// The LLM is optional - capImage works without it, capAnalyze will fail at runtime if LLM is unavailable.
func (f *Factory) GetVideoTool() (*hometool.VideoTool, error) {
	f.videoOnce.Do(func() {
		// Get the small LLM instance (optional - capImage doesn't need it)
		smallLLM, _ := f.GetSmallLLM()
		f.videoTool = hometool.NewVideoTool(smallLLM)
	})
	return f.videoTool, f.videoErr
}

// GetLLMTool returns the singleton LLMTool instance (lazy initialized).
// It provides unified LLM operations: image analysis, text processing, device spec analysis.
func (f *Factory) GetLLMTool() (*hometool.LLMTool, error) {
	f.llmOnce.Do(func() {
		// Get the small LLM instance
		smallLLM, err := f.GetSmallLLM()
		if err != nil {
			f.llmErr = fmt.Errorf("failed to get small LLM for LLMTool: %w", err)
			return
		}

		// Get device operation store
		deviceOpStore, err := f.GetDeviceOpStore()
		if err != nil {
			f.llmErr = fmt.Errorf("failed to get device op store for LLMTool: %w", err)
			return
		}

		// Get device store
		deviceStore, err := f.GetDeviceStore()
		if err != nil {
			f.llmErr = fmt.Errorf("failed to get device store for LLMTool: %w", err)
			return
		}

		f.llmTool = hometool.NewLLMToolWithStores(smallLLM, f.Workspace, deviceOpStore, deviceStore)

		// Register for clients updates
		f.registerClientsSetter(f.llmTool)
	})
	return f.llmTool, f.llmErr
}

// GetCommonTool returns the singleton CommonTool instance (lazy initialized).
// It provides common utility methods that are brand-agnostic.
func (f *Factory) GetCommonTool() (*hometool.CommonTool, error) {
	f.commonOnce.Do(func() {
		deviceStore, err := f.GetDeviceStore()
		if err != nil {
			f.commonErr = fmt.Errorf("failed to get device store for CommonTool: %w", err)
			return
		}

		homeStore, err := f.GetHomeStore()
		if err != nil {
			f.commonErr = fmt.Errorf("failed to get home store for CommonTool: %w", err)
			return
		}

		f.commonTool = hometool.NewCommonTool(deviceStore, homeStore)
	})
	return f.commonTool, f.commonErr
}

// GetCLITool returns the singleton CLITool instance (lazy initialized).
// It creates the tool with all required stores; clients and authStore are set separately.
func (f *Factory) GetCLITool() (*hometool.CLITool, error) {
	f.cliToolMu.Lock()
	defer f.cliToolMu.Unlock()

	if f.cliTool != nil {
		return f.cliTool, nil
	}

	homeStore, err := f.GetHomeStore()
	if err != nil {
		return nil, fmt.Errorf("get home store: %w", err)
	}

	spaceStore, err := f.GetSpaceStore()
	if err != nil {
		return nil, fmt.Errorf("get space store: %w", err)
	}

	deviceStore, err := f.GetDeviceStore()
	if err != nil {
		return nil, fmt.Errorf("get device store: %w", err)
	}

	deviceOpStore, err := f.GetDeviceOpStore()
	if err != nil {
		return nil, fmt.Errorf("get device op store: %w", err)
	}

	authStore, err := f.GetAuthStore()
	if err != nil {
		// Log warning but continue without authStore
		// Auth operations won't work but device control will
		authStore = nil
	}

	f.cliTool = hometool.NewCLITool(homeStore, spaceStore, deviceStore, deviceOpStore, authStore)
	f.cliTool.SetWorkspace(f.Workspace)
	// Register for clients updates
	f.registerClientsSetter(f.cliTool)

	return f.cliTool, nil
}

// SetCLIToolClients sets the brand clients for all registered tools.
// This is called by ThirdFactory.GetClients() after initializing all brand clients.
func (f *Factory) SetClients(clients *third.ClientsManager) {
	f.clientsMu.Lock()
	defer f.clientsMu.Unlock()

	// Store the initialized clients for late-registered tools
	f.initializedClients = clients

	// Set clients for all registered setters
	for _, setter := range f.clientsSetters {
		setter.SetClients(clients)
	}
}
