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
	// CountActive returns the number of non-deleted records.
	CountActive(ctx context.Context) (int, error)

	// MarkPendingDelete 标记文件为待删除，设置 pending_delete_at 时间戳。
	MarkPendingDelete(ctx context.Context, path string, pendingAt int64) error
	// ClearPendingDelete 清除待删除标记（文件重新出现时使用）。
	ClearPendingDelete(ctx context.Context, path string) error
	// ListPendingDeletes 返回 pending_delete_at <= olderThan 的待删除记录。
	ListPendingDeletes(ctx context.Context, olderThan int64) ([]FileRecord, error)

	Close() error
}
