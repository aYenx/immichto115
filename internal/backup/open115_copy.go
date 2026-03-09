package backup

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aYenx/immichto115/internal/config"
	"github.com/aYenx/immichto115/internal/manifest"
	"github.com/aYenx/immichto115/internal/open115"
	"github.com/aYenx/immichto115/internal/open115crypt"
)

type LogEmitter func(stream string, text string)

type Open115CopySummary struct {
	Scanned  int
	Uploaded int
	Skipped  int
}

type Open115CopyRunner struct {
	cfgMgr   *config.Manager
	service  *open115.Service
	backend  *Open115Backend
	emit     LogEmitter
	manifest manifest.Store
}

func defaultManifestPath(cfg config.AppConfig, cfgPath string) string {
	if strings.TrimSpace(cfg.Backup.ManifestPath) != "" {
		return strings.TrimSpace(cfg.Backup.ManifestPath)
	}
	baseDir := "."
	if strings.TrimSpace(cfgPath) != "" {
		baseDir = filepath.Dir(cfgPath)
	}
	return filepath.Join(baseDir, "manifest.db")
}

func NewOpen115CopyRunner(cfgMgr *config.Manager, service *open115.Service, emit LogEmitter) (*Open115CopyRunner, error) {
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

func scanLocalFiles(baseDir string, prefix string) ([]localFile, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return nil, nil
	}
	info, err := os.Stat(baseDir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s 不是目录", baseDir)
	}
	files := make([]localFile, 0)
	err = filepath.WalkDir(baseDir, func(current string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		stat, err := d.Info()
		if err != nil {
			return err
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
	return files, err
}

func (r *Open115CopyRunner) uploadChangedFiles(ctx context.Context, files []localFile, remoteRoot string) (int, int, error) {
	uploaded := 0
	skipped := 0
	for i, file := range files {
		if ctx.Err() != nil {
			return uploaded, skipped, ctx.Err()
		}
		// 非首个上传文件时，加入节流延迟以避免触发 115 API 限速
		if i > 0 && uploaded > 0 {
			select {
			case <-ctx.Done():
				return uploaded, skipped, ctx.Err()
			case <-time.After(800 * time.Millisecond):
			}
		}
		existingRec, err := r.manifest.Get(ctx, file.RelPath)
		if err != nil {
			return uploaded, skipped, err
		}
		if existingRec != nil && !existingRec.Deleted && existingRec.Size == file.Size && existingRec.MTime == file.MTime {
			skipped++
			continue
		}
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
		encCfg := open115crypt.FromAppConfig(r.cfgMgr.Get())
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
					return uploaded, skipped, err
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
		if uploadErr != nil {
			if cleanupPath != "" {
				_ = open115crypt.CleanupTempFile(cleanupPath)
			}
			return uploaded, skipped, uploadErr
		}
		if cleanupPath != "" {
			_ = open115crypt.CleanupTempFile(cleanupPath)
		}
		if err := r.manifest.Put(ctx, record); err != nil {
			return uploaded, skipped, err
		}
		uploaded++
	}
	return uploaded, skipped, nil
}

func (r *Open115CopyRunner) syncDeleteRemoved(ctx context.Context, currentFiles []localFile, remoteRoot string) error {
	if r == nil || r.manifest == nil {
		return nil
	}
	existing := make(map[string]struct{}, len(currentFiles))
	for _, file := range currentFiles {
		existing[file.RelPath] = struct{}{}
	}
	records, err := r.manifest.List(ctx, 100000, 0)
	if err != nil {
		return err
	}
	for _, rec := range records {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if rec.Deleted {
			continue
		}
		if _, ok := existing[rec.Path]; ok {
			continue
		}
		remotePath := rec.RemotePath
		if strings.TrimSpace(remotePath) == "" {
			remotePath = path.Join(remoteRoot, rec.Path)
		}
		r.log("stdout", fmt.Sprintf("[immichto115] Open115 sync 删除远端文件: %s", remotePath))
		if err := r.backend.DeleteRemote(ctx, remotePath); err != nil {
			return err
		}
		if err := r.manifest.MarkDeleted(ctx, rec.Path, true); err != nil {
			return err
		}
	}
	return nil
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
	libraryFiles, err := scanLocalFiles(cfg.Backup.LibraryDir, "library")
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	backupFiles, err := scanLocalFiles(cfg.Backup.BackupsDir, "backups")
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	allFiles := append(libraryFiles, backupFiles...)
	if len(allFiles) == 0 {
		r.log("stderr", "[immichto115] Open115 未发现可备份文件")
		return &Open115CopySummary{}, nil
	}
	r.log("stdout", fmt.Sprintf("[immichto115] Open115 增量扫描完成，共 %d 个文件待检查", len(allFiles)))
	uploaded, skipped, err := r.uploadChangedFiles(ctx, allFiles, remoteRoot)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Backup.Mode) == "sync" {
		if !cfg.Backup.AllowRemoteDelete {
			r.log("stderr", "[immichto115] 当前为 sync 模式，但未启用 allow_remote_delete，已跳过远端删除阶段")
		} else {
			if err := r.syncDeleteRemoved(ctx, allFiles, remoteRoot); err != nil {
				return nil, err
			}
			r.log("stdout", "[immichto115] Open115 sync 删除阶段执行完成")
		}
	}
	r.log("stdout", fmt.Sprintf("[immichto115] Open115 copy 完成：上传 %d，跳过 %d", uploaded, skipped))
	return &Open115CopySummary{Scanned: len(allFiles), Uploaded: uploaded, Skipped: skipped}, nil
}
