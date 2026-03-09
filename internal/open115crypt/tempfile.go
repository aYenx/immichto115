package open115crypt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveTempDir(tempDir string) (string, error) {
	tempDir = strings.TrimSpace(tempDir)
	if tempDir == "" {
		tempDir = filepath.Join(os.TempDir(), "immichto115-open115-encrypt")
	}
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		return "", err
	}
	return tempDir, nil
}

func CreateTempEncryptedPath(tempDir string) (string, error) {
	resolved, err := ResolveTempDir(tempDir)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp(resolved, "enc-*.bin")
	if err != nil {
		return "", err
	}
	path := f.Name()
	_ = f.Close()
	return path, nil
}

func CleanupTempFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cleanup temp encrypted file failed: %w", err)
	}
	return nil
}
