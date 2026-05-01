package home

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	homeconfig "github.com/home-ai-union/homeocto/pkg/config"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// HomeChannelConfig holds the configuration for HomeChannel.
// Token, AllowOrigins, and AllowTokenQuery are extracted from picoclaw's Pico channel config.
// Port comes from homeocto's homeConfig.ToolWSPort.
type HomeChannelConfig struct {
	Token           string
	AllowOrigins    []string
	AllowTokenQuery bool
	MaxConnections  int
	PingInterval    int // seconds
	ReadTimeout     int // seconds
	Port            int
}

// homeConn wraps a WebSocket connection.
// Reference: pico channel's picoConn (picoclaw/pkg/channels/pico/pico.go L25-32)
type homeConn struct {
	id        string
	conn      *websocket.Conn
	sessionID string
	writeMu   sync.Mutex
	closed    atomic.Bool
	cancel    context.CancelFunc
}

func (hc *homeConn) writeJSON(v any) error {
	if hc.closed.Load() {
		return fmt.Errorf("connection closed")
	}
	hc.writeMu.Lock()
	defer hc.writeMu.Unlock()
	return hc.conn.WriteJSON(v)
}

func (hc *homeConn) close() {
	if hc.closed.CompareAndSwap(false, true) {
		hc.conn.Close()
		if hc.cancel != nil {
			hc.cancel()
		}
	}
}

// HomeChannel implements a lightweight WebSocket channel for direct tool execution.
// Reference: pico channel's PicoChannel (picoclaw/pkg/channels/pico/pico.go L71-81)
type HomeChannel struct {
	config       HomeChannelConfig
	toolExecutor ToolExecutor
	upgrader     websocket.Upgrader

	// Connection management (reference: pico L76-78)
	connections        map[string]*homeConn
	sessionConnections map[string]map[string]*homeConn
	connsMu            sync.RWMutex

	running    atomic.Bool
	httpServer *http.Server
	ctx        context.Context
	cancel     context.CancelFunc
}

// BuildHomeChannelConfig merges configuration from picoclaw and homeocto.
func BuildHomeChannelConfig(picoConfig *config.Config, homeConfig *homeconfig.HomeConfig) HomeChannelConfig {
	cfg := HomeChannelConfig{
		Port:           homeConfig.Port,
		MaxConnections: 100,
		PingInterval:   30,
		ReadTimeout:    60,
	}
	// Extract token/allowOrigins/allowTokenQuery from Pico channel config
	if ch, ok := picoConfig.Channels[config.ChannelPico]; ok && ch != nil {
		if decoded, err := ch.GetDecoded(); err == nil && decoded != nil {
			if s, ok := decoded.(*config.PicoSettings); ok {
				cfg.Token = s.Token.String()
				cfg.AllowOrigins = s.AllowOrigins
				cfg.AllowTokenQuery = s.AllowTokenQuery
			}
		}
	}
	return cfg
}

// NewHomeChannel creates a new HomeChannel instance.
func NewHomeChannel(cfg HomeChannelConfig, executor ToolExecutor) *HomeChannel {
	ctx, cancel := context.WithCancel(context.Background())

	hc := &HomeChannel{
		config:             cfg,
		toolExecutor:       executor,
		connections:        make(map[string]*homeConn),
		sessionConnections: make(map[string]map[string]*homeConn),
		ctx:                ctx,
		cancel:             cancel,
	}

	// Configure WebSocket upgrader (reference: pico L83-96)
	hc.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if len(cfg.AllowOrigins) == 0 {
				return true
			}
			origin := r.Header.Get("Origin")
			for _, allowed := range cfg.AllowOrigins {
				if allowed == "*" || allowed == origin {
					return true
				}
			}
			return false
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	return hc
}

// Name returns the channel name.
func (hc *HomeChannel) Name() string { return "home" }

// IsRunning returns whether the channel is running.
func (hc *HomeChannel) IsRunning() bool { return hc.running.Load() }

