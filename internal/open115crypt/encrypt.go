package open115crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type EncryptedTempFile struct {
	TempPath      string
	EncryptedSize int64
	OriginalSize  int64
	Version       string
}

func EncryptFileToTemp(srcPath string, cfg Config) (*EncryptedTempFile, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("open115 encrypt 未启用")
	}
	in, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	stat, err := in.Stat()
	if err != nil {
		return nil, err
	}
	key, err := DeriveKey(cfg.Password, cfg.Salt)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	header := FileHeader{
		Magic:         "IM115ENC",
		Version:       "v1",
		Algorithm:     "aes-256-gcm",
		OriginalName:  filepath.Base(srcPath),
		OriginalSize:  stat.Size(),
		OriginalMTime: stat.ModTime().Unix(),
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}
	plaintext, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, headerBytes)
	tempPath, err := CreateTempEncryptedPath(cfg.TempDir)
	if err != nil {
		return nil, err
	}
	out, err := os.Create(tempPath)
	if err != nil {
		return nil, err
	}
	defer out.Close()
	if _, err := out.Write([]byte("IM115ENC\n")); err != nil {
		return nil, err
	}
	if _, err := out.Write(headerBytes); err != nil {
		return nil, err
	}
	if _, err := out.Write([]byte("\n")); err != nil {
		return nil, err
	}
	if _, err := out.Write(nonce); err != nil {
		return nil, err
	}
	if _, err := out.Write(ciphertext); err != nil {
		return nil, err
	}
	if err := out.Sync(); err != nil {
		return nil, err
	}
	info, err := out.Stat()
	if err != nil {
		return nil, err
	}
	return &EncryptedTempFile{TempPath: tempPath, EncryptedSize: info.Size(), OriginalSize: stat.Size(), Version: "v1"}, nil
}
