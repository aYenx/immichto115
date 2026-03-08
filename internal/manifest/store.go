package manifest

import "context"

// Store 定义增量索引的最小操作集。
type Store interface {
	Init(ctx context.Context) error
	Get(ctx context.Context, path string) (*FileRecord, error)
	Put(ctx context.Context, record *FileRecord) error
	MarkDeleted(ctx context.Context, path string, deleted bool) error
	Delete(ctx context.Context, path string) error
	List(ctx context.Context, limit int, offset int) ([]FileRecord, error)
	Close() error
}
