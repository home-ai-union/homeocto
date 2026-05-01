package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	homeconfig "github.com/home-ai-union/homeocto/pkg/config"
	"github.com/sipeed/picoclaw/pkg/config"
)

// registerHomeOctoRoutes binds HomeOcto management endpoints to the ServeMux.
func (h *Handler) registerHomeOctoRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/homeocto/config", h.handleGetHomeOctoConfig)
}

// HomeOctoConfigResponse is the response structure for homeocto config API.
type HomeOctoConfigResponse struct {
	Enabled       bool   `json:"enabled"`
	IntentEnabled bool   `json:"intent_enabled"`
	Port          int    `json:"port"`
	WsURL         string `json:"ws_url"`
	Token         string `json:"token"`
}

// handleGetHomeOctoConfig returns HomeOcto configuration for the frontend.
//
//	GET /api/homeocto/config
func (h *Handler) handleGetHomeOctoConfig(w http.ResponseWriter, r *http.Request) {
	homeCfg, err := homeconfig.LoadHomeConfig()
	if err != nil {
		// If config doesn't exist, return default values
		resp := HomeOctoConfigResponse{
			Enabled:       false,
			IntentEnabled: false,
			Port:          0,
			WsURL:         "",
			Token:         "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Load Pico config to get the token
	picoCfg, err := config.LoadConfig(h.configPath)
	token := ""
	if err == nil && picoCfg != nil {
		if ch, ok := picoCfg.Channels[config.ChannelPico]; ok && ch != nil {
			if decoded, err := ch.GetDecoded(); err == nil && decoded != nil {
				if s, ok := decoded.(*config.PicoSettings); ok {
					token = s.Token.String()
				}
			}
		}
	}

	// Build WebSocket URL if port is configured
	wsURL := ""
	if homeCfg.Port > 0 {
		scheme := "ws"
		if r.TLS != nil {
			scheme = "wss"
		}
		// Use the same host but with HomeOcto port
		host := r.Host
		// Replace the port in the host
		if lastColon := strings.LastIndex(host, ":"); lastColon > 0 {
			host = host[:lastColon]
		}
		wsURL = scheme + "://" + host + ":" + strconv.Itoa(homeCfg.Port) + "/home/ws"
	}

	resp := HomeOctoConfigResponse{
		Enabled:       homeCfg.Enabled,
		IntentEnabled: homeCfg.IntentEnabled,
		Port:          homeCfg.Port,
		WsURL:         wsURL,
		Token:         token,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
