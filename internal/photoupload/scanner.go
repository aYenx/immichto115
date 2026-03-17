package photoupload

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aYenx/immichto115/internal/config"
	"github.com/aYenx/immichto115/internal/open115"
)

// FileEntry 表示一个待上传的摄影文件。
type FileEntry struct {
	LocalPath string
	FileName  string
	Size      int64
	Date      time.Time // 拍摄/修改日期
}

// Summary 汇总上传结果。
type Summary struct {
	Scanned  int `json:"scanned"`
	Uploaded int `json:"uploaded"`
	Skipped  int `json:"skipped"`
	Failed   int `json:"failed"`
	Deleted  int `json:"deleted"`
}

// LogFunc 日志回调。
type LogFunc func(stream string, text string)

// Runner 摄影文件上传执行器。
type Runner struct {
	uploader *open115.Uploader
	cfg      config.PhotoUploadConfig
	log      LogFunc
}

// NewRunner 创建上传执行器。
func NewRunner(uploader *open115.Uploader, cfg config.PhotoUploadConfig, logFn LogFunc) *Runner {
	if logFn == nil {
		logFn = func(stream, text string) {
			log.Printf("[photo-upload][%s] %s", stream, text)
		}
	}
	return &Runner{
		uploader: uploader,
		cfg:      cfg,
		log:      logFn,
	}
}

// parseExtensions 将逗号分隔的扩展名字符串解析为小写 set。
func parseExtensions(ext string) map[string]bool {
	exts := make(map[string]bool)
	for _, e := range strings.Split(ext, ",") {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		exts[e] = true
	}
	return exts
}

// matchExtension 检查文件名是否匹配指定的扩展名集合。
func matchExtension(fileName string, exts map[string]bool) bool {
	ext := strings.ToLower(filepath.Ext(fileName))
	return exts[ext]
}

// Scan 扫描目录下的所有匹配文件。
func Scan(watchDir string, extensions string) ([]FileEntry, error) {
	exts := parseExtensions(extensions)
	if len(exts) == 0 {
		return nil, fmt.Errorf("未配置任何文件扩展名")
	}

	var entries []FileEntry
	err := filepath.WalkDir(watchDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !matchExtension(d.Name(), exts) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil // 跳过无法获取信息的文件
		}
		date := extractDate(path, info)
		entries = append(entries, FileEntry{
			LocalPath: path,
			FileName:  d.Name(),
			Size:      info.Size(),
			Date:      date,
		})
		return nil
	})
	return entries, err
}

// extractDate 尝试从文件 EXIF 提取拍摄日期，失败则使用文件修改时间。
func extractDate(filePath string, info fs.FileInfo) time.Time {
	ext := strings.ToLower(filepath.Ext(filePath))
	// 优先对 JPG 文件尝试 EXIF 解析
	if ext == ".jpg" || ext == ".jpeg" {
		if t, err := readExifDate(filePath); err == nil && !t.IsZero() {
			return t
		}
	}
	return info.ModTime()
}

// Run 执行完整的扫描→上传→删除流程。
func (r *Runner) Run(ctx context.Context) (*Summary, error) {
	summary := &Summary{}

	watchDir := strings.TrimSpace(r.cfg.WatchDir)
	if watchDir == "" {
		return nil, fmt.Errorf("未配置本地监控目录")
	}
	remoteDir := strings.TrimSpace(r.cfg.RemoteDir)
	if remoteDir == "" {
		remoteDir = "/摄影"
	}
	dateFormat := strings.TrimSpace(r.cfg.DateFormat)
	if dateFormat == "" {
		dateFormat = "2006/01/02"
	}

	r.log("stdout", fmt.Sprintf("[photo-upload] 开始扫描目录: %s", watchDir))
	entries, err := Scan(watchDir, r.cfg.Extensions)
	if err != nil {
		return nil, fmt.Errorf("扫描目录失败: %w", err)
	}
	summary.Scanned = len(entries)
	r.log("stdout", fmt.Sprintf("[photo-upload] 扫描完成，找到 %d 个文件", len(entries)))

	if len(entries) == 0 {
		r.log("stdout", "[photo-upload] 没有找到需要上传的文件")
		return summary, nil
	}

	for i, entry := range entries {
		if ctx.Err() != nil {
			r.log("stderr", "[photo-upload] 任务已被取消")
			return summary, ctx.Err()
		}

		dateDir := entry.Date.Format(dateFormat)
		remotePath := strings.TrimSuffix(remoteDir, "/") + "/" + dateDir + "/" + entry.FileName

		r.log("stdout", fmt.Sprintf("[photo-upload] [%d/%d] 上传 %s → %s", i+1, len(entries), entry.FileName, remotePath))

		err := r.uploader.UploadFile(ctx, entry.LocalPath, remotePath)
		if err != nil {
			summary.Failed++
			r.log("stderr", fmt.Sprintf("[photo-upload] 上传失败 %s: %v", entry.FileName, err))
			continue
		}
		summary.Uploaded++
		r.log("stdout", fmt.Sprintf("[photo-upload] 上传成功: %s", entry.FileName))

		if r.cfg.DeleteAfterUpload {
			if err := os.Remove(entry.LocalPath); err != nil {
				r.log("stderr", fmt.Sprintf("[photo-upload] 删除本地文件失败 %s: %v", entry.FileName, err))
			} else {
				summary.Deleted++
				r.log("stdout", fmt.Sprintf("[photo-upload] 已删除本地文件: %s", entry.FileName))
			}
		}
	}

	r.log("stdout", fmt.Sprintf("[photo-upload] 任务完成！扫描 %d，上传 %d，跳过 %d，失败 %d，删除 %d",
		summary.Scanned, summary.Uploaded, summary.Skipped, summary.Failed, summary.Deleted))
	return summary, nil
}

