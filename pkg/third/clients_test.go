package third

import (
	"testing"

	"github.com/home-ai-union/homeocto/pkg/data"
)

// mockClient is a mock implementation of the Client interface for testing
type mockClient struct {
	brand string
}

func (m *mockClient) Brand() string {
	return m.brand
}

// Implement other Client interface methods with stubs
func (m *mockClient) GetHomes() ([]*HomeInfo, error)            { return nil, nil }
func (m *mockClient) GetRooms(string) ([]*data.Space, error)    { return nil, nil }
func (m *mockClient) GetDevices(string) ([]*data.Device, error) { return nil, nil }
func (m *mockClient) GetSpec(string) (*SpecInfo, error)         { return nil, nil }
func (m *mockClient) Execute(map[string]any) (map[string]any, error) {
	return nil, nil
}
func (m *mockClient) GetProps(map[string]any) (any, error) { return nil, nil }
func (m *mockClient) SetProps(map[string]any) (any, error) { return nil, nil }
func (m *mockClient) EnableEvent(map[string]any) error     { return nil }
func (m *mockClient) DisableEvent(map[string]any) error    { return nil }
func (m *mockClient) GetRtspStr(string) (string, error)    { return "", nil }

func TestClientsManager_AddAndGet(t *testing.T) {
	manager := NewClientsManager()

	// Test adding clients
	xiaomi := &mockClient{brand: "xiaomi"}
	tuya := &mockClient{brand: "tuya"}

	manager.Add(xiaomi)
	manager.Add(tuya)

	// Test getting clients
	client, err := manager.Get("xiaomi")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client.Brand() != "xiaomi" {
		t.Errorf("expected brand 'xiaomi', got %s", client.Brand())
	}

	// Test getting non-existent client
	_, err = manager.Get("unknown")
	if err == nil {
		t.Error("expected error for unknown brand, got nil")
	}
}

func TestClientsManager_ListBrands(t *testing.T) {
	manager := NewClientsManager()

	// Add clients in random order
	manager.Add(&mockClient{brand: "tuya"})
	manager.Add(&mockClient{brand: "xiaomi"})
	manager.Add(&mockClient{brand: "apple"})

	brands := manager.ListBrands()

	// Should be sorted
	if len(brands) != 3 {
		t.Fatalf("expected 3 brands, got %d", len(brands))
	}

	expected := []string{"apple", "tuya", "xiaomi"}
	for i, brand := range brands {
		if brand != expected[i] {
			t.Errorf("expected brand[%d] to be %s, got %s", i, expected[i], brand)
		}
	}
}

func TestClientsManager_HasClient(t *testing.T) {
	manager := NewClientsManager()
	manager.Add(&mockClient{brand: "xiaomi"})

	if !manager.HasClient("xiaomi") {
		t.Error("expected HasClient('xiaomi') to be true")
	}

	if manager.HasClient("tuya") {
		t.Error("expected HasClient('tuya') to be false")
	}
}

func TestClientsManager_Remove(t *testing.T) {
	manager := NewClientsManager()
	manager.Add(&mockClient{brand: "xiaomi"})
	manager.Add(&mockClient{brand: "tuya"})

	// Remove existing client
	if !manager.Remove("xiaomi") {
		t.Error("expected Remove('xiaomi') to return true")
	}

	if manager.HasClient("xiaomi") {
		t.Error("expected 'xiaomi' to be removed")
	}

	// Remove non-existent client
	if manager.Remove("unknown") {
		t.Error("expected Remove('unknown') to return false")
	}
}

func TestClientsManager_ToMap(t *testing.T) {
	manager := NewClientsManager()
	manager.Add(&mockClient{brand: "xiaomi"})
	manager.Add(&mockClient{brand: "tuya"})

	clientsMap := manager.ToMap()

	if len(clientsMap) != 2 {
		t.Fatalf("expected 2 clients in map, got %d", len(clientsMap))
	}

	if _, ok := clientsMap["xiaomi"]; !ok {
		t.Error("expected 'xiaomi' in map")
	}

	if _, ok := clientsMap["tuya"]; !ok {
		t.Error("expected 'tuya' in map")
	}
}

func TestClientsManager_AddNil(t *testing.T) {
	manager := NewClientsManager()
	manager.Add(nil)

	if len(manager.ListBrands()) != 0 {
		t.Error("expected no clients after adding nil")
	}
}

func TestClientsManager_Replace(t *testing.T) {
	manager := NewClientsManager()

	// Add initial client
	manager.Add(&mockClient{brand: "xiaomi"})

	// Replace with new client
	newXiaomi := &mockClient{brand: "xiaomi"}
	manager.Add(newXiaomi)

	// Should still have only one client
	brands := manager.ListBrands()
	if len(brands) != 1 {
		t.Errorf("expected 1 brand after replacement, got %d", len(brands))
	}

	// Should get the new client
	client, _ := manager.Get("xiaomi")
	if client.Brand() != newXiaomi.Brand() {
		t.Error("expected to get the replaced client")
	}
}
