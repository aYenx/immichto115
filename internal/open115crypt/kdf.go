package open115crypt

import (
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/scrypt"
)

func DeriveKey(password string, salt string) ([]byte, error) {
	if password == "" {
		return nil, fmt.Errorf("open115 encrypt password 不能为空")
	}
	if salt == "" {
		h := sha256.Sum256([]byte(password))
		salt = fmt.Sprintf("%x", h[:8])
	}
	return scrypt.Key([]byte(password), []byte(salt), 1<<15, 8, 1, 32)
}
