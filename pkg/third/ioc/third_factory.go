// Package ioc provides the ThirdFactory for creating and managing
// third-party smart home platform components (e.g., Xiaomi MIoT).
package ioc

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/AlexxIT/go2rtc/pkg/xiaomi"
	hcc "github.com/home-ai-union/homeocto/pkg/config"
	hcd "github.com/home-ai-union/homeocto/pkg/data"
	"github.com/home-ai-union/homeocto/pkg/event"
	"github.com/home-ai-union/homeocto/pkg/ioc"
	"github.com/home-ai-union/homeocto/pkg/third"
	"github.com/home-ai-union/homeocto/pkg/third/miio"
	"github.com/home-ai-union/homeocto/pkg/third/tuya"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// ThirdFactory is the central factory for creating and managing third-party
// smart home platform components. It follows the singleton pattern for components
// that should exist only once per application lifecycle.
type ThirdFactory struct {
	Workspace string
	cfg       *config.Config
	hcfg      *hcc.HomeConfig
	factory   *ioc.Factory

	// Singleton instances - lazy loaded
	jsonStore *hcd.JSONStore
	storeOnce sync.Once
	storeErr  error

	tuyaModelStore     *hcd.JSONStore
	tuyaModelStoreOnce sync.Once
	tuyaModelStoreErr  error

	tuyaClient     *tuya.TuyaClient
	tuyaClientOnce sync.Once
	tuyaClientErr  error

	miDeviceStore miio.MiDeviceStore
	miHomeStore   miio.MiHomeStore
	cloud         *xiaomi.Cloud
	miClient      *miio.MiClient
	clients       *third.ClientsManager
	clientsOnce   sync.Once
	clientsErr    error

	// Event listener for token changes
	tokenListener event.Listener
}

// NewThirdFactory creates a new ThirdFactory instance.
// workspace is the data root used for all third-party data files.
func NewThirdFactory(factory *ioc.Factory) *ThirdFactory {
	tf := &ThirdFactory{
		Workspace: factory.Workspace,
		cfg:       factory.Cfg,
		hcfg:      factory.Hcfg,
		factory:   factory,
	}

	// Subscribe to token events
	tf.tokenListener = event.NewListener(event.EventTypeToken, tf.onTokenEvent)
	event.GetCenter().Subscribe(event.EventTypeToken, tf.tokenListener)

	return tf
}

// GetJSONStore returns the singleton JSONStore instance (lazy initialized).
func (f *ThirdFactory) GetJSONStore() (*hcd.JSONStore, error) {
	f.storeOnce.Do(func() {
		f.jsonStore, f.storeErr = hcd.NewJSONStore(filepath.Join(f.Workspace, "third"))
	})
	return f.jsonStore, f.storeErr
}

// GetTuyaModelStore returns the singleton JSONStore instance for Tuya model specs (lazy initialized).
func (f *ThirdFactory) GetTuyaModelStore() (*hcd.JSONStore, error) {
	f.tuyaModelStoreOnce.Do(func() {
		f.tuyaModelStore, f.tuyaModelStoreErr = hcd.NewJSONStore(filepath.Join(f.Workspace, "third", "tuya-spec"))
	})
	return f.tuyaModelStore, f.tuyaModelStoreErr
}

// GetAuthStore returns the singleton AuthStore instance (lazy initialized).
// It delegates to the main factory's GetAuthStore.
func (f *ThirdFactory) GetAuthStore() (hcd.AuthStore, error) {
	return f.factory.GetAuthStore()
}

// GetMiDeviceStore returns the singleton MiDeviceStore instance (lazy initialized).
func (f *ThirdFactory) GetMiDeviceStore() (miio.MiDeviceStore, error) {
	if f.miDeviceStore != nil {
		return f.miDeviceStore, nil
	}

	store, err := f.GetJSONStore()
	if err != nil {
		return nil, fmt.Errorf("get json store: %w", err)
	}

	f.miDeviceStore, err = miio.NewMiDeviceStore(store)
	if err != nil {
		return nil, fmt.Errorf("mi device store init failed: %w", err)
	}
	return f.miDeviceStore, nil
}

// GetMiHomeStore returns the singleton MiHomeStore instance (lazy initialized).
func (f *ThirdFactory) GetMiHomeStore() (miio.MiHomeStore, error) {
	if f.miHomeStore != nil {
		return f.miHomeStore, nil
	}

	store, err := f.GetJSONStore()
	if err != nil {
		return nil, fmt.Errorf("get json store: %w", err)
	}

	f.miHomeStore, err = miio.NewMiHomeStore(store)
	if err != nil {
		return nil, fmt.Errorf("mi home store init failed: %w", err)
	}
	return f.miHomeStore, nil
}

// GetCloud returns the singleton Cloud instance (lazy initialized).
// The sid parameter defaults to "xiaomiio" if empty.
func (f *ThirdFactory) GetCloud(sid string) *xiaomi.Cloud {
	if f.cloud != nil {
		return f.cloud
	}
	if sid == "" {
		sid = "xiaomiio"
	}
	f.cloud = xiaomi.NewCloud(sid)
	var Xiaomi struct {
		Cfg map[string]string `yaml:"xiaomi"`
	}

	hcc.LoadGo2RTCConfig(&Xiaomi)

	// Get first key-value pair: userId=key, token=value
	var userId, token string
	for k, v := range Xiaomi.Cfg {
		userId = k
		token = v
		break
	}
	f.cloud.LoginWithToken(userId, token)
	return f.cloud
}

