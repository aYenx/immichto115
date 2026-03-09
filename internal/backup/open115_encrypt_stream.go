package backup

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/aYenx/immichto115/internal/open115crypt"
)

func (b *Open115Backend) UploadEncryptedStream(ctx context.Context, localPath string, remotePath string, cfg open115crypt.Config) error {
	log.Printf("[open115-stream] measure start: local=%s remote=%s", localPath, remotePath)
	info, err := open115crypt.MeasureEncryptedStream(localPath, cfg)
	if err != nil {
		return fmt.Errorf("measure encrypted stream failed: %w", err)
	}
	log.Printf("[open115-stream] measure done: size=%d sha1=%s preid=%s version=%s", info.Size, info.SHA1, info.PreID, info.Version)

	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	reader, err := open115crypt.NewEncryptedReader(file, cfg, open115crypt.StreamMeta{
		OriginalName:  filepath.Base(localPath),
		OriginalSize:  stat.Size(),
		OriginalMTime: stat.ModTime().Unix(),
	})
	if err != nil {
		return fmt.Errorf("create encrypted reader failed: %w", err)
	}
	log.Printf("[open115-stream] upload start: remote=%s", remotePath)
	err = b.uploader.UploadReaderWithInit(ctx, reader, remotePath, info.Size, info.SHA1, info.PreID)
	if err != nil {
		return fmt.Errorf("upload encrypted stream failed: %w", err)
	}
	log.Printf("[open115-stream] upload done: remote=%s", remotePath)
	return nil
}
