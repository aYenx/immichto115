package open115crypt

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const StreamChunkSize = 64 * 1024

type StreamMeta struct {
	OriginalName  string
	OriginalSize  int64
	OriginalMTime int64
}

type StreamInfo struct {
	Size    int64
	SHA1    string
	PreID   string
	Version string
}

// NewEncryptedReader 创建一个分块流式加密 reader。
// 第一阶段约定：
// - header: magic + json + '\n'
// - body: [4-byte chunk len][nonce][ciphertext]
// - 每个 chunk 独立 AES-GCM 加密，便于后续流式解密和 multipart 扩展
func NewEncryptedReader(src io.Reader, cfg Config, meta StreamMeta) (io.Reader, error) {
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
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		header := FileHeader{
			Magic:         "IM115ENC",
			Version:       "v2-stream",
			Algorithm:     "aes-256-gcm-chunked",
			OriginalName:  meta.OriginalName,
			OriginalSize:  meta.OriginalSize,
			OriginalMTime: meta.OriginalMTime,
		}
		headerBytes, err := json.Marshal(header)
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		if _, err := pw.Write([]byte("IM115ENC2\n")); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		if _, err := pw.Write(headerBytes); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		if _, err := pw.Write([]byte("\n")); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		reader := bufio.NewReader(src)
		buf := make([]byte, StreamChunkSize)
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				nonce := make([]byte, gcm.NonceSize())
				if _, e := rand.Read(nonce); e != nil {
					_ = pw.CloseWithError(e)
					return
				}
				sealed := gcm.Seal(nil, nonce, buf[:n], headerBytes)
				frameLen := uint32(len(nonce) + len(sealed))
				var lenBuf [4]byte
				binary.BigEndian.PutUint32(lenBuf[:], frameLen)
				if _, e := pw.Write(lenBuf[:]); e != nil {
					_ = pw.CloseWithError(e)
					return
				}
				if _, e := pw.Write(nonce); e != nil {
					_ = pw.CloseWithError(e)
					return
				}
				if _, e := pw.Write(sealed); e != nil {
					_ = pw.CloseWithError(e)
					return
				}
			}
			if err == io.EOF {
				return
			}
			if err != nil {
				_ = pw.CloseWithError(fmt.Errorf("encrypt stream read failed: %w", err))
				return
			}
		}
	}()
	return pr, nil
}
