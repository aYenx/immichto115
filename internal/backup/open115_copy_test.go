package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aYenx/immichto115/internal/manifest"
)

func TestScanLocalFilesMissingDirReturnsEmpty(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-exists")
	items, err := scanLocalFiles(missing, "library")
	if err != nil {
		t.Fatalf("scanLocalFiles error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items for missing dir, got %d", len(items))
	}
}

func TestScanLocalFilesAppliesPrefixAndFindsFiles(t *testing.T) {
	base := filepath.Join(t.TempDir(), "library")
	if err := os.MkdirAll(filepath.Join(base, "album1"), 0o755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	filePath := filepath.Join(base, "album1", "hello.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	items, err := scanLocalFiles(base, "library")
	if err != nil {
		t.Fatalf("scanLocalFiles error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].RelPath != "library/album1/hello.txt" {
		t.Fatalf("unexpected RelPath: %q", items[0].RelPath)
	}
	if items[0].AbsPath != filePath {
		t.Fatalf("unexpected AbsPath: %q", items[0].AbsPath)
	}
	if items[0].Size != 5 {
		t.Fatalf("unexpected Size: %d", items[0].Size)
	}
}

type listStubStore struct {
	items []manifest.FileRecord
	calls []struct{ limit, offset int }
}

func (s *listStubStore) Init(ctx context.Context) error                                 { return nil }
func (s *listStubStore) Get(ctx context.Context, path string) (*manifest.FileRecord, error) { return nil, nil }
func (s *listStubStore) Put(ctx context.Context, record *manifest.FileRecord) error     { return nil }
func (s *listStubStore) MarkDeleted(ctx context.Context, path string, deleted bool) error { return nil }
func (s *listStubStore) Delete(ctx context.Context, path string) error                  { return nil }
func (s *listStubStore) Close() error                                                   { return nil }
func (s *listStubStore) List(ctx context.Context, limit int, offset int) ([]manifest.FileRecord, error) {
	s.calls = append(s.calls, struct{ limit, offset int }{limit: limit, offset: offset})
	if offset >= len(s.items) {
		return nil, nil
	}
	end := offset + limit
	if end > len(s.items) {
		end = len(s.items)
	}
	return s.items[offset:end], nil
}

func TestListAllManifestRecordsPaginatesUntilComplete(t *testing.T) {
	const total = 2505
	items := make([]manifest.FileRecord, 0, total)
	for i := 0; i < total; i++ {
		items = append(items, manifest.FileRecord{Path: fmt.Sprintf("file-%04d", i)})
	}
	store := &listStubStore{items: items}
	runner := &Open115CopyRunner{manifest: store}

	got, err := runner.listAllManifestRecords(context.Background())
	if err != nil {
		t.Fatalf("listAllManifestRecords error: %v", err)
	}
	if len(got) != total {
		t.Fatalf("expected %d records, got %d", total, len(got))
	}
	if len(store.calls) != 3 {
		t.Fatalf("expected 3 paged List calls, got %d", len(store.calls))
	}
	if store.calls[0].offset != 0 || store.calls[1].offset != 1000 || store.calls[2].offset != 2000 {
		t.Fatalf("unexpected pagination offsets: %+v", store.calls)
	}
}
