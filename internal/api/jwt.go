package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// --- JWT constants ---

const (
	jwtCookieName   = "immichto115_token"
	jwtDefaultTTL   = 24 * time.Hour
	csrfHeaderName  = "X-CSRF-Token"
)

// --- Errors ---

var (
	errTokenExpired  = errors.New("token expired")
	errTokenInvalid  = errors.New("invalid token")
	errTokenMalformed = errors.New("malformed token")
)

// --- JWT Claims ---

// JWTClaims represents the payload embedded in a JWT token.
type JWTClaims struct {
	Sub  string `json:"sub"`            // username
	CSRF string `json:"csrf,omitempty"` // CSRF protection token
	Exp  int64  `json:"exp"`            // expiration (Unix seconds)
	Iat  int64  `json:"iat"`            // issued at (Unix seconds)
}

// --- Secret generation ---

// generateJWTSecret creates a 32-byte random secret encoded as base64.
func generateJWTSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate JWT secret: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// generateCSRFToken creates a 16-byte random token encoded as hex.
func generateCSRFToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate CSRF token: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// --- JWT creation / validation (HS256, stdlib) ---

// createToken generates an HS256 JWT token string.
func createToken(username, csrfToken, secret string, expiry time.Duration) (string, error) {
	header := base64Encode([]byte(`{"alg":"HS256","typ":"JWT"}`))

	now := time.Now()
	claims := JWTClaims{
		Sub:  username,
		CSRF: csrfToken,
		Exp:  now.Add(expiry).Unix(),
		Iat:  now.Unix(),
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	payload := base64Encode(claimsJSON)

	sigInput := header + "." + payload
	sig := signHS256([]byte(sigInput), []byte(secret))

	return sigInput + "." + sig, nil
}

// validateToken parses and validates an HS256 JWT token string.
func validateToken(tokenString, secret string) (*JWTClaims, error) {
	parts := strings.SplitN(tokenString, ".", 3)
	if len(parts) != 3 {
		return nil, errTokenMalformed
	}

	// Verify signature
	sigInput := parts[0] + "." + parts[1]
	expectedSig := signHS256([]byte(sigInput), []byte(secret))
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, errTokenInvalid
	}

	// Decode claims
	claimsJSON, err := base64Decode(parts[1])
	if err != nil {
		return nil, errTokenMalformed
	}
	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, errTokenMalformed
	}

	// Check expiry
	if time.Now().Unix() > claims.Exp {
		return nil, errTokenExpired
	}

	return &claims, nil
}

// --- Cookie helpers ---

// setTokenCookie sets the JWT token as an HttpOnly cookie.
func isSecureRequest(c *gin.Context) bool {
	return c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
}

func setTokenCookie(c *gin.Context, token string, maxAge int) {
	c.SetCookie(
		jwtCookieName,
		token,
		maxAge,
		"/",
		"",                    // domain: auto
		isSecureRequest(c),    // secure: follows request protocol
		true,                  // httpOnly
	)
}

// clearTokenCookie removes the JWT cookie.
func clearTokenCookie(c *gin.Context) {
	c.SetCookie(jwtCookieName, "", -1, "/", "", isSecureRequest(c), true)
}

// getTokenFromRequest extracts a JWT token from (in order):
// 1. Cookie
// 2. Authorization: Bearer <token> header
// URL query params are intentionally NOT supported to prevent token
// leakage via browser history, referrer headers, and proxy logs.
func getTokenFromRequest(c *gin.Context) string {
	// 1. Cookie
	if cookie, err := c.Cookie(jwtCookieName); err == nil && cookie != "" {
		return cookie
	}
	// 2. Bearer token
	if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// --- Internal helpers ---

func base64Encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64Decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

func signHS256(data, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return base64Encode(mac.Sum(nil))
}

// isMutatingMethod returns true for HTTP methods that modify state.
func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}