// Start starts the HomeChannel HTTP server.
func (hc *HomeChannel) Start(ctx context.Context) error {
	if hc.running.CompareAndSwap(false, true) {
		hc.ctx, hc.cancel = context.WithCancel(ctx)

		mux := http.NewServeMux()
		mux.HandleFunc("/home/ws", hc.handleWebSocket)

		addr := fmt.Sprintf(":%d", hc.config.Port)
		hc.httpServer = &http.Server{
			Addr:    addr,
			Handler: mux,
		}

		go func() {
			logger.InfoCF("homeocto", "Starting HomeChannel WebSocket server", map[string]any{
				"addr": addr,
				"path": "/home/ws",
			})
			if err := hc.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.ErrorCF("homeocto", "HomeChannel WebSocket server error", map[string]any{
					"error": err.Error(),
				})
			}
		}()
	}
	return nil
}

// Stop stops the HomeChannel HTTP server and closes all connections.
func (hc *HomeChannel) Stop(ctx context.Context) error {
	hc.running.Store(false)

	// Close all connections
	hc.closeAllConnections()

	// Shutdown HTTP server
	if hc.httpServer != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := hc.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.ErrorCF("homeocto", "HomeChannel server shutdown error", map[string]any{
				"error": err.Error(),
			})
		}
		hc.httpServer = nil
	}

	if hc.cancel != nil {
		hc.cancel()
	}

	logger.InfoCF("homeocto", "HomeChannel stopped", nil)
	return nil
}

// Send broadcasts a message to all connected clients.
// Reference: pico channel's Send() (pico.go L259)
func (hc *HomeChannel) Send(ctx context.Context, msg any) error {
	hc.connsMu.RLock()
	conns := make([]*homeConn, 0, len(hc.connections))
	for _, c := range hc.connections {
		conns = append(conns, c)
	}
	hc.connsMu.RUnlock()

	for _, c := range conns {
		_ = c.writeJSON(msg)
	}
	return nil
}

// WebhookPath returns the webhook path prefix.
func (hc *HomeChannel) WebhookPath() string { return "/home/" }

// ServeHTTP routes incoming HTTP requests to the appropriate handler.
func (hc *HomeChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/home/ws" || path == "/home/ws/" {
		hc.handleWebSocket(w, r)
	} else {
		http.NotFound(w, r)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// WebSocket lifecycle (reference: pico channel)
// ─────────────────────────────────────────────────────────────────────────────

// handleWebSocket upgrades the HTTP connection and handles the WebSocket session.
// Reference: pico channel's handleWebSocket (pico.go L345-404)
func (hc *HomeChannel) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	logger.Infof("[HomeChannel.handleWebSocket] called")

	// Check if running
	if !hc.running.Load() {
		http.Error(w, "HomeChannel not running", http.StatusServiceUnavailable)
		return
	}

	// Authenticate
	if !hc.authenticate(r) {
		logger.WarnCF("homeocto", "WebSocket authentication failed", nil)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Echo the matched subprotocol back
	proto := hc.matchedSubprotocol(r)
	var responseHeader http.Header
	if proto != "" {
		responseHeader = http.Header{"Sec-WebSocket-Protocol": {proto}}
	}

	// Upgrade to WebSocket
	conn, err := hc.upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		logger.ErrorCF("homeocto", "WebSocket upgrade failed", map[string]any{"error": err.Error()})
		return
	}

	// Check connection limit
	if hc.currentConnCount() >= hc.config.MaxConnections {
		conn.Close()
		logger.WarnCF("homeocto", "Connection limit exceeded", map[string]any{
			"max": hc.config.MaxConnections,
		})
		return
	}

	// Create connection and add to pool
	hcConn, err := hc.createAndAddConnection(conn, r)
	if err != nil {
		conn.Close()
		logger.ErrorCF("homeocto", "Failed to create connection", map[string]any{"error": err.Error()})
		return
	}

	logger.InfoCF("homeocto", "HomeChannel client connected", map[string]any{
		"conn_id": hcConn.id,
		"session": hcConn.sessionID,
	})

	// Start read loop
	go hc.readLoop(hcConn)
}

