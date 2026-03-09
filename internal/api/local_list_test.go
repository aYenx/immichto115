package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleLocalListReadsFilesystemDirectly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "library"), 0o755); err != nil {
		t.Fatalf("mkdir library: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write hello.txt: %v", err)
	}

	s := &Server{}
	r := gin.New()
	r.GET("/api/v1/local/ls", s.handleLocalList)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/local/ls?path="+root, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var items []struct {
		Path  string
		Name  string
		IsDir bool
	}
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal response: %v, body = %s", err, w.Body.String())
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d: %s", len(items), w.Body.String())
	}

	seenDir := false
	seenFile := false
	for _, item := range items {
		switch item.Name {
		case "library":
			if !item.IsDir {
				t.Fatalf("library should be dir: %+v", item)
			}
			seenDir = true
		case "hello.txt":
			if item.IsDir {
				t.Fatalf("hello.txt should be file: %+v", item)
			}
			seenFile = true
		}
	}
	if !seenDir || !seenFile {
		t.Fatalf("missing expected entries, dir=%v file=%v, body=%s", seenDir, seenFile, w.Body.String())
	}
}

func TestHandleLocalListAllowsDotDotInLegitNameButRejectsParentSegment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	root := t.TempDir()
	legitDir := filepath.Join(root, "a..b")
	if err := os.Mkdir(legitDir, 0o755); err != nil {
		t.Fatalf("mkdir legitDir: %v", err)
	}

	s := &Server{}
	r := gin.New()
	r.GET("/api/v1/local/ls", s.handleLocalList)

	goodReq := httptest.NewRequest(http.MethodGet, "/api/v1/local/ls?path="+legitDir, nil)
	goodW := httptest.NewRecorder()
	r.ServeHTTP(goodW, goodReq)
	if goodW.Code != http.StatusOK {
		t.Fatalf("good path status = %d, body = %s", goodW.Code, goodW.Body.String())
	}

	badReq := httptest.NewRequest(http.MethodGet, "/api/v1/local/ls?path="+root+"/../secret", nil)
	badW := httptest.NewRecorder()
	r.ServeHTTP(badW, badReq)
	if badW.Code != http.StatusBadRequest {
		t.Fatalf("bad path status = %d, body = %s", badW.Code, badW.Body.String())
	}
}
