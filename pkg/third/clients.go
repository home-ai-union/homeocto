package third

import (
	"fmt"
	"sort"
	"sync"
)

// ClientsManager manages multiple third-party platform client instances.
// It provides thread-safe operations for adding, retrieving, and listing clients.
type ClientsManager struct {
	mu      sync.RWMutex
	clients map[string]Client
}

// NewClientsManager creates a new ClientsManager instance.
func NewClientsManager() *ClientsManager {
	return &ClientsManager{
		clients: make(map[string]Client),
	}
}

// Add registers a client instance. If a client with the same brand already exists,
// it will be replaced.
func (m *ClientsManager) Add(client Client) {
	if client == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[client.Brand()] = client
}

// Get retrieves a client by its brand name.
// Returns an error if the brand is not registered.
func (m *ClientsManager) Get(brand string) (Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[brand]
	if !ok {
		return nil, fmt.Errorf("client for brand %q not found", brand)
	}

	return client, nil
}

// ListBrands returns a sorted list of all registered brand names.
func (m *ClientsManager) ListBrands() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	brands := make([]string, 0, len(m.clients))
	for brand := range m.clients {
		brands = append(brands, brand)
	}
	sort.Strings(brands)

	return brands
}

// HasClient checks whether a client for the given brand is registered.
func (m *ClientsManager) HasClient(brand string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.clients[brand]
	return ok
}

// Remove unregisters a client by its brand name.
// It returns true if the client was found and removed, false otherwise.
func (m *ClientsManager) Remove(brand string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.clients[brand]
	if ok {
		delete(m.clients, brand)
	}

	return ok
}

// ToMap returns a copy of all registered clients as a map.
// This is useful for backward compatibility with existing code.
func (m *ClientsManager) ToMap() map[string]Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]Client, len(m.clients))
	for brand, client := range m.clients {
		result[brand] = client
	}

	return result
}
