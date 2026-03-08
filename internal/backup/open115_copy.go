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
	for _, file := range files {
		if ctx.Err() != nil {
			return uploaded, skipped, ctx.Err()
		}
		record, err := r.manifest.Get(ctx, file.RelPath)
		if err != nil {
			return uploaded, skipped, err
		}
		if record != nil && !record.Deleted && record.Size == file.Size && record.MTime == file.MTime {
			skipped++
			continue
		}
		remotePath := path.Join(remoteRoot, file.RelPath)
		r.log("stdout", fmt.Sprintf("[immichto115] Open115 上传文件: %s -> %s", file.AbsPath, remotePath))
		if err := r.backend.UploadFile(ctx, file.AbsPath, remotePath); err != nil {
			return uploaded, skipped, err
		}
		now := time.Now().Unix()
		if err := r.manifest.Put(ctx, &manifest.FileRecord{
			Path:           file.RelPath,
			Size:           file.Size,
			MTime:          file.MTime,
			LastUploadedAt: now,
			Deleted:        false,
		}); err != nil {
			return uploaded, skipped, err
		}
		uploaded++
	}
	return uploaded, skipped, nil
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
	r.log("stdout", fmt.Sprintf("[immichto115] Open115 copy 完成：上传 %d，跳过 %d", uploaded, skipped))
	return &Open115CopySummary{Scanned: len(allFiles), Uploaded: uploaded, Skipped: skipped}, nil
}