// GetMiClient returns the singleton MiClient instance (lazy initialized).
//
// Parameters:
//   - country: region code (cn, de, ru, sg, i2, us, etc.)
func (f *ThirdFactory) GetMiClient(country string) (*miio.MiClient, error) {
	if f.miClient != nil {
		return f.miClient, nil
	}

	cloud := f.GetCloud("xiaomiio")

	// Get device store for persistent device caching
	deviceStore, err := f.GetMiDeviceStore()
	if err != nil {
		return nil, fmt.Errorf("get mi device store: %w", err)
	}

	// Get home store for persistent home/room caching
	homeStore, err := f.GetMiHomeStore()
	if err != nil {
		return nil, fmt.Errorf("get mi home store: %w", err)
	}

	f.miClient = miio.NewMiClient(cloud, country, deviceStore, homeStore)
	return f.miClient, nil
}

// GetTuyaClient returns the singleton TuyaClient instance (lazy initialized).
// It reads the API token from the AuthStore.
// Returns nil, nil if no token is configured.
func (f *ThirdFactory) GetTuyaClient() (*tuya.TuyaClient, error) {
	f.tuyaClientOnce.Do(func() {
		authStore, err := f.GetAuthStore()
		if err != nil {
			f.tuyaClientErr = fmt.Errorf("get auth store: %w", err)
			return
		}

		// Read token if available
		var token string
		if authStore.Exists("tuya_token") {
			_, _, token, _, err = authStore.GetDecryptedBrand("tuya_token")
			if err != nil {
				f.tuyaClientErr = fmt.Errorf("decrypt tuya token: %w", err)
				return
			}
		}

		// Get email, password and region from AuthStore (optional)
		var email, password, region string
		if authStore.Exists("tuya_pass") {
			region, email, password, _, err = authStore.GetDecryptedBrand("tuya_pass")
			if err != nil {
				// Log warning but continue with empty credentials
				email = ""
				password = ""
				region = ""
			}
		}

		store, err := f.GetJSONStore()
		if err != nil {
			f.tuyaClientErr = fmt.Errorf("get json store: %w", err)
			return
		}

		modelStore, err := f.GetTuyaModelStore()
		if err != nil {
			f.tuyaClientErr = fmt.Errorf("get tuya model store: %w", err)
			return
		}

		f.tuyaClient, f.tuyaClientErr = tuya.NewTuyaClient(store, modelStore, token, email, password, region)
		if f.tuyaClientErr != nil {
			return
		}

		// Set auth store to enable lazy token loading
		f.tuyaClient.SetAuthStore(authStore)
	})
	return f.tuyaClient, f.tuyaClientErr
}

// registerTuyaClient registers the Tuya client to the clients manager.
// It handles all error cases and logging internally.
func (f *ThirdFactory) registerTuyaClient(clients *third.ClientsManager) {
	tuyaClient, tuyaErr := f.GetTuyaClient()
	if tuyaErr != nil {
		logger.Warnf("init tuya client err %v -----------------------", tuyaErr)
	} else if tuyaClient == nil {
		logger.Debug("tuya client is nil (unexpected), skipping-------------------------------")
	} else if tuyaClient.GetAPIKey() == "" {
		logger.Debug("tuya client created without token (not yet configured), registered for later use-------------------------------")
		clients.Add(tuyaClient)
	} else {
		logger.Debug("set tuya client-------------------------------")
		clients.Add(tuyaClient)
	}
}

// onTokenEvent handles token change events and triggers client refresh.
func (f *ThirdFactory) onTokenEvent(evt event.Event) {
	data, ok := evt.Data.(map[string]any)
	if !ok {
		logger.Warnf("[TokenEvent] Invalid event data format")
		return
	}

	brand, _ := data["brand"].(string)
	action, _ := data["action"].(string)

	logger.Infof("[TokenEvent] Received token event: brand=%s, action=%s", brand, action)
	if brand == "tuya" {
		f.ResetTuyaClient()
	}

}

// ResetTuyaClient clears the cached Tuya client so it can be recreated with fresh credentials.
// This should be called after saving or deleting Tuya authentication.
func (f *ThirdFactory) ResetTuyaClient() {
	f.tuyaClientOnce = sync.Once{}
	f.tuyaClient = nil
	f.tuyaClientErr = nil
	f.registerTuyaClient(f.clients)
}

// ResetClients clears all cached clients and singletons so they can be
// recreated with fresh configuration. This is called during config reload
// to re-initialize third-party clients with the new config.
func (f *ThirdFactory) ResetClients() {
	f.clientsOnce = sync.Once{}
	f.clients = nil
	f.clientsErr = nil

	f.tuyaClientOnce = sync.Once{}
	f.tuyaClient = nil
	f.tuyaClientErr = nil

	f.miClient = nil
	f.cloud = nil
}

// GetClients initializes and returns the singleton ClientsManager.
// This method initializes all available brand clients and injects them into the tools.
func (f *ThirdFactory) GetClients() (*third.ClientsManager, error) {
	f.clientsOnce.Do(func() {
		clients := third.NewClientsManager()

		// Register Xiaomi client if available
		miClient, miErr := f.GetMiClient("cn")
		if miErr == nil && miClient != nil {
			logger.Debug("set xiaomi client-------------------------------")
			clients.Add(miClient)
		}

		// Register Tuya client if available
		f.registerTuyaClient(clients)

		if len(clients.ListBrands()) == 0 {
			logger.Debug("no brand clients configured-------------------------------")
		}
		f.clients = clients
		// Inject clients into CLI tool and LLM tool
		f.factory.SetClients(clients)
	})

	return f.clients, f.clientsErr
}
