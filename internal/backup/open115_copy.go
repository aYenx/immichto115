package backup

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aYenx/immichto115/internal/config"
	"github.com/aYenx/immichto115/internal/manifest"
	"github.com/aYenx/immichto115/internal/open115"
	"github.com/aYenx/immichto115/internal/open115crypt"
)

type LogEmitter func(stream string, text string)

type Open115CopySummary struct {
	Scanned       int
	Uploaded      int
	Skipped       int
	ScanSkipped   int      // files/dirs skipped due to I/O errors during scan
	MissingDirs   []string // configured dirs that were not found (mount loss etc.)
	ScanComplete  bool     // true only when scan had no errors and no missing dirs
	DeleteSkipped bool     // true if sync-delete was blocked for safety
	DeleteReason  string   // human-readable reason sync-delete was skipped
	Deleted       int      // number of remote files deleted in sync mode
	PendingDelete int      // number of files newly marked for pending deletion
	DryRun        bool     // true if sync-delete ran in dry-run mode
}

// numWorkers 并发上传 worker 数，模仿 rclone --transfers 4。
const numWorkers = 4

// maxTotalErrors 累计错误数上限，超过后取消所有 workers。
const maxTotalErrors = 20

type Open115CopyRunner struct {
	cfgMgr   *config.Manager
	service  *open115.Service
	backend  *Open115Backend
	emit     LogEmitter
	manifest manifest.Store
}

func defaultManifestPath(cfg config.AppConfig, cfgPath string) string {
	baseDir := "."
	if strings.TrimSpace(cfgPath) != "" {
		baseDir = filepath.Dir(cfgPath)
	}
	if mp := strings.TrimSpace(cfg.Backup.ManifestPath); mp != "" {
		if filepath.IsAbs(mp) {
			return mp
		}
		// 相对路径按配置文件所在目录解析，避免依赖进程 cwd
		return filepath.Join(baseDir, mp)
	}
	return filepath.Join(baseDir, "manifest.db")
}

func NewOpen115CopyRunner(cfgMgr *config.Manager, service *open115.Service, emit LogEmitter) (*Open115CopyRunner, error) {
	if cfgMgr == nil {
		return nil, fmt.Errorf("open115 copy runner 初始化失败: cfgMgr 为空")
	}
	if service == nil {
		return nil, fmt.Errorf("open115 copy runner 初始化失败: service 为空")
	}
	cfg := cfgMgr.Get()
	manifestPath := defaultManifestPath(cfg, cfgMgr.FilePath())
	store, err := manifest.NewSQLiteStore(manifestPath)
	if err != nil {
		return nil, err
	}
	if err := store.Init(context.Background()); err != nil {
		return nil, err
	}
	return &Open115CopyRunner{
		cfgMgr:   cfgMgr,
		service:  service,
		backend:  NewOpen115Backend(service),
		emit:     emit,
		manifest: store,
	}, nil
}

func (r *Open115CopyRunner) Close() error {
	if r == nil || r.manifest == nil {
		return nil
	}
	return r.manifest.Close()
}

func (r *Open115CopyRunner) log(stream, text string) {
	if r != nil && r.emit != nil {
		r.emit(stream, text)
	}
}

type localFile struct {
	AbsPath string
	RelPath string
	Size    int64
	MTime   int64
}

// ScanResult aggregates the outcome of scanning a local directory tree.
type ScanResult struct {
	Files      []localFile
	Skipped    int      // files/dirs skipped due to permission or I/O errors
	MissingDir string   // non-empty if the configured dir did not exist
	Complete   bool     // true only when Skipped==0 && MissingDir==""
}

// scanLocalFiles walks baseDir and returns a ScanResult. When the directory
// does not exist (e.g. mount point lost), MissingDir is set and Complete is
// false so that callers can block sync-delete.
func scanLocalFiles(baseDir string, prefix string) (ScanResult, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return ScanResult{Complete: true}, nil
	}
	info, err := os.Stat(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置了目录但目录不存在（挂载点丢失等），视为扫描不完整
			log.Printf("[backup] scan skipped: configured directory does not exist: %s", baseDir)
			return ScanResult{Skipped: 1, MissingDir: baseDir}, nil
		}
		return ScanResult{}, err
	}
	if !info.IsDir() {
		return ScanResult{}, fmt.Errorf("%s 不是目录", baseDir)
	}
	files := make([]localFile, 0)
	skippedCount := 0
	err = filepath.WalkDir(baseDir, func(current string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			skippedCount++
			// 权限不足等非致命错误：跳过该文件/目录，继续扫描
			if os.IsPermission(walkErr) {
				log.Printf("[backup] scan skipped (permission denied): %s", current)
				return nil
			}
			// 其他错误也尝试跳过，避免中断整个扫描
			log.Printf("[backup] scan skipped (error): %s: %v", current, walkErr)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		stat, err := d.Info()
		if err != nil {
			skippedCount++
			log.Printf("[backup] scan skipped (info error): %s: %v", current, err)
			return nil
		}
		rel, err := filepath.Rel(baseDir, current)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		files = append(files, localFile{
			AbsPath: current,
			RelPath: path.Join(prefix, rel),
			Size:    stat.Size(),
			MTime:   stat.ModTime().Unix(),
		})
		return nil
	})
	if err != nil {
		return ScanResult{}, err
	}
	return ScanResult{
		Files:    files,
		Skipped:  skippedCount,
		Complete: skippedCount == 0,
	}, nil
}

