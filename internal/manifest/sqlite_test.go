package manifest

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStoreCRUD(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "manifest.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore error: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.Init(ctx); err != nil {
		t.Fatalf("Init error: %v", err)
	}

	rec := &FileRecord{
		Path:           "library/album1/hello.txt",
		Size:           123,
		MTime:          time.Now().Unix(),
		SHA1:           "ABC",
		PreID:          "DEF",
		RemoteFileID:   "fid-1",
		RemotePickCode: "pick-1",
		LastUploadedAt: time.Now().Unix(),
		Deleted:        false,
	}

	if err := store.Put(ctx, rec); err != nil {
		t.Fatalf("Put error: %v", err)
	}

	got, err := store.Get(ctx, rec.Path)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got == nil {
		t.Fatalf("expected record, got nil")
	}
	if got.Path != rec.Path || got.Size != rec.Size || got.SHA1 != rec.SHA1 {
		t.Fatalf("unexpected record: %+v", got)
	}

	if err := store.MarkDeleted(ctx, rec.Path, true); err != nil {
		t.Fatalf("MarkDeleted error: %v", err)
	}
	got, err = store.Get(ctx, rec.Path)
	if err != nil {
		t.Fatalf("Get after MarkDeleted error: %v", err)
	}
	if got == nil || !got.Deleted {
		t.Fatalf("expected deleted=true, got %+v", got)
	}

	items, err := store.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if err := store.Delete(ctx, rec.Path); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	got, err = store.Get(ctx, rec.Path)
	if err != nil {
		t.Fatalf("Get after Delete error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil after delete, got %+v", got)
	}
}