// ====== 轻量 EXIF 日期读取（纯 Go，无第三方依赖） ======

// readExifDate 尝试从 JPEG 文件读取 EXIF DateTimeOriginal。
func readExifDate(filePath string) (time.Time, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return time.Time{}, err
	}
	defer f.Close()

	return parseExifDateFromReader(f)
}

// parseExifDateFromReader 从 JPEG reader 中提取 EXIF 日期。
func parseExifDateFromReader(r io.ReadSeeker) (time.Time, error) {
	// 读取最多 64KB 来查找 EXIF 数据
	buf := make([]byte, 64*1024)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return time.Time{}, err
	}
	buf = buf[:n]

	// 检查 JPEG 标识
	if n < 2 || buf[0] != 0xFF || buf[1] != 0xD8 {
		return time.Time{}, fmt.Errorf("not a JPEG file")
	}

	// 查找 APP1 marker (0xFFE1) - EXIF 数据
	return findExifDate(buf)
}

func findExifDate(data []byte) (time.Time, error) {
	pos := 2
	for pos < len(data)-4 {
		if data[pos] != 0xFF {
			return time.Time{}, fmt.Errorf("invalid JPEG marker")
		}
		marker := data[pos+1]
		if marker == 0xDA { // Start of Scan - 停止搜索
			break
		}
		segLen := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if marker == 0xE1 { // APP1 - EXIF
			segData := data[pos+4 : min(pos+2+segLen, len(data))]
			if t, err := parseExifSegment(segData); err == nil {
				return t, nil
			}
		}
		pos += 2 + segLen
	}
	return time.Time{}, fmt.Errorf("no EXIF date found")
}

func parseExifSegment(data []byte) (time.Time, error) {
	// 检查 "Exif\x00\x00" 头
	if len(data) < 6 || string(data[:4]) != "Exif" {
		return time.Time{}, fmt.Errorf("not EXIF segment")
	}
	tiffData := data[6:]
	if len(tiffData) < 8 {
		return time.Time{}, fmt.Errorf("TIFF data too short")
	}

	// 判断字节序
	var bo binary.ByteOrder
	switch string(tiffData[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return time.Time{}, fmt.Errorf("unknown byte order")
	}

	// 读取 IFD0 偏移
	ifdOffset := int(bo.Uint32(tiffData[4:8]))
	if ifdOffset >= len(tiffData) {
		return time.Time{}, fmt.Errorf("IFD offset out of range")
	}

	// 搜索 IFD0 中的 ExifIFD 指针 (tag 0x8769)
	exifIFDOffset, err := findTagValue(tiffData, bo, ifdOffset, 0x8769)
	if err == nil && exifIFDOffset > 0 && int(exifIFDOffset) < len(tiffData) {
		// 在 ExifIFD 中搜索 DateTimeOriginal (tag 0x9003)
		if t, err := findDateTag(tiffData, bo, int(exifIFDOffset), 0x9003); err == nil {
			return t, nil
		}
		// 回退：DateTimeDigitized (tag 0x9004)
		if t, err := findDateTag(tiffData, bo, int(exifIFDOffset), 0x9004); err == nil {
			return t, nil
		}
	}

	// 回退：IFD0 中的 DateTime (tag 0x0132)
	if t, err := findDateTag(tiffData, bo, ifdOffset, 0x0132); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("no date tag found")
}

func findTagValue(tiffData []byte, bo binary.ByteOrder, ifdOffset int, targetTag uint16) (uint32, error) {
	if ifdOffset+2 > len(tiffData) {
		return 0, fmt.Errorf("IFD offset out of range")
	}
	numEntries := int(bo.Uint16(tiffData[ifdOffset : ifdOffset+2]))
	for i := 0; i < numEntries; i++ {
		entryOffset := ifdOffset + 2 + i*12
		if entryOffset+12 > len(tiffData) {
			break
		}
		tag := bo.Uint16(tiffData[entryOffset : entryOffset+2])
		if tag == targetTag {
			return bo.Uint32(tiffData[entryOffset+8 : entryOffset+12]), nil
		}
	}
	return 0, fmt.Errorf("tag 0x%04X not found", targetTag)
}

func findDateTag(tiffData []byte, bo binary.ByteOrder, ifdOffset int, targetTag uint16) (time.Time, error) {
	if ifdOffset+2 > len(tiffData) {
		return time.Time{}, fmt.Errorf("IFD offset out of range")
	}
	numEntries := int(bo.Uint16(tiffData[ifdOffset : ifdOffset+2]))
	for i := 0; i < numEntries; i++ {
		entryOffset := ifdOffset + 2 + i*12
		if entryOffset+12 > len(tiffData) {
			break
		}
		tag := bo.Uint16(tiffData[entryOffset : entryOffset+2])
		if tag != targetTag {
			continue
		}
		count := int(bo.Uint32(tiffData[entryOffset+4 : entryOffset+8]))
		if count < 19 {
			continue
		}
		valueOffset := int(bo.Uint32(tiffData[entryOffset+8 : entryOffset+12]))
		if valueOffset+19 > len(tiffData) {
			continue
		}
		dateStr := string(bytes.TrimRight(tiffData[valueOffset:valueOffset+19], "\x00"))
		// EXIF 日期格式: "2006:01:02 15:04:05"
		t, err := time.ParseInLocation("2006:01:02 15:04:05", dateStr, time.Local)
		if err != nil {
			continue
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("date tag not found")
}
