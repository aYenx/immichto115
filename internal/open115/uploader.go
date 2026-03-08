package open115

import (
	"context"
	"fmt"
)

// Uploader 负责承载 Open115 上传相关能力。
//
// 当前先提供最小骨架，后续会逐步补齐：
// - EnsureDir
// - 单文件上传
// - 分片上传
// - 断点续传
// - 远端浏览 / 删除
// - 增量 copy / sync
//
// 上传实现将优先参考：
// - OpenList/drivers/115_open/upload.go
// - OpenListTeam/115-sdk-go/upload.go
//
// 这样做是为了把“授权逻辑”和“上传逻辑”分开，后续更容易测试和维护。
type Uploader struct {
	service *Service
}

func NewUploader(service *Service) *Uploader {
	return &Uploader{service: service}
}

func (u *Uploader) EnsureDir(ctx context.Context, remotePath string) (string, error) {
	_ = ctx
	if u == nil || u.service == nil {
		return "", fmt.Errorf("open115 uploader not initialized")
	}
	if remotePath == "" {
		return "", fmt.Errorf("remotePath 不能为空")
	}
	return remotePath, nil
}

func (u *Uploader) UploadFile(ctx context.Context, localPath string, remotePath string) error {
	_ = ctx
	if u == nil || u.service == nil {
		return fmt.Errorf("open115 uploader not initialized")
	}
	if localPath == "" {
		return fmt.Errorf("localPath 不能为空")
	}
	if remotePath == "" {
		return fmt.Errorf("remotePath 不能为空")
	}
	return fmt.Errorf("open115 UploadFile 尚未实现")
}
