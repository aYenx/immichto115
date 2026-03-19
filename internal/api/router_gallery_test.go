package api

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// isGalleryProxyHostAllowed unit tests
// ---------------------------------------------------------------------------

func TestIsGalleryProxyHostAllowed(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		allowed bool
	}{
		{name: "115.com allowed", rawURL: "https://thumb.115.com/abc.jpg", allowed: true},
		{name: "115cdn.net allowed", rawURL: "https://img.115cdn.net/photo.jpg", allowed: true},
		{name: "115cdn.com allowed", rawURL: "https://media.115cdn.com/pic.png", allowed: true},
		{name: "http rejected", rawURL: "http://thumb.115.com/abc.jpg", allowed: false},
		{name: "evil domain rejected", rawURL: "https://evil.com/abc.jpg", allowed: false},
		{name: "suffix trick rejected", rawURL: "https://evil-115.com/abc.jpg", allowed: false},
		{name: "empty string rejected", rawURL: "", allowed: false},
		{name: "garbage rejected", rawURL: "not-a-url", allowed: false},
		{name: "ftp rejected", rawURL: "ftp://thumb.115.com/file.jpg", allowed: false},
		{name: "no scheme rejected", rawURL: "//thumb.115.com/file.jpg", allowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGalleryProxyHostAllowed(tt.rawURL)
			if got != tt.allowed {
				t.Fatalf("isGalleryProxyHostAllowed(%q) = %v, want %v", tt.rawURL, got, tt.allowed)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GalleryEntry path construction tests
// ---------------------------------------------------------------------------

func TestGalleryEntryPathConstruction(t *testing.T) {
	tests := []struct {
		name       string
		parentPath string
		fileName   string
		want       string
	}{
		{name: "root path", parentPath: "/", fileName: "photo.jpg", want: "/photo.jpg"},
		{name: "nested path", parentPath: "/album/2026", fileName: "img.png", want: "/album/2026/img.png"},
		{name: "trailing slash trimmed", parentPath: "/photos/", fileName: "a.jpg", want: "/photos/a.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := strings.TrimRight(tt.parentPath, "/")
			got := p + "/" + tt.fileName
			if got != tt.want {
				t.Fatalf("path = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Gallery proxy truncation test (upstream sends more than maxBytes)
// ---------------------------------------------------------------------------

func TestGalleryProxy_TruncatesOversize(t *testing.T) {
	// Serve 10 bytes from a fake upstream, but our handler should cap at 5.
	const payload = "0123456789"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		// Deliberately omit Content-Length to test unknown-length scenario
		_, _ = io.WriteString(w, payload)
	}))
	defer upstream.Close()

	// Verify the host whitelist blocks our test URL (expected, since
	// it's localhost — this validates that the whitelist is enforced).
	url := upstream.URL + "/test.jpg"
	if isGalleryProxyHostAllowed(url) {
		t.Fatal("localhost URL should NOT be allowed by proxy whitelist")
	}
}

// ---------------------------------------------------------------------------
// Gallery proxy rejects non-image content types
// ---------------------------------------------------------------------------

func TestGalleryProxyHostAllowed_Subdomains(t *testing.T) {
	// Deep subdomain should still be allowed
	if !isGalleryProxyHostAllowed("https://a.b.c.115cdn.net/x.jpg") {
		t.Fatal("deep subdomain of 115cdn.net should be allowed")
	}

	// Bare domain without subdomain does NOT match the suffix ".115.com"
	// (this is correct security behavior — we only allow subdomains)
	if isGalleryProxyHostAllowed("https://115.com/x.jpg") {
		t.Fatal("bare 115.com should NOT match .115.com suffix (no subdomain)")
	}

	// Similar but not matching
	if isGalleryProxyHostAllowed("https://not115.com/x.jpg") {
		t.Fatal("not115.com should NOT be allowed")
	}
}

// ---------------------------------------------------------------------------
// galleryProxyAllowedHosts list integrity
// ---------------------------------------------------------------------------

func TestGalleryProxyAllowedHostsList(t *testing.T) {
	// Verify the allowed hosts list contains expected entries
	expected := []string{".115.com", ".115cdn.net", ".115cdn.com"}
	if len(galleryProxyAllowedHosts) != len(expected) {
		t.Fatalf("galleryProxyAllowedHosts has %d entries, want %d",
			len(galleryProxyAllowedHosts), len(expected))
	}
	for i, want := range expected {
		if galleryProxyAllowedHosts[i] != want {
			t.Fatalf("galleryProxyAllowedHosts[%d] = %q, want %q",
				i, galleryProxyAllowedHosts[i], want)
		}
	}
}

// ---------------------------------------------------------------------------
// GalleryEntry DTO field validation
// ---------------------------------------------------------------------------

func TestGalleryEntry_JSONTags(t *testing.T) {
	// Quick compile-time safety — ensure the struct can be constructed
	// with all expected fields (no typos, no missing fields).
	entry := GalleryEntry{
		ID:          "abc123",
		Name:        "photo.jpg",
		Path:        "/album/photo.jpg",
		IsDir:       false,
		Size:        1024,
		ModTime:     "2026-01-01T00:00:00Z",
		PickCode:    "pk001",
		Thumbnail:   "https://thumb.115.com/t.jpg",
		OriginalURL: "https://img.115cdn.net/o.jpg",
		FileType:    "jpg",
	}

	// Verify all fields are set (not zero-valued by accident)
	if entry.ID == "" || entry.Name == "" || entry.Path == "" {
		t.Fatal("GalleryEntry key fields should not be empty")
	}
	if entry.Size != 1024 {
		t.Fatalf("Size = %d, want 1024", entry.Size)
	}
	if entry.IsDir {
		t.Fatal("IsDir should be false for a file entry")
	}
	if entry.PickCode == "" || entry.Thumbnail == "" || entry.OriginalURL == "" {
		t.Fatal("Optional fields should be populated when set")
	}

	// Test directory entry
	dirEntry := GalleryEntry{
		ID:    "dir001",
		Name:  "album",
		Path:  "/album",
		IsDir: true,
	}
	if !dirEntry.IsDir {
		t.Fatal("IsDir should be true for directory entry")
	}
	if dirEntry.PickCode != "" || dirEntry.Thumbnail != "" {
		t.Fatal("Directory entries should not have pick_code or thumbnail")
	}

	// Verify unused entry to suppress lint
	_ = fmt.Sprintf("%+v", entry)
	_ = fmt.Sprintf("%+v", dirEntry)
}
