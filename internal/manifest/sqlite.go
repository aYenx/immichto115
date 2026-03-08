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
			deleted INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_manifest_files_uploaded_at ON files(last_uploaded_at)`,
		`CREATE INDEX IF NOT EXISTS idx_manifest_files_deleted ON files(deleted, path)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) Get(ctx context.Context, path string) (*FileRecord, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("manifest sqlite store 未初始化")
	}
	row := s.db.QueryRowContext(ctx, `SELECT path, size, mtime, sha1, preid, remote_file_id, remote_pick_code, last_uploaded_at, deleted FROM files WHERE path = ? LIMIT 1`, path)
	var rec FileRecord
	var deleted int
	if err := row.Scan(&rec.Path, &rec.Size, &rec.MTime, &rec.SHA1, &rec.PreID, &rec.RemoteFileID, &rec.RemotePickCode, &rec.LastUploadedAt, &deleted); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	rec.Deleted = deleted != 0
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
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO files (path, size, mtime, sha1, preid, remote_file_id, remote_pick_code, last_uploaded_at, deleted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			size = excluded.size,
			mtime = excluded.mtime,
			sha1 = excluded.sha1,
			preid = excluded.preid,
			remote_file_id = excluded.remote_file_id,
			remote_pick_code = excluded.remote_pick_code,
			last_uploaded_at = excluded.last_uploaded_at,
			deleted = excluded.deleted
	`, record.Path, record.Size, record.MTime, record.SHA1, record.PreID, record.RemoteFileID, record.RemotePickCode, record.LastUploadedAt, deleted)
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
	rows, err := s.db.QueryContext(ctx, `SELECT path, size, mtime, sha1, preid, remote_file_id, remote_pick_code, last_uploaded_at, deleted FROM files ORDER BY path ASC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]FileRecord, 0)
	for rows.Next() {
		var rec FileRecord
		var deleted int
		if err := rows.Scan(&rec.Path, &rec.Size, &rec.MTime, &rec.SHA1, &rec.PreID, &rec.RemoteFileID, &rec.RemotePickCode, &rec.LastUploadedAt, &deleted); err != nil {
			return nil, err
		}
		rec.Deleted = deleted != 0
		items = append(items, rec)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
