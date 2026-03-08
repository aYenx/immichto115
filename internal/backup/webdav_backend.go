package backup

import (
	"context"
	"fmt"
)

// WebDAVRcloneBackend 预留给现有 WebDAV + rclone 模式的统一包装层。
// 当前阶段先提供骨架，后续再把现有 router/rclone 调用收口到这里。
type WebDAVRcloneBackend struct{}

func NewWebDAVRcloneBackend() *WebDAVRcloneBackend {
	return &WebDAVRcloneBackend{}
}

func (b *WebDAVRcloneBackend) TestConnection(ctx context.Context) error {
	_ = ctx
	return fmt.Errorf("webdav backend test connection 尚未收口到统一接口")
}

func (b *WebDAVRcloneBackend) EnsureDir(ctx context.Context, remotePath string) (string, error) {
	_ = ctx
	return remotePath, nil
}

func (b *WebDAVRcloneBackend) UploadFile(ctx context.Context, localPath string, remotePath string) error {
	_ = ctx
	_ = localPath
	_ = remotePath
	return fmt.Errorf("webdav backend upload 尚未收口到统一接口")
}

func (b *WebDAVRcloneBackend) ListRemote(ctx context.Context, remotePath string) ([]RemoteEntry, error) {
	_ = ctx
	_ = remotePath
	return nil, fmt.Errorf("webdav backend list remote 尚未收口到统一接口")
}

func (b *WebDAVRcloneBackend) DeleteRemote(ctx context.Context, remotePath string) error {
	_ = ctx
	_ = remotePath
	return fmt.Errorf("webdav backend delete remote 尚未收口到统一接口")
}
