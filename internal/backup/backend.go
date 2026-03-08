package backup

import "context"

// RemoteEntry 是统一的远端条目抽象，供不同备份后端复用。
type RemoteEntry struct {
	ID      string
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	ModTime int64
}

// Backend 定义备份后端的最小能力边界。
//
// 第一阶段只要求：
// - TestConnection
// - UploadFile
// 后续再逐步补：
// - EnsureDir
// - ListRemote
// - DeleteRemote
// - 增量 copy / sync
//
// 这样可以让现有 WebDAV / rclone 与未来的 Open115 共享一套上层流程接口。
type Backend interface {
	TestConnection(ctx context.Context) error
	EnsureDir(ctx context.Context, remotePath string) (string, error)
	UploadFile(ctx context.Context, localPath string, remotePath string) error
	ListRemote(ctx context.Context, remotePath string) ([]RemoteEntry, error)
	DeleteRemote(ctx context.Context, remotePath string) error
}
