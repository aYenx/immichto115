package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aYenx/immichto115/internal/manifest"
)

func TestScanLocalFilesMissingDirReturnsIncomplete(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-exists")
	result, err := scanLocalFiles(missing, "library")
	if err != nil {
		t.Fatalf("scanLocalFiles error: %v", err)
	}
	if len(result.Files) != 0 {
		t.Fatalf("expected 0 files for missing dir, got %d", len(result.Files))
	}
	if result.Complete {
		t.Fatal("expected Complete=false for missing dir")
	}
	if result.Skipped != 1 {
		t.Fatalf("expected 1 skipped for missing dir, got %d", result.Skipped)
	}
	if result.MissingDir != missing {
		t.Fatalf("expected MissingDir=%q, got %q", missing, result.MissingDir)
	}
}

func TestScanLocalFilesEmptyDirIsComplete(t *testing.T) {
	result, err := scanLocalFiles("", "library")
	if err != nil {
		t.Fatalf("scanLocalFiles error: %v", err)
	}
	if !result.Complete {
		t.Fatal("expected Complete=true for empty dir config")
	}
	if len(result.Files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(result.Files))
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

	result, err := scanLocalFiles(base, "library")
	if err != nil {
		t.Fatalf("scanLocalFiles error: %v", err)
	}
	if !result.Complete {
		t.Fatal("expected Complete=true")
	}
	if result.Skipped != 0 {
		t.Fatalf("expected 0 skipped, got %d", result.Skipped)
	}
	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result.Files))
	}
	if result.Files[0].RelPath != "library/album1/hello.txt" {
		t.Fatalf("unexpected RelPath: %q", result.Files[0].RelPath)
	}
	if result.Files[0].AbsPath != filePath {
		t.Fatalf("unexpected AbsPath: %q", result.Files[0].AbsPath)
	}
	if result.Files[0].Size != 5 {
		t.Fatalf("unexpected Size: %d", result.Files[0].Size)
	}
}

type listStubStore struct {
	items          []manifest.FileRecord
	calls          []struct{ limit, offset int }
	markDeleted    []string
	pendingDeletes map[string]int64 // path -> pendingAt
}

func (s *listStubStore) Init(ctx context.Context) error                                     { return nil }
func (s *listStubStore) Get(ctx context.Context, path string) (*manifest.FileRecord, error) { return nil, nil }
func (s *listStubStore) Put(ctx context.Context, record *manifest.FileRecord) error         { return nil }
func (s *listStubStore) MarkDeleted(ctx context.Context, path string, deleted bool) error {
	s.markDeleted = append(s.markDeleted, path)
	return nil
}
func (s *listStubStore) Delete(ctx context.Context, path string) error { return nil }
func (s *listStubStore) MarkPendingDelete(ctx context.Context, path string, pendingAt int64) error {
	if s.pendingDeletes == nil {
		s.pendingDeletes = make(map[string]int64)
	}
	s.pendingDeletes[path] = pendingAt
	// Also update the in-memory items
	for i := range s.items {
		if s.items[i].Path == path {
			s.items[i].PendingDeleteAt = pendingAt
		}
	}
	return nil
}
func (s *listStubStore) ClearPendingDelete(ctx context.Context, path string) error {
	delete(s.pendingDeletes, path)
	for i := range s.items {
		if s.items[i].Path == path {
			s.items[i].PendingDeleteAt = 0
		}
	}
	return nil
}
func (s *listStubStore) ListPendingDeletes(ctx context.Context, olderThan int64) ([]manifest.FileRecord, error) {
	var result []manifest.FileRecord
	for _, rec := range s.items {
		if rec.PendingDeleteAt > 0 && rec.PendingDeleteAt <= olderThan && !rec.Deleted {
			result = append(result, rec)
		}
	}
	return result, nil
}
func (s *listStubStore) Close() error { return nil }
func (s *listStubStore) CountActive(ctx context.Context) (int, error) {
	count := 0
	for _, rec := range s.items {
		if !rec.Deleted {
			count++
		}
	}
	return count, nil
}
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