// uploadResult 是 worker 上传单个文件后的结果。
type uploadResult struct {
	file   localFile
	record *manifest.FileRecord // nil 表示失败
	err    error
}

// uploadChangedFiles 使用 numWorkers 个并发 worker 上传已变化的文件。
// 模仿 rclone --transfers 4 的 worker pool 模式。
//
// 架构：
//
//	             ┌─── worker 1 ──→ upload ──→ result
//	files ──→ jobs ├─── worker 2 ──→ upload ──→ result ──→ collector
//	             ├─── worker 3 ──→ upload ──→ result     (主goroutine)
//	             └─── worker 4 ──→ upload ──→ result
func (r *Open115CopyRunner) uploadChangedFiles(ctx context.Context, files []localFile, remoteRoot string) (int, int, error) {
	// 先过滤出需要上传的文件（manifest 检查在主 goroutine 串行完成）
	toUpload := make([]localFile, 0, len(files))
	skipped := 0
	for _, file := range files {
		if ctx.Err() != nil {
			return 0, skipped, ctx.Err()
		}
		existingRec, err := r.manifest.Get(ctx, file.RelPath)
		if err != nil {
			return 0, skipped, err
		}
		if existingRec != nil && !existingRec.Deleted && existingRec.Size == file.Size && existingRec.MTime == file.MTime {
			skipped++
			continue
		}
		toUpload = append(toUpload, file)
	}

	if len(toUpload) == 0 {
		return 0, skipped, nil
	}

	r.log("stdout", fmt.Sprintf("[immichto115] Open115 需上传 %d 个文件（跳过 %d 未变化），使用 %d 并发 worker", len(toUpload), skipped, numWorkers))

	// 创建可取消的子 context
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	jobs := make(chan localFile, numWorkers*2)
	results := make(chan uploadResult, numWorkers*2)
	var totalErrors atomic.Int64
	var (
		firstErrOnce sync.Once
		firstErrMu   sync.Mutex
		firstErr     error
	)
	setFirstErr := func(err error) {
		firstErrOnce.Do(func() {
			firstErrMu.Lock()
			firstErr = err
			firstErrMu.Unlock()
		})
	}
	getFirstErr := func() error {
		firstErrMu.Lock()
		defer firstErrMu.Unlock()
		return firstErr
	}

	encCfg := open115crypt.FromAppConfig(r.cfgMgr.Get())

	// 启动 workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for file := range jobs {
				if workerCtx.Err() != nil {
					return
				}
				result := r.uploadOneFile(workerCtx, file, remoteRoot, encCfg)
				if result.err != nil && workerCtx.Err() == nil {
					n := totalErrors.Add(1)
					setFirstErr(result.err)
					if n >= maxTotalErrors {
						r.log("stderr", fmt.Sprintf("[immichto115] 累计 %d 个文件上传失败，中止备份", n))
						workerCancel()
						return
					}
				}
				select {
				case results <- result:
				case <-workerCtx.Done():
					return
				}
			}
		}(i)
	}

	// 发送 jobs（单独 goroutine）
	go func() {
		defer close(jobs)
		for _, file := range toUpload {
			select {
			case jobs <- file:
			case <-workerCtx.Done():
				return
			}
		}
	}()

	// workers 全部结束后关闭 results
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果（主 goroutine），串行写 manifest
	uploaded := 0
	for result := range results {
		if result.err != nil {
			// 错误已在 worker 中记录和计数
			continue
		}
		if result.record != nil {
			if err := r.manifest.Put(ctx, result.record); err != nil {
				workerCancel()
				// drain remaining results
				for range results {
				}
				return uploaded, skipped, err
			}
			uploaded++
		}
	}

	// 检查是否因错误中止
	if n := totalErrors.Load(); n >= maxTotalErrors {
		if e := getFirstErr(); e != nil {
			return uploaded, skipped, fmt.Errorf("累计 %d 个文件上传失败，中止备份；首个错误: %w", n, e)
		}
	}

	// 返回首个非致命错误（如果有）
	if e := getFirstErr(); e != nil {
		return uploaded, skipped, e
	}

	return uploaded, skipped, nil
}

