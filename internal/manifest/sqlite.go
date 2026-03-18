package manifest

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, fmt.Errorf("manifest sqlite dsn 不能为空")
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Init(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("manifest sqlite store 未初始化")
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS files (
			path TEXT PRIMARY KEY,
			size INTEGER NOT NULL,
			mtime INTEGER NOT NULL,
			sha1 TEXT,
			preid TEXT,
			remote_file_id TEXT,
			remote_pick_code TEXT,
			last_uploaded_at INTEGER NOT NULL,
			deleted INTEGER DEFAULT 0,
			encrypted INTEGER DEFAULT 0,
			encrypted_size INTEGER,
			remote_path TEXT,
			encryption_version TEXT,
			content_sha256 TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_manifest_files_uploaded_at ON files(last_uploaded_at)`,
		`CREATE INDEX IF NOT EXISTS idx_manifest_files_deleted ON files(deleted, path)`,
	}
	alterStmts := []string{
		`ALTER TABLE files ADD COLUMN encrypted INTEGER DEFAULT 0`,
		`ALTER TABLE files ADD COLUMN encrypted_size INTEGER`,
		`ALTER TABLE files ADD COLUMN remote_path TEXT`,
		`ALTER TABLE files ADD COLUMN encryption_version TEXT`,
		`ALTER TABLE files ADD COLUMN content_sha256 TEXT`,
		`ALTER TABLE files ADD COLUMN pending_delete_at INTEGER DEFAULT 0`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	for _, stmt := range alterStmts {
		_, _ = s.db.ExecContext(ctx, stmt)
	}
	return nil
}

func (s *SQLiteStore) Get(ctx context.Context, path string) (*FileRecord, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("manifest sqlite store 未初始化")
	}
	row := s.db.QueryRowContext(ctx, `SELECT path, size, mtime, sha1, preid, remote_file_id, remote_pick_code, last_uploaded_at, deleted, encrypted, encrypted_size, remote_path, encryption_version, content_sha256, COALESCE(pending_delete_at,0) FROM files WHERE path = ? LIMIT 1`, path)
	var rec FileRecord
	var deleted int
	var encrypted int
	if err := row.Scan(&rec.Path, &rec.Size, &rec.MTime, &rec.SHA1, &rec.PreID, &rec.RemoteFileID, &rec.RemotePickCode, &rec.LastUploadedAt, &deleted, &encrypted, &rec.EncryptedSize, &rec.RemotePath, &rec.EncryptionVersion, &rec.ContentSHA256, &rec.PendingDeleteAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	rec.Deleted = deleted != 0
	rec.Encrypted = encrypted != 0
	return &rec, nil
}

func (s *SQLiteStore) Put(ctx context.Context, record *FileRecord) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("manifest sqlite store 未初始化")
	}
	if record == nil || strings.TrimSpace(record.Path) == "" {
		return fmt.Errorf("manifest record 无效")
	}
	deleted := 0
	if record.Deleted {
		deleted = 1
	}
	encrypted := 0
	if record.Encrypted {
		encrypted = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO files (path, size, mtime, sha1, preid, remote_file_id, remote_pick_code, last_uploaded_at, deleted, encrypted, encrypted_size, remote_path, encryption_version, content_sha256)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			size = excluded.size,
			mtime = excluded.mtime,
			sha1 = excluded.sha1,
			preid = excluded.preid,
			remote_file_id = excluded.remote_file_id,
			remote_pick_code = excluded.remote_pick_code,
			last_uploaded_at = excluded.last_uploaded_at,
			deleted = excluded.deleted,
			encrypted = excluded.encrypted,
			encrypted_size = excluded.encrypted_size,
			remote_path = excluded.remote_path,
			encryption_version = excluded.encryption_version,
			content_sha256 = excluded.content_sha256
	`, record.Path, record.Size, record.MTime, record.SHA1, record.PreID, record.RemoteFileID, record.RemotePickCode, record.LastUploadedAt, deleted, encrypted, record.EncryptedSize, record.RemotePath, record.EncryptionVersion, record.ContentSHA256)
	return err
}

func (s *SQLiteStore) MarkDeleted(ctx context.Context, path string, deleted bool) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("manifest sqlite store 未初始化")
	}
	value := 0
	if deleted {
		value = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE files SET deleted = ? WHERE path = ?`, value, path)
	return err
}

func (s *SQLiteStore) Delete(ctx context.Context, path string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("manifest sqlite store 未初始化")
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM files WHERE path = ?`, path)
	return err
}

func (s *SQLiteStore) List(ctx context.Context, limit int, offset int) ([]FileRecord, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("manifest sqlite store 未初始化")
	}
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, `SELECT path, size, mtime, sha1, preid, remote_file_id, remote_pick_code, last_uploaded_at, deleted, encrypted, encrypted_size, remote_path, encryption_version, content_sha256, COALESCE(pending_delete_at,0) FROM files ORDER BY path ASC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]FileRecord, 0)
	for rows.Next() {
		var rec FileRecord
		var deleted int
		var encrypted int
		if err := rows.Scan(&rec.Path, &rec.Size, &rec.MTime, &rec.SHA1, &rec.PreID, &rec.RemoteFileID, &rec.RemotePickCode, &rec.LastUploadedAt, &deleted, &encrypted, &rec.EncryptedSize, &rec.RemotePath, &rec.EncryptionVersion, &rec.ContentSHA256, &rec.PendingDeleteAt); err != nil {
			return nil, err
		}
		rec.Deleted = deleted != 0
		rec.Encrypted = encrypted != 0
		items = append(items, rec)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) CountActive(ctx context.Context) (int, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("manifest sqlite store 未初始化")
	}
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM files WHERE deleted = 0`).Scan(&count)
	return count, err
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStore) MarkPendingDelete(ctx context.Context, path string, pendingAt int64) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("manifest sqlite store 未初始化")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE files SET pending_delete_at = ? WHERE path = ?`, pendingAt, path)
	return err
}

func (s *SQLiteStore) ClearPendingDelete(ctx context.Context, path string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("manifest sqlite store 未初始化")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE files SET pending_delete_at = 0 WHERE path = ?`, path)
	return err
}

func (s *SQLiteStore) ListPendingDeletes(ctx context.Context, olderThan int64) ([]FileRecord, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("manifest sqlite store 未初始化")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT path, size, mtime, sha1, preid, remote_file_id, remote_pick_code, last_uploaded_at, deleted, encrypted, encrypted_size, remote_path, encryption_version, content_sha256, COALESCE(pending_delete_at,0) FROM files WHERE pending_delete_at > 0 AND pending_delete_at <= ? AND deleted = 0 ORDER BY path ASC`, olderThan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]FileRecord, 0)
	for rows.Next() {
		var rec FileRecord
		var deleted int
		var encrypted int
		if err := rows.Scan(&rec.Path, &rec.Size, &rec.MTime, &rec.SHA1, &rec.PreID, &rec.RemoteFileID, &rec.RemotePickCode, &rec.LastUploadedAt, &deleted, &encrypted, &rec.EncryptedSize, &rec.RemotePath, &rec.EncryptionVersion, &rec.ContentSHA256, &rec.PendingDeleteAt); err != nil {
			return nil, err
		}
		rec.Deleted = deleted != 0
		rec.Encrypted = encrypted != 0
		items = append(items, rec)
	}
	return items, rows.Err()
}