// authenticate checks the request for a valid token.
// Reference: pico channel's authenticate (pico.go L410-437)
// 1. Authorization: Bearer <token> header
// 2. Sec-WebSocket-Protocol "token.<value>"
// 3. Query parameter "token" (only when AllowTokenQuery is on)
func (hc *HomeChannel) authenticate(r *http.Request) bool {
	token := hc.config.Token
	if token == "" {
		return false
	}

	// Check Bearer token
	if auth := r.Header.Get("Authorization"); auth != "" {
		if after, ok := strings.CutPrefix(auth, "Bearer "); ok && after == token {
			return true
		}
	}

	// Check Sec-WebSocket-Protocol
	for _, proto := range websocket.Subprotocols(r) {
		if after, ok := strings.CutPrefix(proto, "token."); ok && after == token {
			return true
		}
	}

	// Check query parameter
	if hc.config.AllowTokenQuery {
		if r.URL.Query().Get("token") == token {
			return true
		}
	}

	return false
}

// matchedSubprotocol returns the "token.<value>" subprotocol that matches.
func (hc *HomeChannel) matchedSubprotocol(r *http.Request) string {
	if hc.config.Token == "" {
		return ""
	}
	for _, proto := range websocket.Subprotocols(r) {
		if after, ok := strings.CutPrefix(proto, "token."); ok && after == hc.config.Token {
			return proto
		}
	}
	return ""
}

// readLoop reads messages from the WebSocket connection.
// Reference: pico channel's readLoop (pico.go L452-510)
func (hc *HomeChannel) readLoop(conn *homeConn) {
	defer func() {
		hc.removeConnection(conn.id)
		conn.close()
		logger.InfoCF("homeocto", "HomeChannel client disconnected", map[string]any{
			"conn_id": conn.id,
		})
	}()

	readTimeout := time.Duration(hc.config.ReadTimeout) * time.Second
	_ = conn.conn.SetReadDeadline(time.Now().Add(readTimeout))
	conn.conn.SetPongHandler(func(string) error {
		_ = conn.conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})

	// Start ping loop
	go hc.pingLoop(conn)

	for {
		select {
		case <-hc.ctx.Done():
			return
		default:
		}

		_, raw, err := conn.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.DebugCF("homeocto", "WebSocket read error", map[string]any{"error": err.Error()})
			}
			return
		}

		_ = conn.conn.SetReadDeadline(time.Now().Add(readTimeout))

		var msg homeMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			conn.writeJSON(newError("invalid_message", "failed to parse message"))
			continue
		}

		hc.handleMessage(conn, msg)
	}
}

// pingLoop sends periodic ping messages.
// Reference: pico channel's pingLoop (pico.go L513-533)
func (hc *HomeChannel) pingLoop(conn *homeConn) {
	pingInterval := time.Duration(hc.config.PingInterval) * time.Second
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if conn.closed.Load() {
				return
			}
			conn.writeMu.Lock()
			err := conn.conn.WriteMessage(websocket.PingMessage, nil)
			conn.writeMu.Unlock()
			if err != nil {
				return
			}
		case <-hc.ctx.Done():
			return
		}
	}
}

// handleMessage dispatches incoming messages to the appropriate handler.
// Reference: pico channel's handleMessage (pico.go L536-553)
func (hc *HomeChannel) handleMessage(conn *homeConn, msg homeMessage) {
	switch msg.Type {
	case TypePing:
		conn.writeJSON(newMessage(TypePong, nil))

	case TypeMessageSend, TypeMediaSend:
		hc.executeToolCall(conn, msg)

	default:
		logger.WarnCF("homeocto", "Unknown message type", map[string]any{
			"type": msg.Type,
		})
	}
}

