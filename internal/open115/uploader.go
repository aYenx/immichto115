package open115

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	sdk "github.com/xhofe/115-sdk-go"
)

const preHashSize int64 = 128 * 1024

// Uploader 负责承载 Open115 上传相关能力。
type Uploader struct {
	service *Service
}

func NewUploader(service *Service) *Uploader {
	return &Uploader{service: service}
}

func normalizeUploadPath(remotePath string) string {
	cleaned := path.Clean("/" + strings.TrimSpace(strings.ReplaceAll(remotePath, "\\", "/")))
	if cleaned == "." || cleaned == "" {
		return "/"
	}
	return cleaned
}

func (u *Uploader) rootID() string {
	if u == nil || u.service == nil {
		return "0"
	}
	cfg := u.service.Config()
	if strings.TrimSpace(cfg.RootID) == "" {
		return "0"
	}
	return strings.TrimSpace(cfg.RootID)
}

func (u *Uploader) listDirItems(ctx context.Context, parentID string) ([]sdk.GetFilesResp_File, error) {
	client, err := u.service.Client()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetFiles(ctx, &sdk.GetFilesReq{
		CID:     parentID,
		Limit:   200,
		Offset:  0,
		ASC:     true,
		O:       "file_name",
		ShowDir: true,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (u *Uploader) findDirByName(ctx context.Context, parentID, name string) (string, error) {
	items, err := u.listDirItems(ctx, parentID)
	if err != nil {
		return "", err
	}
	for _, item := range items {
		if item.Fc == "0" && item.Fn == name {
			return item.Fid, nil
		}
	}
	return "", nil
}

// EnsureDir 确保逻辑路径存在，并返回最终目录 ID。
func (u *Uploader) EnsureDir(ctx context.Context, remotePath string) (string, error) {
	if u == nil || u.service == nil {
		return "", fmt.Errorf("open115 uploader not initialized")
	}
	cleaned := normalizeUploadPath(remotePath)
	if cleaned == "/" {
		return u.rootID(), nil
	}
	client, err := u.service.Client()
	if err != nil {
		return "", err
	}
	currentID := u.rootID()
	for _, seg := range strings.Split(strings.TrimPrefix(cleaned, "/"), "/") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		existingID, err := u.findDirByName(ctx, currentID, seg)
		if err != nil {
			return "", err
		}
		if existingID != "" {
			currentID = existingID
			continue
		}
		created, err := client.Mkdir(ctx, currentID, seg)
		if err != nil {
			return "", err
		}
		currentID = created.FileID
	}
	return currentID, nil
}

func fileSHA1AndPreID(localPath string) (fileSHA1 string, preID string, size int64, err error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", "", 0, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", "", 0, err
	}
	size = info.Size()

	h := sha1.New()
	if _, err = io.Copy(h, file); err != nil {
		return "", "", 0, err
	}
	fileSHA1 = strings.ToUpper(fmt.Sprintf("%x", h.Sum(nil)))

	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return "", "", 0, err
	}
	preLen := preHashSize
	if size < preLen {
		preLen = size
	}
	preHasher := sha1.New()
	if _, err = io.CopyN(preHasher, file, preLen); err != nil && err != io.EOF {
		return "", "", 0, err
	}
	preID = strings.ToUpper(fmt.Sprintf("%x", preHasher.Sum(nil)))
	return fileSHA1, preID, size, nil
}

func signValForRange(localPath string, signCheck string) (string, error) {
	parts := strings.Split(signCheck, "-")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid sign_check: %s", signCheck)
	}
	var start, end int64
	_, err := fmt.Sscanf(signCheck, "%d-%d", &start, &end)
	if err != nil {
		return "", err
	}
	f, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return "", err
	}
	h := sha1.New()
	if _, err := io.CopyN(h, f, end-start+1); err != nil && err != io.EOF {
		return "", err
	}
	return strings.ToUpper(fmt.Sprintf("%x", h.Sum(nil))), nil
}

