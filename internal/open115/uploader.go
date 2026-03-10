package open115

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	sdk "github.com/xhofe/115-sdk-go"
)

const preHashSize int64 = 128 * 1024
const multipartThreshold int64 = 20 * 1024 * 1024
const multipartChunkSize int64 = 20 * 1024 * 1024

// Uploader 负责承载 Open115 上传相关能力。
type Uploader struct {
	service    *Service
	Pacer      *Pacer
	dirMu      sync.Mutex         // 保护 dirCache 读写
	dirCache   map[string]string  // normalized path → dir ID
	dirFlight  sync.Map           // path → chan struct{} (in-flight 去重)
}

func NewUploader(service *Service) *Uploader {
	return &Uploader{
		service:  service,
		Pacer:    NewPacer(),
		dirCache: make(map[string]string),
	}
}

// ClearDirCache 清空目录 ID 缓存。
// 建议在每次备份任务结束后调用，避免跨任务的陈旧缓存。
func (u *Uploader) ClearDirCache() {
	u.dirMu.Lock()
	u.dirCache = make(map[string]string)
	u.dirMu.Unlock()
	u.dirFlight = sync.Map{}
}

func normalizeUploadPath(remotePath string) string {
	normalized := strings.TrimSpace(strings.ReplaceAll(remotePath, "\\", "/"))
	if normalized == "" || normalized == "/" {
		return "/"
	}
	trailingSlash := strings.HasSuffix(normalized, "/")
	cleaned := path.Clean("/" + normalized)
	if cleaned == "." || cleaned == "" {
		return "/"
	}
	if trailingSlash && cleaned != "/" {
		return cleaned + "/"
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

func IsRateLimitedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// 已知的限速错误关键词
	if strings.Contains(msg, "refresh frequently") || strings.Contains(msg, "40140117") {
		return true
	}
	// 115 API 有时返回 code:0 + 空 message 作为隐式限速响应
	// 注意：SDK 可能将错误包装成 "someprefix: code: 0, message:"，所以不能用 TrimPrefix
	const marker = "code: 0, message:"
	if idx := strings.Index(msg, marker); idx >= 0 {
		after := strings.TrimSpace(msg[idx+len(marker):])
		if after == "" {
			return true
		}
	}
	return false
}

const defaultMaxRetries = 6

func (u *Uploader) listDirItems(ctx context.Context, parentID string) ([]sdk.GetFilesResp_File, error) {
	client, err := u.service.Client()
	if err != nil {
		return nil, err
	}

	const pageSize int64 = 200
	var offset int64 = 0
	items := make([]sdk.GetFilesResp_File, 0, pageSize)

	for {
		resp, err := Call(ctx, u.Pacer, "GetFiles", defaultMaxRetries, func() (*sdk.GetFilesResp, error) {
			return client.GetFiles(ctx, &sdk.GetFilesReq{
				CID:     parentID,
				Limit:   pageSize,
				Offset:  offset,
				ASC:     true,
				O:       "file_name",
				ShowDir: true,
			})
		})
		if err != nil {
			return nil, err
		}
		items = append(items, resp.Data...)
		if len(resp.Data) == 0 || int64(len(items)) >= resp.Count || int64(len(resp.Data)) < pageSize {
			break
		}
		offset += pageSize
	}

	return items, nil
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
// 使用 dirCache 缓存已解析的目录 ID，同路径只查一次 API。
// 并发安全：多个 goroutine 同时请求同一路径时，只有一个执行 API 调用，
// 其余等待结果。不同路径可完全并发。
func (u *Uploader) EnsureDir(ctx context.Context, remotePath string) (string, error) {
	if u == nil || u.service == nil {
		return "", fmt.Errorf("open115 uploader not initialized")
	}
	cleaned := normalizeUploadPath(remotePath)
	if cleaned == "/" {
		return u.rootID(), nil
	}

	// 快速路径：缓存命中
	u.dirMu.Lock()
	if id, ok := u.dirCache[cleaned]; ok {
		u.dirMu.Unlock()
		return id, nil
	}
	u.dirMu.Unlock()

	// 慢路径：per-path 去重，同路径只有一个 goroutine 执行 API 调用
	// 其他请求同路径的 goroutine 等待 channel 关闭后从缓存读取结果
	ch := make(chan struct{})
	if existing, loaded := u.dirFlight.LoadOrStore(cleaned, ch); loaded {
		// 另一个 goroutine 正在解析此路径，等待它完成
		select {
		case <-existing.(chan struct{}):
			// 完成，从缓存获取结果
			u.dirMu.Lock()
			id, ok := u.dirCache[cleaned]
			u.dirMu.Unlock()
			if ok {
				return id, nil
			}
			// 前一个调用失败了，重试
			return u.ensureDirSlow(ctx, cleaned)
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	// 我们是第一个请求此路径的 goroutine
	defer func() {
		close(ch)
		u.dirFlight.Delete(cleaned)
	}()

	return u.ensureDirSlow(ctx, cleaned)
}

// ensureDirSlow 执行实际的目录解析（不持有 dirMu 期间调用 API）。
func (u *Uploader) ensureDirSlow(ctx context.Context, cleaned string) (string, error) {
	client, err := u.service.Client()
	if err != nil {
		return "", err
	}

	currentID := u.rootID()
	builtPath := ""
	for _, seg := range strings.Split(strings.TrimPrefix(cleaned, "/"), "/") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		builtPath += "/" + seg

		// 中间路径查缓存（短暂加锁）
		u.dirMu.Lock()
		if id, ok := u.dirCache[builtPath]; ok {
			u.dirMu.Unlock()
			currentID = id
			continue
		}
		u.dirMu.Unlock()

		// 缓存未命中 → 查询 API（不持锁）
		existingID, err := u.findDirByName(ctx, currentID, seg)
		if err != nil {
			return "", err
		}
		if existingID != "" {
			u.dirMu.Lock()
			u.dirCache[builtPath] = existingID
			u.dirMu.Unlock()
			currentID = existingID
			continue
		}

		// 目录不存在 → 创建（不持锁）
		created, err := Call(ctx, u.Pacer, "Mkdir", defaultMaxRetries, func() (*sdk.MkdirResp, error) {
			return client.Mkdir(ctx, currentID, seg)
		})
		if err != nil {
			return "", err
		}
		u.dirMu.Lock()
		u.dirCache[builtPath] = created.FileID
		u.dirMu.Unlock()
		currentID = created.FileID
	}

	// 缓存完整路径
	u.dirMu.Lock()
	u.dirCache[cleaned] = currentID
	u.dirMu.Unlock()
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

	// 在一次遍历中同时计算完整文件 SHA1 和前 128KB 的 preID
	fullHasher := sha1.New()
	preHasher := sha1.New()
	preLen := preHashSize
	if size < preLen {
		preLen = size
	}
	var preCollected int64

	buf := make([]byte, 32*1024)
	for {
		n, readErr := file.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			fullHasher.Write(chunk)
			if preCollected < preLen {
				remaining := preLen - preCollected
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
			return "", "", 0, readErr
		}
	}

	fileSHA1 = strings.ToUpper(fmt.Sprintf("%x", fullHasher.Sum(nil)))
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

func newOSSBucket(tokenResp *sdk.UploadGetTokenResp, initResp *sdk.UploadInitResp) (*oss.Bucket, error) {
	ossClient, err := oss.New(tokenResp.Endpoint, tokenResp.AccessKeyId, tokenResp.AccessKeySecret, oss.SecurityToken(tokenResp.SecurityToken))
	if err != nil {
		return nil, err
	}
	return ossClient.Bucket(initResp.Bucket)
}

func putObject(ctx context.Context, localPath string, tokenResp *sdk.UploadGetTokenResp, initResp *sdk.UploadInitResp) error {
	bucket, err := newOSSBucket(tokenResp, initResp)
	if err != nil {
		return err
	}
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return bucket.PutObject(initResp.Object, f,
		oss.Callback(base64.StdEncoding.EncodeToString([]byte(initResp.Callback.Value.Callback))),
		oss.CallbackVar(base64.StdEncoding.EncodeToString([]byte(initResp.Callback.Value.CallbackVar))),
	)
}

func putObjectReader(ctx context.Context, reader io.Reader, tokenResp *sdk.UploadGetTokenResp, initResp *sdk.UploadInitResp) error {
	bucket, err := newOSSBucket(tokenResp, initResp)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return bucket.PutObject(initResp.Object, reader,
		oss.Callback(base64.StdEncoding.EncodeToString([]byte(initResp.Callback.Value.Callback))),
		oss.CallbackVar(base64.StdEncoding.EncodeToString([]byte(initResp.Callback.Value.CallbackVar))),
	)
}

func multipartUpload(ctx context.Context, localPath string, size int64, tokenResp *sdk.UploadGetTokenResp, initResp *sdk.UploadInitResp) error {
	bucket, err := newOSSBucket(tokenResp, initResp)
	if err != nil {
		return err
	}
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()
	imur, err := bucket.InitiateMultipartUpload(initResp.Object, oss.Sequential())
	if err != nil {
		return err
	}
	partNum := (size + multipartChunkSize - 1) / multipartChunkSize
	parts := make([]oss.UploadPart, 0, partNum)
	for i := int64(0); i < partNum; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		offset := i * multipartChunkSize
		partSize := multipartChunkSize
		if offset+partSize > size {
			partSize = size - offset
		}
		section := io.NewSectionReader(file, offset, partSize)
		part, err := bucket.UploadPart(imur, section, partSize, int(i+1))
		if err != nil {
			return err
		}
		parts = append(parts, part)
	}
	_, err = bucket.CompleteMultipartUpload(
		imur,
		parts,
		oss.Callback(base64.StdEncoding.EncodeToString([]byte(initResp.Callback.Value.Callback))),
		oss.CallbackVar(base64.StdEncoding.EncodeToString([]byte(initResp.Callback.Value.CallbackVar))),
	)
	return err
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
	trimmed := strings.Trim(strings.TrimPrefix(cleaned, "/"), "/")
	for _, seg := range strings.Split(trimmed, "/") {
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
	cleaned := normalizeUploadPath(remotePath)
	dirID, err := u.ResolveDirID(ctx, cleaned)
	if err != nil {
		return nil, err
	}
	items, err := u.listDirItems(ctx, dirID)
	if err != nil {
		return nil, err
	}
	base := strings.TrimSuffix(cleaned, "/")
	if base == "" {
		base = "/"
	}
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

// UploadReader 直接上传一个 reader 到指定逻辑远端路径（包含文件名）。
// 保留为兼容入口，默认使用占位 init 参数；更推荐调用 UploadReaderWithInit。
func (u *Uploader) UploadReader(ctx context.Context, reader io.Reader, remotePath string) error {
	return u.UploadReaderWithInit(ctx, reader, remotePath, 1, strings.Repeat("0", 40), strings.Repeat("0", 40))
}

func (u *Uploader) UploadReaderWithInit(ctx context.Context, reader io.Reader, remotePath string, size int64, fileID string, preID string) error {
	if u == nil || u.service == nil {
		return fmt.Errorf("open115 uploader not initialized")
	}
	if reader == nil {
		return fmt.Errorf("reader 不能为空")
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
	if size <= 0 {
		size = 1
	}
	if strings.TrimSpace(fileID) == "" {
		fileID = strings.Repeat("0", 40)
	}
	if strings.TrimSpace(preID) == "" {
		preID = strings.Repeat("0", 40)
	}
	log.Printf("[open115-reader] UploadInitReader start: remote=%s size=%d", remotePath, size)
	initResp, err := Call(ctx, u.Pacer, "UploadInitReader", defaultMaxRetries, func() (*sdk.UploadInitResp, error) {
		return client.UploadInit(ctx, &sdk.UploadInitReq{
			FileName: fileName,
			FileSize: size,
			Target:   dirID,
			FileID:   fileID,
			PreID:    preID,
		})
	})
	if err != nil {
		return err
	}
	log.Printf("[open115-reader] UploadInitReader done: remote=%s object=%s status=%d", remotePath, initResp.Object, initResp.Status)
	// 秒传成功，无需实际上传
	if initResp.Status == 2 {
		log.Printf("[open115-reader] fast transfer (秒传) success: remote=%s", remotePath)
		return nil
	}
	tokenResp, err := Call(ctx, u.Pacer, "UploadGetTokenReader", defaultMaxRetries, func() (*sdk.UploadGetTokenResp, error) {
		return client.UploadGetToken(ctx)
	})
	if err != nil {
		return err
	}
	log.Printf("[open115-reader] UploadGetTokenReader done: remote=%s bucket=%s", remotePath, initResp.Bucket)
	err = putObjectReader(ctx, reader, tokenResp, initResp)
	if err != nil {
		return err
	}
	log.Printf("[open115-reader] PutObjectReader done: remote=%s", remotePath)
	return nil
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
	initResp, err := Call(ctx, u.Pacer, "UploadInit", defaultMaxRetries, func() (*sdk.UploadInitResp, error) {
		return client.UploadInit(ctx, &sdk.UploadInitReq{
			FileName: fileName,
			FileSize: size,
			Target:   dirID,
			FileID:   fileSHA1,
			PreID:    preID,
		})
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
		initResp, err = Call(ctx, u.Pacer, "UploadInitSign", defaultMaxRetries, func() (*sdk.UploadInitResp, error) {
			return client.UploadInit(ctx, &sdk.UploadInitReq{
				FileName: fileName,
				FileSize: size,
				Target:   dirID,
				FileID:   fileSHA1,
				PreID:    preID,
				SignKey:  initResp.SignKey,
				SignVal:  signVal,
			})
		})
		if err != nil {
			return err
		}
		if initResp.Status == 2 {
			return nil
		}
	}
	tokenResp, err := Call(ctx, u.Pacer, "UploadGetToken", defaultMaxRetries, func() (*sdk.UploadGetTokenResp, error) {
		return client.UploadGetToken(ctx)
	})
	if err != nil {
		return err
	}
	if size >= multipartThreshold {
		return multipartUpload(ctx, localPath, size, tokenResp, initResp)
	}
	return putObject(ctx, localPath, tokenResp, initResp)
}