// uploadOneFile 上传单个文件，由 worker goroutine 调用。
func (r *Open115CopyRunner) uploadOneFile(ctx context.Context, file localFile, remoteRoot string, encCfg open115crypt.Config) uploadResult {
	remotePath := path.Join(remoteRoot, file.RelPath)
	uploadPath := file.AbsPath
	record := &manifest.FileRecord{
		Path:           file.RelPath,
		Size:           file.Size,
		MTime:          file.MTime,
		LastUploadedAt: time.Now().Unix(),
		Deleted:        false,
		RemotePath:     remotePath,
	}

	var cleanupPath string
	if encCfg.Enabled {
		remotePath = remotePath + ".enc"
		record.Encrypted = true
		record.RemotePath = remotePath
		if strings.TrimSpace(encCfg.Mode) == "stream" {
			record.EncryptionVersion = "v2-stream"
		} else {
			encFile, err := open115crypt.EncryptFileToTemp(file.AbsPath, encCfg)
			if err != nil {
				r.log("stderr", fmt.Sprintf("[immichto115] Open115 加密文件失败（跳过）: %s: %v", file.AbsPath, err))
				return uploadResult{file: file, err: err}
			}
			uploadPath = encFile.TempPath
			cleanupPath = encFile.TempPath
			record.EncryptedSize = encFile.EncryptedSize
			record.EncryptionVersion = encFile.Version
		}
	}

	r.log("stdout", fmt.Sprintf("[immichto115] Open115 上传文件: %s -> %s", uploadPath, remotePath))

	var uploadErr error
	if encCfg.Enabled && strings.TrimSpace(encCfg.Mode) == "stream" {
		uploadErr = r.backend.UploadEncryptedStream(ctx, file.AbsPath, remotePath, encCfg)
	} else {
		uploadErr = r.backend.UploadFile(ctx, uploadPath, remotePath)
	}
	if cleanupPath != "" {
		_ = open115crypt.CleanupTempFile(cleanupPath)
	}
	if uploadErr != nil {
		if ctx.Err() != nil {
			return uploadResult{file: file, err: ctx.Err()}
		}
		r.log("stderr", fmt.Sprintf("[immichto115] Open115 上传失败（跳过）: %s: %v", file.RelPath, uploadErr))
		return uploadResult{file: file, err: uploadErr}
	}

	return uploadResult{file: file, record: record}
}

func (r *Open115CopyRunner) listAllManifestRecords(ctx context.Context) ([]manifest.FileRecord, error) {
	if r == nil || r.manifest == nil {
		return nil, nil
	}
	const pageSize = 1000
	all := make([]manifest.FileRecord, 0, pageSize)
	for offset := 0; ; offset += pageSize {
		items, err := r.manifest.List(ctx, pageSize, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if len(items) < pageSize {
			break
		}
	}
	return all, nil
}

// maxDeleteRatio is the safety threshold: if more than 20% of active manifest
// records would be deleted, sync-delete is blocked to prevent data loss from
// incomplete scans (e.g. mount point temporarily missing).
const maxDeleteRatio = 0.20

// syncDeleteResult carries the outcome of a sync-delete attempt.
type syncDeleteResult struct {
	Deleted       int
	PendingMarked int
	Skipped       bool
	SkipReason    string
	DryRun        bool
}

// syncDeleteOpts carries options for syncDeleteRemoved.
type syncDeleteOpts struct {
	GracePeriod time.Duration
	DryRun      bool
}

// parseGracePeriod parses a Go duration string, defaulting to 24h on error.
func parseGracePeriod(s string) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return 24 * time.Hour
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 24 * time.Hour
	}
	return d
}