func putObject(localPath string, tokenResp *sdk.UploadGetTokenResp, initResp *sdk.UploadInitResp) error {
	ossClient, err := oss.New(tokenResp.Endpoint, tokenResp.AccessKeyId, tokenResp.AccessKeySecret, oss.SecurityToken(tokenResp.SecurityToken))
	if err != nil {
		return err
	}
	bucket, err := ossClient.Bucket(initResp.Bucket)
	if err != nil {
		return err
	}
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return bucket.PutObject(initResp.Object, f,
		oss.Callback(base64.StdEncoding.EncodeToString([]byte(initResp.Callback.Value.Callback))),
		oss.CallbackVar(base64.StdEncoding.EncodeToString([]byte(initResp.Callback.Value.CallbackVar))),
	)
}

func (u *Uploader) ResolveDirID(ctx context.Context, remotePath string) (string, error) {
	if u == nil || u.service == nil {
		return "", fmt.Errorf("open115 uploader not initialized")
	}
	cleaned := normalizeUploadPath(remotePath)
	if cleaned == "/" {
		return u.rootID(), nil
	}
	currentID := u.rootID()
	for _, seg := range strings.Split(strings.TrimPrefix(cleaned, "/"), "/") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		existingID, err := u.findDirByName(ctx, currentID, seg)
		if err != nil {
			return "", err
		}
		if existingID == "" {
			return "", fmt.Errorf("目录不存在: %s", cleaned)
		}
		currentID = existingID
	}
	return currentID, nil
}

func (u *Uploader) ListRemote(ctx context.Context, remotePath string) ([]RemoteEntry, error) {
	if u == nil || u.service == nil {
		return nil, fmt.Errorf("open115 uploader not initialized")
	}
	dirID, err := u.ResolveDirID(ctx, remotePath)
	if err != nil {
		return nil, err
	}
	items, err := u.listDirItems(ctx, dirID)
	if err != nil {
		return nil, err
	}
	base := normalizeUploadPath(remotePath)
	result := make([]RemoteEntry, 0, len(items))
	for _, item := range items {
		p := path.Join(base, item.Fn)
		result = append(result, RemoteEntry{
			ID:       item.Fid,
			Name:     item.Fn,
			Path:     p,
			IsDir:    item.Fc == "0",
			Size:     item.FS,
			ModTime:  time.Unix(item.Upt, 0),
			PickCode: item.Pc,
		})
	}
	return result, nil
}

// UploadFile 上传一个本地文件到指定逻辑远端路径（包含文件名）。
func (u *Uploader) UploadFile(ctx context.Context, localPath string, remotePath string) error {
	if u == nil || u.service == nil {
		return fmt.Errorf("open115 uploader not initialized")
	}
	if strings.TrimSpace(localPath) == "" {
		return fmt.Errorf("localPath 不能为空")
	}
	if strings.TrimSpace(remotePath) == "" {
		return fmt.Errorf("remotePath 不能为空")
	}
	client, err := u.service.Client()
	if err != nil {
		return err
	}
	cleaned := normalizeUploadPath(remotePath)
	dirPath, fileName := path.Split(cleaned)
	if strings.TrimSpace(fileName) == "" {
		return fmt.Errorf("remotePath 必须包含文件名")
	}
	dirID, err := u.EnsureDir(ctx, dirPath)
	if err != nil {
		return err
	}
	fileSHA1, preID, size, err := fileSHA1AndPreID(localPath)
	if err != nil {
		return err
	}
	initResp, err := client.UploadInit(ctx, &sdk.UploadInitReq{
		FileName: fileName,
		FileSize: size,
		Target:   dirID,
		FileID:   fileSHA1,
		PreID:    preID,
	})
	if err != nil {
		return err
	}
	if initResp.Status == 2 {
		return nil
	}
	if initResp.Status == 6 || initResp.Status == 7 || initResp.Status == 8 {
		signVal, err := signValForRange(localPath, initResp.SignCheck)
		if err != nil {
			return err
		}
		initResp, err = client.UploadInit(ctx, &sdk.UploadInitReq{
			FileName: fileName,
			FileSize: size,
			Target:   dirID,
			FileID:   fileSHA1,
			PreID:    preID,
			SignKey:  initResp.SignKey,
			SignVal:  signVal,
		})
		if err != nil {
			return err
		}
		if initResp.Status == 2 {
			return nil
		}
	}
	tokenResp, err := client.UploadGetToken(ctx)
	if err != nil {
		return err
	}
	return putObject(localPath, tokenResp, initResp)
}
