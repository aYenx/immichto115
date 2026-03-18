package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aYenx/immichto115/internal/config"
	"github.com/gin-gonic/gin"
)

func TestCreateAndValidateToken(t *testing.T) {
	secret := generateJWTSecret()
	csrf := generateCSRFToken()

	token, err := createToken("admin", csrf, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("createToken error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Validate
	claims, err := validateToken(token, secret)
	if err != nil {
		t.Fatalf("validateToken error: %v", err)
	}
	if claims.Sub != "admin" {
		t.Fatalf("expected sub=admin, got %q", claims.Sub)
	}
	if claims.CSRF != csrf {
		t.Fatalf("expected csrf=%q, got %q", csrf, claims.CSRF)
	}
	if claims.Exp == 0 || claims.Iat == 0 {
		t.Fatal("expected non-zero exp and iat")
	}
}

func TestExpiredToken(t *testing.T) {
	secret := generateJWTSecret()

	// Create token with -1h expiry (already expired)
	token, err := createToken("admin", "csrf123", secret, -1*time.Hour)
	if err != nil {
		t.Fatalf("createToken error: %v", err)
	}

	_, err = validateToken(token, secret)
	if err != errTokenExpired {
		t.Fatalf("expected errTokenExpired, got %v", err)
	}
}

func TestInvalidSignature(t *testing.T) {
	secret1 := generateJWTSecret()
	secret2 := generateJWTSecret()

	token, err := createToken("admin", "csrf123", secret1, 1*time.Hour)
	if err != nil {
		t.Fatalf("createToken error: %v", err)
	}

	_, err = validateToken(token, secret2)
	if err != errTokenInvalid {
		t.Fatalf("expected errTokenInvalid, got %v", err)
	}
}

func TestMalformedToken(t *testing.T) {
	_, err := validateToken("not.a.valid.token.string", "secret")
	if err != errTokenMalformed && err != errTokenInvalid {
		t.Fatalf("expected malformed/invalid error, got %v", err)
	}

	_, err = validateToken("", "secret")
	if err != errTokenMalformed {
		t.Fatalf("expected errTokenMalformed for empty token, got %v", err)
	}
}

func TestCSRFTokenGeneration(t *testing.T) {
	t1 := generateCSRFToken()
	t2 := generateCSRFToken()
	if t1 == t2 {
		t.Fatal("expected different CSRF tokens")
	}
	if len(t1) != 32 { // 16 bytes -> 32 hex chars
		t.Fatalf("expected 32 char CSRF token, got %d", len(t1))
	}
}

func TestJWTSecretGeneration(t *testing.T) {
	s1 := generateJWTSecret()
	s2 := generateJWTSecret()
	if s1 == s2 {
		t.Fatal("expected different JWT secrets")
	}
	if len(s1) < 40 { // 32 bytes base64 -> ~43 chars
		t.Fatalf("expected JWT secret >= 40 chars, got %d", len(s1))
	}
}

func TestIsMutatingMethod(t *testing.T) {
	mutating := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, m := range mutating {
		if !isMutatingMethod(m) {
			t.Errorf("expected %s to be mutating", m)
		}
	}
	nonMutating := []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	for _, m := range nonMutating {
		if isMutatingMethod(m) {
			t.Errorf("expected %s to NOT be mutating", m)
		}
	}
}

// TestLoginLogoutFlow tests the full login → CSRF → logout flow via HTTP test.
func TestLoginLogoutFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a config manager with auth enabled
	cfgPath := t.TempDir() + "/config.yaml"
	mgr, err := config.NewManager(cfgPath)
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}
	// Set up auth credentials and satisfy IsSetupComplete
	hash, _ := config.HashPassword("testpass")
	cfg := mgr.Get()
	cfg.Provider = "webdav"
	cfg.WebDAV.URL = "http://test"
	cfg.WebDAV.User = "u"
	cfg.WebDAV.Password = "p"
	cfg.Backup.RemoteDir = "/remote"
	cfg.Backup.LibraryDir = "/library"
	cfg.Server.AuthEnabled = true
	cfg.Server.AuthUser = "admin"
	cfg.Server.AuthPasswordHash = hash
	if err := mgr.Update(cfg); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	srv := &Server{
		Config:    mgr,
		authLimit: newAuthLimiter(),
		Build:     BuildInfo{Version: "test"},
	}

	router := gin.New()
	router.Use(srv.authMiddleware())
	v1 := router.Group("/api/v1")
	v1.POST("/auth/login", srv.handleAuthLogin)
	v1.POST("/auth/logout", srv.handleAuthLogout)
	v1.GET("/auth/csrf", srv.handleAuthCSRF)
	v1.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// 1. Login with correct credentials
	loginBody := `{"username":"admin","password":"testpass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var loginResp struct {
		CSRFToken string `json:"csrf_token"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("login response parse error: %v", err)
	}
	if loginResp.CSRFToken == "" {
		t.Fatal("expected non-empty csrf_token in login response")
	}

	// Extract cookie
	cookies := w.Result().Cookies()
	var jwtCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == jwtCookieName {
			jwtCookie = c
			break
		}
	}
	if jwtCookie == nil {
		t.Fatal("expected JWT cookie to be set")
	}

	// 2. Access protected endpoint with JWT cookie (GET - no CSRF needed)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req.AddCookie(jwtCookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ping with jwt: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Get CSRF token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/csrf", nil)
	req.AddCookie(jwtCookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("csrf: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Logout
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(jwtCookie)
	req.Header.Set(csrfHeaderName, loginResp.CSRFToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("logout: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLoginWrongPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfgPath := t.TempDir() + "/config.yaml"
	mgr, err := config.NewManager(cfgPath)
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}
	hash, _ := config.HashPassword("correct")
	cfg := mgr.Get()
	cfg.Server.AuthEnabled = true
	cfg.Server.AuthUser = "admin"
	cfg.Server.AuthPasswordHash = hash
	if err := mgr.Update(cfg); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	srv := &Server{
		Config:    mgr,
		authLimit: newAuthLimiter(),
		Build:     BuildInfo{Version: "test"},
	}

	router := gin.New()
	router.Use(srv.authMiddleware())
	v1 := router.Group("/api/v1")
	v1.POST("/auth/login", srv.handleAuthLogin)

	loginBody := `{"username":"admin","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCSRFRequiredForMutatingRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := generateJWTSecret()
	csrf := generateCSRFToken()
	token, _ := createToken("admin", csrf, secret, 1*time.Hour)

	cfgPath := t.TempDir() + "/config.yaml"
	mgr, err := config.NewManager(cfgPath)
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}
	hash, _ := config.HashPassword("pass")
	cfg := mgr.Get()
	cfg.Provider = "webdav"
	cfg.WebDAV.URL = "http://test"
	cfg.WebDAV.User = "u"
	cfg.WebDAV.Password = "p"
	cfg.Backup.RemoteDir = "/remote"
	cfg.Backup.LibraryDir = "/library"
	cfg.Server.AuthEnabled = true
	cfg.Server.AuthUser = "admin"
	cfg.Server.AuthPasswordHash = hash
	cfg.Server.JWTSecret = secret
	if err := mgr.Update(cfg); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	srv := &Server{
		Config:    mgr,
		authLimit: newAuthLimiter(),
		Build:     BuildInfo{Version: "test"},
	}

	router := gin.New()
	router.Use(srv.authMiddleware())
	router.POST("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// POST without CSRF header → 403
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: jwtCookieName, Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without CSRF, got %d: %s", w.Code, w.Body.String())
	}

	// POST with correct CSRF header → 200
	req = httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: jwtCookieName, Value: token})
	req.Header.Set(csrfHeaderName, csrf)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with CSRF, got %d: %s", w.Code, w.Body.String())
	}

	// POST with wrong CSRF header → 403
	req = httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: jwtCookieName, Value: token})
	req.Header.Set(csrfHeaderName, "wrong-csrf")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 with wrong CSRF, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetTokenFromRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenValue := "test-jwt-token-value"

	tests := []struct {
		name     string
		setup    func(r *http.Request)
		expected string
	}{
		{
			name: "from cookie",
			setup: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: jwtCookieName, Value: tokenValue})
			},
			expected: tokenValue,
		},
		{
			name: "from Bearer header",
			setup: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+tokenValue)
			},
			expected: tokenValue,
		},
		{
			name: "query param ignored",
			setup: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("token", tokenValue)
				r.URL.RawQuery = q.Encode()
			},
			expected: "",
		},
		{
			name:     "no token at all",
			setup:    func(r *http.Request) {},
			expected: "",
		},
		{
			name: "cookie takes precedence over Bearer",
			setup: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: jwtCookieName, Value: "cookie-token"})
				r.Header.Set("Authorization", "Bearer bearer-token")
			},
			expected: "cookie-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setup(c.Request)

			got := getTokenFromRequest(c)
			if got != tt.expected {
				t.Fatalf("getTokenFromRequest() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsSecureRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		setup    func(r *http.Request)
		expected bool
	}{
		{
			name:     "plain HTTP",
			setup:    func(r *http.Request) {},
			expected: false,
		},
		{
			name: "X-Forwarded-Proto https",
			setup: func(r *http.Request) {
				r.Header.Set("X-Forwarded-Proto", "https")
			},
			expected: true,
		},
		{
			name: "X-Forwarded-Proto http",
			setup: func(r *http.Request) {
				r.Header.Set("X-Forwarded-Proto", "http")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setup(c.Request)

			got := isSecureRequest(c)
			if got != tt.expected {
				t.Fatalf("isSecureRequest() = %v, want %v", got, tt.expected)
			}
		})
	}
}
