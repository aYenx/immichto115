package api

import (
	"errors"
	"testing"
)

func TestSanitizeError_Nil(t *testing.T) {
	if got := sanitizeError(nil); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestSanitizeError_URLCredentials(t *testing.T) {
	err := errors.New("dial tcp: connect to https://admin:s3cret@192.168.1.10:443 failed")
	got := sanitizeError(err)
	if contains(got, "s3cret") || contains(got, "admin:") {
		t.Errorf("credentials leaked: %s", got)
	}
	if !contains(got, "***@") {
		t.Errorf("expected redacted URL, got: %s", got)
	}
}

func TestSanitizeError_AbsoluteWindowsPath(t *testing.T) {
	err := errors.New(`open C:\Users\admin\AppData\config\immichto115.yaml: permission denied`)
	got := sanitizeError(err)
	if contains(got, `C:\Users\admin`) {
		t.Errorf("Windows path leaked: %s", got)
	}
	if !contains(got, "immichto115.yaml") {
		t.Errorf("should keep basename, got: %s", got)
	}
}

func TestSanitizeError_AbsoluteUnixPath(t *testing.T) {
	err := errors.New("open /home/deploy/.config/immichto115/config.yaml: no such file")
	got := sanitizeError(err)
	if contains(got, "/home/deploy") {
		t.Errorf("Unix path leaked: %s", got)
	}
	if !contains(got, "config.yaml") {
		t.Errorf("should keep basename, got: %s", got)
	}
}

func TestSanitizeError_StackTrace(t *testing.T) {
	err := errors.New("panic: runtime error\ngoroutine 42 [running]:\nmain.go:123")
	got := sanitizeError(err)
	if contains(got, "goroutine 42") {
		t.Errorf("stack trace leaked: %s", got)
	}
}

func TestSanitizeError_SafeMessage(t *testing.T) {
	msg := "WebDAV 连接超时，请检查网络"
	err := errors.New(msg)
	got := sanitizeError(err)
	if got != msg {
		t.Errorf("safe message changed: got %q, want %q", got, msg)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && containsImpl(s, sub)
}

func containsImpl(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
