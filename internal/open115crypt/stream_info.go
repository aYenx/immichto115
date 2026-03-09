package open115crypt

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func MeasureEncryptedStream(localPath string, cfg Config) (*StreamInfo, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	reader, err := NewEncryptedReader(f, cfg, StreamMeta{
		OriginalName:  filepath.Base(localPath),
		OriginalSize:  stat.Size(),
		OriginalMTime: stat.ModTime().Unix(),
	})
	if err != nil {
		return nil, err
	}
	hasher := sha1.New()
	var size int64
	preBuf := make([]byte, 0, 128*1024)
	buf := make([]byte, 32*1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			size += int64(n)
			_, _ = hasher.Write(chunk)
			if len(preBuf) < 128*1024 {
				remaining := 128*1024 - len(preBuf)
				if remaining > n {
					remaining = n
				}
				preBuf = append(preBuf, chunk[:remaining]...)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	preHasher := sha1.New()
	_, _ = preHasher.Write(preBuf)
	return &StreamInfo{
		Size:    size,
		SHA1:    strings.ToUpper(fmt.Sprintf("%x", hasher.Sum(nil))),
		PreID:   strings.ToUpper(fmt.Sprintf("%x", preHasher.Sum(nil))),
		Version: "v2-stream",
	}, nil
}