func TestSyncDeleteBlockedByThreshold(t *testing.T) {
	// Create 10 active manifest records, but only 5 local files exist.
	// Candidates = 5/10 = 50% > 20% threshold → should be blocked.
	items := make([]manifest.FileRecord, 10)
	for i := 0; i < 10; i++ {
		items[i] = manifest.FileRecord{Path: fmt.Sprintf("file-%02d", i)}
	}
	store := &listStubStore{items: items}
	runner := &Open115CopyRunner{manifest: store}

	// Only 5 of the 10 files exist locally
	localFiles := make([]localFile, 5)
	for i := 0; i < 5; i++ {
		localFiles[i] = localFile{RelPath: fmt.Sprintf("file-%02d", i)}
	}

	result, err := runner.syncDeleteRemoved(context.Background(), localFiles, "/backup", syncDeleteOpts{GracePeriod: 0})
	if err != nil {
		t.Fatalf("syncDeleteRemoved error: %v", err)
	}
	if !result.Skipped {
		t.Fatal("expected delete to be skipped due to threshold")
	}
	if result.Deleted != 0 {
		t.Fatalf("expected 0 deleted, got %d", result.Deleted)
	}
	if len(store.markDeleted) != 0 {
		t.Fatalf("expected no MarkDeleted calls, got %d", len(store.markDeleted))
	}
}

func TestSyncDeleteAllowedBelowThreshold(t *testing.T) {
	// Create 10 active manifest records, 9 local files exist.
	// Candidates = 1/10 = 10% < 20% threshold → should NOT be threshold-skipped.
	items := make([]manifest.FileRecord, 10)
	for i := 0; i < 10; i++ {
		items[i] = manifest.FileRecord{Path: fmt.Sprintf("file-%02d", i)}
	}
	store := &listStubStore{items: items}
	runner := &Open115CopyRunner{manifest: store}

	// 9 of the 10 files exist locally → 1 candidate (10%)
	localFiles := make([]localFile, 9)
	for i := 0; i < 9; i++ {
		localFiles[i] = localFile{RelPath: fmt.Sprintf("file-%02d", i)}
	}

	// The runner has no backend, so the actual delete will panic.
	// We recover from the panic and just verify the threshold was not applied.
	var result syncDeleteResult
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		var err error
		result, err = runner.syncDeleteRemoved(context.Background(), localFiles, "/backup", syncDeleteOpts{GracePeriod: 0})
		if err != nil {
			t.Logf("Expected error from missing backend: %v", err)
		}
	}()

	// The key assertion: with two-phase delete and 0 grace period,
	// first run marks pending, then phase 2 tries to delete.
	// Since there's no backend, the actual delete will panic.
	if !panicked && result.Skipped {
		t.Fatal("expected delete NOT to be skipped (below threshold)")
	}
}

func TestSyncDeleteTwoPhaseGracePeriod(t *testing.T) {
	// 10 records, 9 local files → 1 candidate (10%), below threshold
	items := make([]manifest.FileRecord, 10)
	for i := 0; i < 10; i++ {
		items[i] = manifest.FileRecord{Path: fmt.Sprintf("file-%02d", i)}
	}
	store := &listStubStore{items: items}
	runner := &Open115CopyRunner{manifest: store}

	// 9 of 10 files exist → file-09 is candidate
	localFiles := make([]localFile, 9)
	for i := 0; i < 9; i++ {
		localFiles[i] = localFile{RelPath: fmt.Sprintf("file-%02d", i)}
	}

	// Run with 1 hour grace period → should only mark pending, not delete
	result, err := runner.syncDeleteRemoved(context.Background(), localFiles, "/backup", syncDeleteOpts{GracePeriod: 1 * time.Hour})
	if err != nil {
		t.Fatalf("syncDeleteRemoved error: %v", err)
	}
	if result.PendingMarked != 1 {
		t.Fatalf("expected 1 pending marked, got %d", result.PendingMarked)
	}
	if result.Deleted != 0 {
		t.Fatalf("expected 0 deleted (grace period not expired), got %d", result.Deleted)
	}
	if len(store.markDeleted) != 0 {
		t.Fatalf("expected no MarkDeleted calls for hard delete, got %d", len(store.markDeleted))
	}

	// Verify the item was marked in the store
	if store.pendingDeletes == nil || store.pendingDeletes["file-09"] == 0 {
		t.Fatal("expected file-09 to be in pendingDeletes")
	}
}

