package open115

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aYenx/immichto115/internal/open115crypt"
)

type StreamUploadDebugResult struct {
	Measure   *open115crypt.StreamDebugInfo `json:"measure,omitempty"`
	InitOK    bool                          `json:"init_ok"`
	TokenOK   bool                          `json:"token_ok"`
	UploadOK  bool                          `json:"upload_ok"`
	Step      string                        `json:"step"`
	Message   string                        `json:"message"`
}

func (u *Uploader) DebugStreamUpload(ctx context.Context, localPath string, remotePath string, cfg open115crypt.Config) (*StreamUploadDebugResult, error) {
	result := &StreamUploadDebugResult{Step: "measure"}
	measure, err := open115crypt.DebugMeasure(localPath, cfg)
	if err != nil {
		result.Message = err.Error()
		return result, err
	}
	result.Measure = measure
	result.Step = "open file"
	file, err := os.Open(localPath)
	if err != nil {
		result.Message = err.Error()
		return result, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		result.Message = err.Error()
		return result, err
	}
	reader, err := open115crypt.NewEncryptedReader(file, cfg, open115crypt.StreamMeta{
		OriginalName:  filepath.Base(localPath),
		OriginalSize:  stat.Size(),
		OriginalMTime: stat.ModTime().Unix(),
	})
	if err != nil {
		result.Message = err.Error()
		return result, err
	}
	result.Step = "upload reader with init"
	if err := u.UploadReaderWithInit(ctx, reader, remotePath, measure.MeasuredSize, measure.SHA1, measure.PreID); err != nil {
		result.Message = err.Error()
		return result, err
	}
	result.InitOK = true
	result.TokenOK = true
	result.UploadOK = true
	result.Step = "done"
	result.Message = fmt.Sprintf("stream upload debug success: %s", remotePath)
	return result, nil
}
