package backup

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aYenx/immichto115/internal/open115crypt"
)

func (b *Open115Backend) UploadEncryptedStream(ctx context.Context, localPath string, remotePath string, cfg open115crypt.Config) error {
	log.Printf("[open115-stream] encrypt+measure start: local=%s remote=%s", localPath, remotePath)

	// 1. 打开源文件
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	// 2. 创建加密 reader
	reader, err := open115crypt.NewEncryptedReader(file, cfg, open115crypt.StreamMeta{
		OriginalName:  filepath.Base(localPath),
		OriginalSize:  stat.Size(),
		OriginalMTime: stat.ModTime().Unix(),
	})
	if err != nil {
		return fmt.Errorf("create encrypted reader failed: %w", err)
	}

	// 3. 一次加密，同时写入临时文件并计算 SHA1/preID/size
	tmpDir, err := open115crypt.ResolveTempDir(cfg.TempDir)
	if err != nil {
		return fmt.Errorf("resolve temp dir failed: %w", err)
	}
	tmpFile, err := os.CreateTemp(tmpDir, "stream-cache-*.bin")
	if err != nil {
		return fmt.Errorf("create temp cache file failed: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		_ = open115crypt.CleanupTempFile(tmpPath)
	}()

	fullHasher := sha1.New()
	preHasher := sha1.New()
	var totalSize int64
	var preCollected int64
	const preHashLimit int64 = 128 * 1024

	buf := make([]byte, 32*1024)
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if _, we := tmpFile.Write(chunk); we != nil {
				return fmt.Errorf("write to temp cache failed: %w", we)
			}
			fullHasher.Write(chunk)
			totalSize += int64(n)
			if preCollected < preHashLimit {
				remaining := preHashLimit - preCollected
				if int64(n) <= remaining {
					preHasher.Write(chunk)
				} else {
					preHasher.Write(chunk[:remaining])
				}
				preCollected += int64(n)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read encrypted stream failed: %w", readErr)
		}
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync temp cache failed: %w", err)
	}

	fileSHA1 := strings.ToUpper(fmt.Sprintf("%x", fullHasher.Sum(nil)))
	preID := strings.ToUpper(fmt.Sprintf("%x", preHasher.Sum(nil)))
	log.Printf("[open115-stream] encrypt+measure done: size=%d sha1=%s preid=%s", totalSize, fileSHA1, preID)

	// 4. 从临时文件上传（避免第二次加密）
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek temp cache failed: %w", err)
	}

	log.Printf("[open115-stream] upload start: remote=%s", remotePath)
	err = b.uploader.UploadReaderWithInit(ctx, tmpFile, remotePath, totalSize, fileSHA1, preID)
	if err != nil {
		return fmt.Errorf("upload encrypted stream failed: %w", err)
	}
	log.Printf("[open115-stream] upload done: remote=%s", remotePath)
	return nil
}
