package open115crypt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeriveKey_EmptyPassword(t *testing.T) {
	_, err := DeriveKey("", "salt")
	if err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestDeriveKey_Deterministic(t *testing.T) {
	k1, err := DeriveKey("password123", "salt456")
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}
	k2, err := DeriveKey("password123", "salt456")
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}
	if len(k1) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(k1))
	}
	for i := range k1 {
		if k1[i] != k2[i] {
			t.Fatalf("keys differ at byte %d: %x != %x", i, k1[i], k2[i])
		}
	}
}

func TestDeriveKey_EmptySaltFallback(t *testing.T) {
	// 空 salt 应该回退到 SHA256(password) 前 8 字节
	key, err := DeriveKey("password", "")
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}
}

func TestDeriveKey_DifferentPasswordsDifferentKeys(t *testing.T) {
	k1, _ := DeriveKey("alice", "salt")
	k2, _ := DeriveKey("bob", "salt")
	allSame := true
	for i := range k1 {
		if k1[i] != k2[i] {
			allSame = false
			break
		}
	}
	if allSame {
		t.Fatal("expected different keys for different passwords")
	}
}

func TestEncryptFileToTemp_DisabledConfig(t *testing.T) {
	cfg := Config{Enabled: false, Password: "pw", Salt: "s"}
	_, err := EncryptFileToTemp("anything.txt", cfg)
	if err == nil {
		t.Fatal("expected error when encrypt is disabled")
	}
}

func TestEncryptFileToTemp_SmallFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello open115 encryption test data")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	cfg := Config{
		Enabled:  true,
		Password: "test-password-123",
		Salt:     "test-salt",
		TempDir:  tmpDir,
	}

	result, err := EncryptFileToTemp(srcFile, cfg)
	if err != nil {
		t.Fatalf("EncryptFileToTemp: %v", err)
	}

	// 基本验证
	if result.TempPath == "" {
		t.Fatal("expected non-empty TempPath")
	}
	if result.OriginalSize != int64(len(content)) {
		t.Fatalf("OriginalSize = %d, want %d", result.OriginalSize, len(content))
	}
	if result.EncryptedSize <= result.OriginalSize {
		t.Fatalf("EncryptedSize (%d) should be > OriginalSize (%d) due to header+nonce+tag", result.EncryptedSize, result.OriginalSize)
	}
	if result.Version != "v1" {
		t.Fatalf("Version = %q, want 'v1'", result.Version)
	}

	// 验证加密文件以 magic header 开头
	data, err := os.ReadFile(result.TempPath)
	if err != nil {
		t.Fatalf("read encrypted file: %v", err)
	}
	if len(data) < 8 || string(data[:8]) != "IM115ENC" {
		t.Fatalf("expected magic header 'IM115ENC', got %q", string(data[:8]))
	}

	// 清理
	os.Remove(result.TempPath)
}
