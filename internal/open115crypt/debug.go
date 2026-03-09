package open115crypt

import (
	"path/filepath"
	"os"
)

type StreamDebugInfo struct {
	OriginalName  string `json:"original_name"`
	OriginalSize  int64  `json:"original_size"`
	MeasuredSize  int64  `json:"measured_size"`
	SHA1          string `json:"sha1"`
	PreID         string `json:"preid"`
	Version       string `json:"version"`
}

func DebugMeasure(localPath string, cfg Config) (*StreamDebugInfo, error) {
	info, err := os.Stat(localPath)
	if err != nil {
		return nil, err
	}
	measured, err := MeasureEncryptedStream(localPath, cfg)
	if err != nil {
		return nil, err
	}
	return &StreamDebugInfo{
		OriginalName: filepath.Base(localPath),
		OriginalSize: info.Size(),
		MeasuredSize: measured.Size,
		SHA1:         measured.SHA1,
		PreID:        measured.PreID,
		Version:      measured.Version,
	}, nil
}
