package event

import "time"

// EventType represents the type of event
type EventType string

// Event type constants
const (
	EventTypeDevice    EventType = "device"
	EventTypeRoom      EventType = "room"
	EventTypeProp      EventType = "prop"
	EventTypeDeviceMsg EventType = "device_msg"
	EventTypeToken     EventType = "token"
	EventTypeNet       EventType = "net"
)

// ==================== Event Structure ====================

// Event represents a unified event structure
type Event struct {
	Type      EventType // Event type enum
	Source    string    // Who published the event
	Timestamp time.Time // Event timestamp
	Data      any       // Payload: Device, Space, TokenData, NetData, etc.
}

// NewEvent creates a new Event with the given type, source, and data payload
func NewEvent(eventType EventType, source string, data any) Event {
	return Event{
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// IsType checks if the event is of the given type
func (e *Event) IsType(eventType EventType) bool {
	return e.Type == eventType
}
