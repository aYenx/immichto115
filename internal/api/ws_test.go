package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aYenx/immichto115/internal/config"
	"github.com/aYenx/immichto115/internal/rclone"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestWebSocketUpgradeAndBroadcast(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := NewHub()

	// Create test server without auth
	router := gin.New()
	router.GET("/ws/logs", HandleWebSocket(hub, &Server{
		Config:    newTestConfigManager(t, false), // auth disabled
		authLimit: newAuthLimiter(),
		Build:     BuildInfo{Version: "test"},
	}))

	server := httptest.NewServer(router)
	defer server.Close()

	// Connect via WebSocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/logs"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial error: %v", err)
	}
	defer conn.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected 101, got %d", resp.StatusCode)
	}

	// Read welcome message
	var welcome rclone.LogLine
	if err := conn.ReadJSON(&welcome); err != nil {
		t.Fatalf("read welcome error: %v", err)
	}
	if welcome.Stream != "stdout" {
		t.Fatalf("expected stream=stdout, got %q", welcome.Stream)
	}

	// Broadcast a message
	testLine := rclone.LogLine{Stream: "stdout", Text: "test broadcast message"}
	hub.Broadcast(testLine)

	// Read the broadcast
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var received rclone.LogLine
	if err := conn.ReadJSON(&received); err != nil {
		t.Fatalf("read broadcast error: %v", err)
	}
	if received.Text != "test broadcast message" {
		t.Fatalf("expected broadcast text, got %q", received.Text)
	}
}

func TestWebSocketAuthRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := NewHub()

	// Create test server with auth enabled
	router := gin.New()
	router.GET("/ws/logs", HandleWebSocket(hub, &Server{
		Config:    newTestConfigManager(t, true), // auth enabled
		authLimit: newAuthLimiter(),
		Build:     BuildInfo{Version: "test"},
	}))

	server := httptest.NewServer(router)
	defer server.Close()

	// Try to connect without auth → should fail
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/logs"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected websocket dial to fail without auth")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHubBroadcastAndUnregister(t *testing.T) {
	hub := NewHub()

	hub.mu.RLock()
	if len(hub.clients) != 0 {
		t.Fatalf("expected 0 clients, got %d", len(hub.clients))
	}
	hub.mu.RUnlock()

	// Broadcast to empty hub should not panic
	hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "no clients"})
}

// TestWebSocketQueryTokenRejected verifies that after removing c.Query("token")
// from ws.go, a WebSocket connection attempt using ?token= is rejected, while
// cookie-based JWT auth still succeeds.
func TestWebSocketQueryTokenRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := NewHub()

	// Create auth-enabled config with JWT secret
	cfgMgr := newTestConfigManager(t, true)
	cfg := cfgMgr.Get()
	cfg.Server.JWTSecret = generateJWTSecret()
	if err := cfgMgr.Update(cfg); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	router := gin.New()
	router.GET("/ws/logs", HandleWebSocket(hub, &Server{
		Config:    cfgMgr,
		authLimit: newAuthLimiter(),
		Build:     BuildInfo{Version: "test"},
	}))

	server := httptest.NewServer(router)
	defer server.Close()

	// Generate a valid JWT token
	csrf := generateCSRFToken()
	token, err := createToken("admin", csrf, cfg.Server.JWTSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("createToken error: %v", err)
	}

	t.Run("query param token rejected", func(t *testing.T) {
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/logs?token=" + token
		_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err == nil {
			t.Fatal("expected websocket dial to fail with query token")
		}
		if resp != nil && resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("cookie JWT accepted", func(t *testing.T) {
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/logs"
		header := http.Header{}
		header.Set("Cookie", jwtCookieName+"="+token)
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err != nil {
			t.Fatalf("expected websocket dial to succeed with cookie JWT, got: %v", err)
		}
		defer conn.Close()
		if resp.StatusCode != http.StatusSwitchingProtocols {
			t.Fatalf("expected 101, got %d", resp.StatusCode)
		}
	})
}

// helper: create a config manager for tests
func newTestConfigManager(t *testing.T, authEnabled bool) *config.Manager {
	t.Helper()
	cfgPath := t.TempDir() + "/config.yaml"
	mgr, err := config.NewManager(cfgPath)
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}
	cfg := mgr.Get()
	cfg.Server.AuthEnabled = authEnabled
	if authEnabled {
		hash, _ := config.HashPassword("testpass")
		cfg.Server.AuthUser = "admin"
		cfg.Server.AuthPasswordHash = hash
	}
	if err := mgr.Update(cfg); err != nil {
		t.Fatalf("Update error: %v", err)
	}
	return mgr
}