func (r *Open115CopyRunner) syncDeleteRemoved(ctx context.Context, currentFiles []localFile, remoteRoot string, opts syncDeleteOpts) (syncDeleteResult, error) {
	if r == nil || r.manifest == nil {
		return syncDeleteResult{}, nil
	}
	existing := make(map[string]struct{}, len(currentFiles))
	for _, file := range currentFiles {
		existing[file.RelPath] = struct{}{}
	}
	records, err := r.listAllManifestRecords(ctx)
	if err != nil {
		return syncDeleteResult{}, err
	}

	// Count active (non-deleted) records and delete candidates
	var activeCount, candidateCount int
	for _, rec := range records {
		if rec.Deleted {
			continue
		}
		activeCount++
		if _, ok := existing[rec.Path]; !ok {
			candidateCount++
		}
	}

	// Delete protection threshold: refuse if too many files would be deleted
	if activeCount > 0 && float64(candidateCount)/float64(activeCount) > maxDeleteRatio {
		reason := fmt.Sprintf("待删除 %d 条（占 %d 条总记录的 %.0f%%），超出 %.0f%% 安全阈值，已跳过删除阶段",
			candidateCount, activeCount,
			float64(candidateCount)/float64(activeCount)*100,
			maxDeleteRatio*100)
		r.log("stderr", "[immichto115] Open115 sync 安全保护："+reason)
		return syncDeleteResult{Skipped: true, SkipReason: reason}, nil
	}

	now := time.Now()

	// --- Phase 1: Mark candidates / clear reappeared ---
	pendingMarked := 0
	for _, rec := range records {
		if ctx.Err() != nil {
			return syncDeleteResult{PendingMarked: pendingMarked}, ctx.Err()
		}
		if rec.Deleted {
			continue
		}
		_, stillExists := existing[rec.Path]

		if !stillExists && rec.PendingDeleteAt == 0 {
			// File missing locally and not yet pending → mark pending
			if err := r.manifest.MarkPendingDelete(ctx, rec.Path, now.Unix()); err != nil {
				return syncDeleteResult{PendingMarked: pendingMarked}, err
			}
			pendingMarked++
			r.log("stdout", fmt.Sprintf("[immichto115] Open115 sync 标记待删除: %s (grace period: %s)", rec.Path, opts.GracePeriod))
		} else if stillExists && rec.PendingDeleteAt > 0 {
			// File reappeared → clear pending mark
			if err := r.manifest.ClearPendingDelete(ctx, rec.Path); err != nil {
				return syncDeleteResult{PendingMarked: pendingMarked}, err
			}
			r.log("stdout", fmt.Sprintf("[immichto115] Open115 sync 文件重新出现，取消待删除: %s", rec.Path))
		}
	}

	// --- Phase 2: Confirm and delete files past grace period ---
	cutoff := now.Add(-opts.GracePeriod).Unix()
	pendingRecords, err := r.manifest.ListPendingDeletes(ctx, cutoff)
	if err != nil {
		return syncDeleteResult{PendingMarked: pendingMarked}, err
	}

	deleted := 0
	for _, rec := range pendingRecords {
		if ctx.Err() != nil {
			return syncDeleteResult{Deleted: deleted, PendingMarked: pendingMarked, DryRun: opts.DryRun}, ctx.Err()
		}

		remotePath := rec.RemotePath
		if strings.TrimSpace(remotePath) == "" {
			remotePath = path.Join(remoteRoot, rec.Path)
		}

		if opts.DryRun {
			r.log("stdout", fmt.Sprintf("[immichto115] Open115 sync [DRY-RUN] 将删除远端文件: %s (pending since %s)",
				remotePath, time.Unix(rec.PendingDeleteAt, 0).Format("2006-01-02 15:04:05")))
			deleted++
			continue
		}

		r.log("stdout", fmt.Sprintf("[immichto115] Open115 sync 删除远端文件: %s (pending since %s)",
			remotePath, time.Unix(rec.PendingDeleteAt, 0).Format("2006-01-02 15:04:05")))
		if err := r.backend.DeleteRemote(ctx, remotePath); err != nil {
			if strings.Contains(err.Error(), "远端条目不存在") {
				r.log("stderr", fmt.Sprintf("[immichto115] Open115 sync 删除跳过：远端文件不存在，按已删除处理: %s", remotePath))
			} else {
				return syncDeleteResult{Deleted: deleted, PendingMarked: pendingMarked, DryRun: opts.DryRun}, err
			}
		}
		if err := r.manifest.MarkDeleted(ctx, rec.Path, true); err != nil {
			return syncDeleteResult{Deleted: deleted, PendingMarked: pendingMarked, DryRun: opts.DryRun}, err
		}
		deleted++
	}
	return syncDeleteResult{Deleted: deleted, PendingMarked: pendingMarked, DryRun: opts.DryRun}, nil
}

