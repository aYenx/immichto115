package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aYenx/immichto115/internal/open115"
)

func TestNormalizeRemoteDir(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty becomes root", in: "", want: "/"},
		{name: "spaces become root", in: "   ", want: "/"},
		{name: "relative path gets leading slash", in: "immich-backup", want: "/immich-backup"},
		{name: "duplicate slashes are cleaned", in: "//albums///2026//", want: "/albums/2026"},
		{name: "dot path becomes root", in: ".", want: "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeRemoteDir(tt.in)
			if got != tt.want {
				t.Fatalf("normalizeRemoteDir(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsSetupWhitelistedOpen115(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/open115/auth/start", nil)
	if !isSetupWhitelisted(req) {
		t.Fatalf("expected open115 auth start to be whitelisted during setup")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/open115/auth/status?uid=test", nil)
	if !isSetupWhitelisted(req) {
		t.Fatalf("expected open115 auth status to be whitelisted during setup")
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/open115/auth/finish", nil)
	if !isSetupWhitelisted(req) {
		t.Fatalf("expected open115 auth finish to be whitelisted during setup")
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/open115/test", nil)
	if !isSetupWhitelisted(req) {
		t.Fatalf("expected open115 test to be whitelisted during setup")
	}

	// Directory listing endpoints needed by setup wizard
	wizardPaths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/local/ls"},
		{http.MethodPost, "/api/v1/open115/ls"},
	}
	for _, tc := range wizardPaths {
		req = httptest.NewRequest(tc.method, tc.path, nil)
		if !isSetupWhitelisted(req) {
			t.Fatalf("expected %s %s to be WHITELISTED during setup", tc.method, tc.path)
		}
	}

	// Remote/Restore explorer must NOT be whitelisted during setup
	blockedPaths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/remote/ls"},
	}
	for _, tc := range blockedPaths {
		req = httptest.NewRequest(tc.method, tc.path, nil)
		if isSetupWhitelisted(req) {
			t.Fatalf("expected %s %s to be BLOCKED during setup", tc.method, tc.path)
		}
	}
}

func TestOpen115AuthSessionStore(t *testing.T) {
	s := &Server{authSessions: make(map[string]*open115.AuthSession)}
	session := &open115.AuthSession{UID: "uid-1", CreatedAt: time.Now()}
	s.storeAuthSession(session)
	got, ok := s.loadAuthSession("uid-1")
	if !ok || got == nil || got.UID != "uid-1" {
		t.Fatalf("expected auth session to be stored and retrievable")
	}
	s.deleteAuthSession("uid-1")
	if _, ok := s.loadAuthSession("uid-1"); ok {
		t.Fatalf("expected auth session to be deleted")
	}
}
