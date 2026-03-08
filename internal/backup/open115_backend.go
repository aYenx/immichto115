package backup

import (
	"context"

	"github.com/aYenx/immichto115/internal/open115"
)

type Open115Backend struct {
	uploader *open115.Uploader
	service  *open115.Service
}

func NewOpen115Backend(service *open115.Service) *Open115Backend {
	return &Open115Backend{
		service:  service,
		uploader: open115.NewUploader(service),
	}
}

func (b *Open115Backend) TestConnection(ctx context.Context) error {
	return b.service.TestConnection(ctx)
}

func (b *Open115Backend) EnsureDir(ctx context.Context, remotePath string) (string, error) {
	return b.uploader.EnsureDir(ctx, remotePath)
}

func (b *Open115Backend) UploadFile(ctx context.Context, localPath string, remotePath string) error {
	return b.uploader.UploadFile(ctx, localPath, remotePath)
}

func (b *Open115Backend) ListRemote(ctx context.Context, remotePath string) ([]RemoteEntry, error) {
	items, err := b.uploader.ListRemote(ctx, remotePath)
	if err != nil {
		return nil, err
	}
	result := make([]RemoteEntry, 0, len(items))
	for _, item := range items {
		result = append(result, RemoteEntry{
			ID:      item.ID,
			Name:    item.Name,
			Path:    item.Path,
			IsDir:   item.IsDir,
			Size:    item.Size,
			ModTime: item.ModTime.Unix(),
		})
	}
	return result, nil
}

func (b *Open115Backend) DeleteRemote(ctx context.Context, remotePath string) error {
	_ = ctx
	_ = remotePath
	return nil
}