func (r *Open115CopyRunner) Run(ctx context.Context) (*Open115CopySummary, error) {
	if r == nil || r.cfgMgr == nil || r.service == nil || r.backend == nil {
		return nil, fmt.Errorf("open115 copy runner 未正确初始化")
	}
	cfg := r.cfgMgr.Get()
	if err := r.backend.TestConnection(ctx); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	remoteRoot := cfg.Backup.RemoteDir
	if strings.TrimSpace(remoteRoot) == "" {
		remoteRoot = "/immich-backup"
	}

	// Phase 1: Scan local files
	libScan, err := scanLocalFiles(cfg.Backup.LibraryDir, "library")
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	bkpScan, err := scanLocalFiles(cfg.Backup.BackupsDir, "backups")
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Merge scan results
	scanSkipped := libScan.Skipped + bkpScan.Skipped
	scanComplete := libScan.Complete && bkpScan.Complete
	var missingDirs []string
	if libScan.MissingDir != "" {
		missingDirs = append(missingDirs, libScan.MissingDir)
	}
	if bkpScan.MissingDir != "" {
		missingDirs = append(missingDirs, bkpScan.MissingDir)
	}

	allFiles := make([]localFile, 0, len(libScan.Files)+len(bkpScan.Files))
	allFiles = append(allFiles, libScan.Files...)
	allFiles = append(allFiles, bkpScan.Files...)

	if len(allFiles) == 0 {
		r.log("stderr", "[immichto115] Open115 未发现可备份文件")
	} else {
		r.log("stdout", fmt.Sprintf("[immichto115] Open115 增量扫描完成，共 %d 个文件待检查", len(allFiles)))
	}
	if scanSkipped > 0 {
		r.log("stderr", fmt.Sprintf("[immichto115] 扫描期间跳过 %d 个无法读取的文件/目录", scanSkipped))
	}
	for _, d := range missingDirs {
		r.log("stderr", fmt.Sprintf("[immichto115] 配置的目录不存在（挂载点丢失？）: %s", d))
	}

	summary := &Open115CopySummary{
		Scanned:      len(allFiles),
		ScanSkipped:  scanSkipped,
		MissingDirs:  missingDirs,
		ScanComplete: scanComplete,
	}

	// Phase 2: Upload changed files
	if len(allFiles) > 0 {
		uploaded, skipped, err := r.uploadChangedFiles(ctx, allFiles, remoteRoot)
		if err != nil {
			return nil, err
		}
		summary.Uploaded = uploaded
		summary.Skipped = skipped
	}

	// Phase 3: Sync-delete (if applicable)
	if strings.TrimSpace(cfg.Backup.Mode) == "sync" {
		if !cfg.Backup.AllowRemoteDelete {
			reason := "当前为 sync 模式，但未启用 allow_remote_delete，已跳过远端删除阶段"
			r.log("stderr", "[immichto115] "+reason)
			summary.DeleteSkipped = true
			summary.DeleteReason = reason
		} else if !scanComplete {
			reason := fmt.Sprintf("当前为 sync 模式，但扫描不完整（跳过 %d 个文件/目录），为避免误删远端数据，已跳过远端删除阶段", scanSkipped)
			r.log("stderr", "[immichto115] "+reason)
			summary.DeleteSkipped = true
			summary.DeleteReason = reason
		} else {
			opts := syncDeleteOpts{
				GracePeriod: parseGracePeriod(cfg.Backup.SyncDeleteGracePeriod),
				DryRun:      cfg.Backup.SyncDeleteDryRun,
			}
			if opts.DryRun {
				r.log("stdout", "[immichto115] Open115 sync 删除阶段运行在 DRY-RUN 模式（不会实际删除）")
			}
			r.log("stdout", fmt.Sprintf("[immichto115] Open115 sync 删除阶段 grace period: %s", opts.GracePeriod))
			delResult, err := r.syncDeleteRemoved(ctx, allFiles, remoteRoot, opts)
			if err != nil {
				return nil, err
			}
			summary.Deleted = delResult.Deleted
			summary.PendingDelete = delResult.PendingMarked
			summary.DryRun = delResult.DryRun
			summary.DeleteSkipped = delResult.Skipped
			summary.DeleteReason = delResult.SkipReason
			if !delResult.Skipped {
				r.log("stdout", fmt.Sprintf("[immichto115] Open115 sync 删除阶段执行完成 (新标记待删除: %d, 确认删除: %d, dry-run: %v)",
					delResult.PendingMarked, delResult.Deleted, delResult.DryRun))
			}
		}
	}

	// 备份结束后清理目录 ID 缓存
	r.backend.Uploader().ClearDirCache()

	r.log("stdout", fmt.Sprintf("[immichto115] Open115 copy 完成：上传 %d，跳过 %d", summary.Uploaded, summary.Skipped))
	return summary, nil
}