func TestSyncDeleteClearsPendingOnReappear(t *testing.T) {
	// file-09 was previously marked pending
	items := make([]manifest.FileRecord, 10)
	for i := 0; i < 10; i++ {
		items[i] = manifest.FileRecord{Path: fmt.Sprintf("file-%02d", i)}
	}
	items[9].PendingDeleteAt = time.Now().Add(-2 * time.Hour).Unix()
	store := &listStubStore{
		items:          items,
		pendingDeletes: map[string]int64{"file-09": items[9].PendingDeleteAt},
	}
	runner := &Open115CopyRunner{manifest: store}

	// All 10 files exist now (file-09 reappeared)
	localFiles := make([]localFile, 10)
	for i := 0; i < 10; i++ {
		localFiles[i] = localFile{RelPath: fmt.Sprintf("file-%02d", i)}
	}

	result, err := runner.syncDeleteRemoved(context.Background(), localFiles, "/backup", syncDeleteOpts{GracePeriod: 1 * time.Hour})
	if err != nil {
		t.Fatalf("syncDeleteRemoved error: %v", err)
	}
	if result.PendingMarked != 0 {
		t.Fatalf("expected 0 pending marked, got %d", result.PendingMarked)
	}
	// file-09 should have been cleared from pending
	if _, ok := store.pendingDeletes["file-09"]; ok {
		t.Fatal("expected file-09 to be cleared from pendingDeletes")
	}
}

func TestSyncDeleteDryRun(t *testing.T) {
	// file-09 was pending and grace period expired
	items := make([]manifest.FileRecord, 10)
	for i := 0; i < 10; i++ {
		items[i] = manifest.FileRecord{Path: fmt.Sprintf("file-%02d", i)}
	}
	items[9].PendingDeleteAt = time.Now().Add(-2 * time.Hour).Unix()
	store := &listStubStore{
		items:          items,
		pendingDeletes: map[string]int64{"file-09": items[9].PendingDeleteAt},
	}
	runner := &Open115CopyRunner{manifest: store}

	// 9 of 10 files exist → file-09 still missing
	localFiles := make([]localFile, 9)
	for i := 0; i < 9; i++ {
		localFiles[i] = localFile{RelPath: fmt.Sprintf("file-%02d", i)}
	}

	result, err := runner.syncDeleteRemoved(context.Background(), localFiles, "/backup", syncDeleteOpts{GracePeriod: 1 * time.Hour, DryRun: true})
	if err != nil {
		t.Fatalf("syncDeleteRemoved error: %v", err)
	}
	if !result.DryRun {
		t.Fatal("expected DryRun=true")
	}
	if result.Deleted != 1 {
		t.Fatalf("expected 1 dry-run deleted, got %d", result.Deleted)
	}
	// No actual MarkDeleted calls should have happened in dry-run
	if len(store.markDeleted) != 0 {
		t.Fatalf("expected no MarkDeleted calls in dry-run, got %d", len(store.markDeleted))
	}
}

func TestParseGracePeriod(t *testing.T) {
	cases := []struct {
		input    string
		expected time.Duration
	}{
		{"", 24 * time.Hour},
		{"24h", 24 * time.Hour},
		{"1h", 1 * time.Hour},
		{"30m", 30 * time.Minute},
		{"invalid", 24 * time.Hour},
		{"-1h", 24 * time.Hour},
	}
	for _, tc := range cases {
		got := parseGracePeriod(tc.input)
		if got != tc.expected {
			t.Errorf("parseGracePeriod(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}