// executeToolCall parses and executes a tool call.
func (hc *HomeChannel) executeToolCall(conn *homeConn, msg homeMessage) {
	if hc.toolExecutor == nil {
		conn.writeJSON(newError("not_initialized", "tool executor not initialized"))
		return
	}

	content, _ := msg.Payload["content"].(string)
	if strings.TrimSpace(content) == "" {
		return
	}

	// Parse tool command
	toolName, commandJSON, ok := hc.toolExecutor.ParseToolCommand(content)
	if !ok {
		hc.sendToolResponse(conn, msg.ID, "not a valid tool command", true)
		return
	}

	logger.InfoCF("homeocto", "HomeChannel tool call received", map[string]any{
		"tool_name": toolName,
		"msg_id":    msg.ID,
	})

	// Execute tool
	result, isError := hc.toolExecutor.ExecuteTool(hc.ctx, toolName, commandJSON)

	if isError {
		logger.ErrorCF("homeocto", "HomeChannel tool execution failed", map[string]any{
			"tool_name": toolName,
			"error":     result,
		})
		hc.sendToolResponse(conn, msg.ID, result, true)
		return
	}

	logger.InfoCF("homeocto", "HomeChannel tool executed successfully", map[string]any{
		"tool_name":     toolName,
		"result_length": len(result),
	})

	hc.sendToolResponse(conn, msg.ID, result, false)
}

// sendToolResponse sends a tool response back to the client.
func (hc *HomeChannel) sendToolResponse(conn *homeConn, originalMsgID string, content string, isError bool) {
	response := newMessage(TypeMessageCreate, map[string]any{
		"content":  content,
		"is_error": isError,
	})
	response.ID = "tool-response-" + originalMsgID

	if err := conn.writeJSON(response); err != nil {
		logger.ErrorCF("homeocto", "Failed to send tool response", map[string]any{
			"error": err.Error(),
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Connection management (reference: pico channel L124-214)
// ─────────────────────────────────────────────────────────────────────────────

// createAndAddConnection creates a homeConn and adds it to the connection pool.
// Reference: pico channel's createAndAddConnection (pico.go L124-154)
func (hc *HomeChannel) createAndAddConnection(conn *websocket.Conn, r *http.Request) (*homeConn, error) {
	hc.connsMu.Lock()
	defer hc.connsMu.Unlock()

	connID := uuid.New().String()
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	connCtx, connCancel := context.WithCancel(hc.ctx)
	_ = connCtx // used for per-connection context

	hcConn := &homeConn{
		id:        connID,
		conn:      conn,
		sessionID: sessionID,
		cancel:    connCancel,
	}

	hc.connections[connID] = hcConn

	// Track by session
	if _, ok := hc.sessionConnections[sessionID]; !ok {
		hc.sessionConnections[sessionID] = make(map[string]*homeConn)
	}
	hc.sessionConnections[sessionID][connID] = hcConn

	return hcConn, nil
}

// removeConnection removes a connection from the pool.
// Reference: pico channel's removeConnection (pico.go L157-175)
func (hc *HomeChannel) removeConnection(connID string) {
	hc.connsMu.Lock()
	defer hc.connsMu.Unlock()

	conn, ok := hc.connections[connID]
	if !ok {
		return
	}

	delete(hc.connections, connID)

	// Remove from session tracking
	if conn.sessionID != "" {
		if sessionConns, ok := hc.sessionConnections[conn.sessionID]; ok {
			delete(sessionConns, connID)
			if len(sessionConns) == 0 {
				delete(hc.sessionConnections, conn.sessionID)
			}
		}
	}
}

// closeAllConnections closes and removes all connections.
func (hc *HomeChannel) closeAllConnections() {
	hc.connsMu.Lock()
	defer hc.connsMu.Unlock()

	for connID, conn := range hc.connections {
		conn.close()
		delete(hc.connections, connID)
	}
	hc.sessionConnections = make(map[string]map[string]*homeConn)
}

// currentConnCount returns the current number of connections.
func (hc *HomeChannel) currentConnCount() int {
	hc.connsMu.RLock()
	defer hc.connsMu.RUnlock()
	return len(hc.connections)
}
