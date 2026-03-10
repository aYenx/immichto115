package api

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/aYenx/immichto115/internal/backup"
	"github.com/aYenx/immichto115/internal/config"
	appCron "github.com/aYenx/immichto115/internal/cron"
	"github.com/aYenx/immichto115/internal/notify"
	"github.com/aYenx/immichto115/internal/open115"
	"github.com/aYenx/immichto115/internal/open115crypt"
	"github.com/aYenx/immichto115/internal/rclone"
	"github.com/gin-gonic/gin"
)

const maskedSecret = "********"

// maskSecret 如果字段非空则替换为遮蔽值。
func maskSecret(field *string) {
	if *field != "" {
		*field = maskedSecret
	}
}

// restoreSecret 如果新值是遮蔽值，则从旧配置还原。
// 空字符串视为用户主动清除该值，不做还原。
func restoreSecret(newField *string, oldValue string) {
	if *newField == maskedSecret {
		*newField = oldValue
	}
}

func normalizeRemoteDir(remoteDir string) string {
	cleaned := path.Clean("/" + strings.TrimSpace(remoteDir))
	if cleaned == "." || cleaned == "" {
		return "/"
	}
	return cleaned
}

func normalizeCronExpression(expr string) string {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return ""
	}
	parts := strings.Fields(expr)
	if len(parts) == 5 {
		return "0 " + expr
	}
	return expr
}

// Server 持有所有 API 依赖。
type Server struct {
	Runner    *rclone.Runner
	Hub       *Hub
	Config    *config.Manager
	Open115   *open115.Service
	Scheduler *appCron.Scheduler

	authSessionMu sync.RWMutex
	authSessions  map[string]*open115.AuthSession

	backupMu      sync.RWMutex
	backupCancel  context.CancelFunc
	backupActive  bool
	backupTrigger string
}

// NewServer 创建 API Server 实例。
func NewServer(cfgMgr *config.Manager) *Server {
	s := &Server{
		Runner:       rclone.NewRunner(),
		Hub:          NewHub(),
		Config:       cfgMgr,
		Open115:      open115.NewService(cfgMgr),
		authSessions: make(map[string]*open115.AuthSession),
	}

	// 定时任务：触发时执行备份
	s.Scheduler = appCron.NewScheduler(func() {
		s.triggerBackup("定时任务")
	})

	return s
}

// triggerBackup 是定时任务和手动触发共用的备份逻辑。
func (s *Server) triggerBackup(trigger string) {
	jobCtx, ok := s.beginBackupJob(trigger)
	if !ok {
		log.Println("[backup] skipped: already running")
		s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 已有备份任务正在运行，本次请求已跳过"})
		return
	}
	defer s.finishBackupJob()
	s.runBackupBody(jobCtx)
}

// runBackupBody 是备份的实际执行体，供 triggerBackup 和 handleBackupStart 共用。
// 调用前必须已通过 beginBackupJob 成功占位。
func (s *Server) runBackupBody(jobCtx context.Context) {

	cfg := s.Config.Get()
	trigger := s.currentBackupTrigger()
	backupMode := cfg.Backup.Mode
	if backupMode != "sync" {
		backupMode = "copy" // 默认增量备份
	}
	modeLabel := "增量备份（copy）"
	if backupMode == "sync" {
		modeLabel = "镜像同步（sync）"
	}
	completedStages := make([]string, 0, 2)
	plannedStages := make([]string, 0, 2)
	summarizeProgress := func(failedStage string) string {
		parts := make([]string, 0, 3)
		if len(plannedStages) > 0 {
			parts = append(parts, "计划阶段："+strings.Join(plannedStages, "、"))
		}
		if len(completedStages) > 0 {
			parts = append(parts, "已完成："+strings.Join(completedStages, "、"))
		} else {
			parts = append(parts, "已完成：无")
		}
		if failedStage != "" {
			parts = append(parts, "失败阶段："+failedStage)
		}
		return strings.Join(parts, "；")
	}
	s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 备份任务已启动，正在检查配置与目标路径..."})
	s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 触发方式: " + trigger})
	s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 备份模式: " + modeLabel})

	provider := strings.TrimSpace(cfg.Provider)
	if provider == "" {
		provider = "webdav"
	}
	if provider == "open115" {
		s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 当前使用 115 Open 模式，开始执行增量 copy 备份"})
		runner, err := backup.NewOpen115CopyRunner(s.Config, s.Open115, func(stream string, text string) {
			s.Hub.Broadcast(rclone.LogLine{Stream: stream, Text: text})
		})
		if err != nil {
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 初始化 Open115 备份执行器失败：" + err.Error()})
			return
		}
		defer func() { _ = runner.Close() }()
		summary, err := runner.Run(jobCtx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				msg := "[immichto115] Open115 备份已手动停止"
				s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: msg})
				s.sendBackupNotify(cfg, notify.BackupNotification{
					Success:    false,
					Trigger:    trigger,
					Mode:       modeLabel,
					Stage:      "Open115 备份",
					RemotePath: cfg.Backup.RemoteDir,
					Detail:     "任务已被手动停止",
				})
				return
			}
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] Open115 备份失败：" + err.Error()})
			s.sendBackupNotify(cfg, notify.BackupNotification{
				Success:    false,
				Trigger:    trigger,
				Mode:       modeLabel,
				Stage:      "Open115 备份",
				RemotePath: cfg.Backup.RemoteDir,
				Detail:     err.Error(),
			})
			return
		}
		detail := "Open115 copy 增量备份执行完成"
		if summary != nil {
			detail = fmt.Sprintf("Open115 copy 增量备份执行完成；扫描 %d，上传 %d，跳过 %d", summary.Scanned, summary.Uploaded, summary.Skipped)
			log.Printf("[backup] %s", detail)
		}
		s.sendBackupNotify(cfg, notify.BackupNotification{
			Success:    true,
			Trigger:    trigger,
			Mode:       modeLabel,
			Stage:      "Open115 备份",
			RemotePath: cfg.Backup.RemoteDir,
			Detail:     detail,
		})
		return
	}

	s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 正在生成临时 rclone 配置..."})

	// 生成临时 rclone.conf
	confPath, err := config.GenerateRcloneConf(cfg)
	if err != nil {
		log.Printf("[backup] failed to generate rclone.conf: %v", err)
		s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 生成 rclone 配置失败：" + err.Error()})
		s.sendBackupNotify(cfg, notify.BackupNotification{
			Success:    false,
			Trigger:    trigger,
			Mode:       modeLabel,
			Stage:      "准备阶段",
			RemotePath: config.GetRemoteName(cfg),
			Detail:     "生成 rclone 配置失败：" + err.Error(),
		})
		return
	}
	s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] rclone 配置生成完成，开始执行同步任务"})
	defer func() {
		config.CleanupRcloneConf(confPath)
		s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 临时 rclone 配置已清理"})
	}()

	remote := config.GetRemoteName(cfg)
	if cfg.Encrypt.Enabled {
		s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 已启用加密远端，数据将同步到加密目录"})
	} else {
		s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 未启用加密，将分别同步照片库与数据库备份目录"})
	}

	hasSyncTarget := false

	libraryDir := strings.TrimSpace(cfg.Backup.LibraryDir)
	backupsDir := strings.TrimSpace(cfg.Backup.BackupsDir)

	// 备份 Library 目录
	if libraryDir != "" {
		plannedStages = append(plannedStages, "照片库备份")
		hasSyncTarget = true
		if jobCtx.Err() != nil {
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 任务已停止，照片库备份尚未开始"})
			return
		}
		dest := remote
		if !cfg.Encrypt.Enabled {
			dest = remote + "/library"
		}
		s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 开始备份照片库目录: " + libraryDir})
		s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 目标位置: " + dest})
		// 预创建远端目录，避免某些 WebDAV 服务器返回 404
		if err := rclone.Mkdir(dest, confPath); err != nil {
			log.Printf("[backup] mkdir warning (non-fatal): %v", err)
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 预创建远端目录失败（非致命，继续尝试备份）: " + err.Error()})
		}
		logCh, errCh, err := s.Runner.Run(backupMode, libraryDir, dest, nil, confPath)
		if err != nil {
			log.Printf("[backup] failed to start library backup: %v", err)
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 无法启动照片库备份: " + err.Error()})
			s.sendBackupNotify(cfg, notify.BackupNotification{
				Success:    false,
				Trigger:    trigger,
				Mode:       modeLabel,
				Stage:      "照片库备份",
				RemotePath: dest,
				Detail:     summarizeProgress("照片库备份") + "；启动失败：" + err.Error(),
			})
			return
		}
		s.Hub.BroadcastFromChannel(logCh) // 阻塞直到完成
		if runErr := <-errCh; runErr != nil {
			if errors.Is(runErr, rclone.ErrCancelled) {
				log.Printf("[backup] library backup cancelled by user")
				s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 照片库备份已手动停止"})
				s.sendBackupNotify(cfg, notify.BackupNotification{
					Success:    false,
					Trigger:    trigger,
					Mode:       modeLabel,
					Stage:      "照片库备份",
					RemotePath: dest,
					Detail:     summarizeProgress("照片库备份") + "；任务已被手动停止",
				})
				return
			}
			log.Printf("[backup] library backup failed: %v", runErr)
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 照片库备份失败: " + runErr.Error()})
			s.sendBackupNotify(cfg, notify.BackupNotification{
				Success:    false,
				Trigger:    trigger,
				Mode:       modeLabel,
				Stage:      "照片库备份",
				RemotePath: dest,
				Detail:     summarizeProgress("照片库备份") + "；" + runErr.Error(),
			})
			return
		}
		completedStages = append(completedStages, "照片库备份")
		s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 照片库目录备份阶段已结束"})
		if jobCtx.Err() != nil {
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 任务已停止，照片库备份已结束，后续阶段不会继续执行"})
			return
		}
	} else {
		s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 未配置照片库目录，跳过该阶段"})
	}

	// 备份 Database Dumps 目录
	if backupsDir != "" {
		plannedStages = append(plannedStages, "数据库备份")
		hasSyncTarget = true
		if jobCtx.Err() != nil {
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 任务已停止，数据库备份尚未开始"})
			return
		}
		dest := remote
		if !cfg.Encrypt.Enabled {
			dest = remote + "/backups"
		}
		s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 开始备份数据库备份目录: " + backupsDir})
		s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 目标位置: " + dest})
		// 预创建远端目录，避免某些 WebDAV 服务器返回 404
		if err := rclone.Mkdir(dest, confPath); err != nil {
			log.Printf("[backup] mkdir warning (non-fatal): %v", err)
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 预创建远端目录失败（非致命，继续尝试备份）: " + err.Error()})
		}
		logCh, errCh, err := s.Runner.Run(backupMode, backupsDir, dest, nil, confPath)
		if err != nil {
			log.Printf("[backup] failed to start backups backup: %v", err)
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 无法启动数据库备份目录同步: " + err.Error()})
			s.sendBackupNotify(cfg, notify.BackupNotification{
				Success:    false,
				Trigger:    trigger,
				Mode:       modeLabel,
				Stage:      "数据库备份",
				RemotePath: dest,
				Detail:     summarizeProgress("数据库备份") + "；启动失败：" + err.Error(),
			})
			return
		}
		s.Hub.BroadcastFromChannel(logCh) // 阻塞直到完成
		if runErr := <-errCh; runErr != nil {
			if errors.Is(runErr, rclone.ErrCancelled) {
				log.Printf("[backup] backups backup cancelled by user")
				s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 数据库备份已手动停止"})
				s.sendBackupNotify(cfg, notify.BackupNotification{
					Success:    false,
					Trigger:    trigger,
					Mode:       modeLabel,
					Stage:      "数据库备份",
					RemotePath: dest,
					Detail:     summarizeProgress("数据库备份") + "；任务已被手动停止",
				})
				return
			}
			log.Printf("[backup] backups backup failed: %v", runErr)
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 数据库备份失败: " + runErr.Error()})
			s.sendBackupNotify(cfg, notify.BackupNotification{
				Success:    false,
				Trigger:    trigger,
				Mode:       modeLabel,
				Stage:      "数据库备份",
				RemotePath: dest,
				Detail:     summarizeProgress("数据库备份") + "；" + runErr.Error(),
			})
			return
		}
		completedStages = append(completedStages, "数据库备份")
		s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 数据库备份目录同步阶段已结束"})
		if jobCtx.Err() != nil {
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 任务已停止，数据库备份已结束，后续阶段不会继续执行"})
			return
		}
	} else {
		s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 未配置数据库备份目录，跳过该阶段"})
	}

	if !hasSyncTarget {
		s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] 未配置任何可备份目录，请先在设置中填写照片库或数据库备份目录"})
		s.sendBackupNotify(cfg, notify.BackupNotification{
			Success:    false,
			Trigger:    trigger,
			Mode:       modeLabel,
			Stage:      "准备阶段",
			RemotePath: remote,
			Detail:     summarizeProgress("准备阶段") + "；未配置任何可备份目录，请先填写照片库或数据库备份目录",
		})
		return
	}

	s.Hub.Broadcast(rclone.LogLine{Stream: "stdout", Text: "[immichto115] 所有备份阶段执行完毕"})
	s.sendBackupNotify(cfg, notify.BackupNotification{
		Success:    true,
		Trigger:    trigger,
		Mode:       modeLabel,
		Stage:      "全部备份",
		RemotePath: remote,
		Detail:     summarizeProgress("") + "；照片库与数据库备份阶段都已执行完成",
	})

}

func (s *Server) beginBackupJob(trigger string) (context.Context, bool) {
	s.backupMu.Lock()
	defer s.backupMu.Unlock()

	if s.backupActive {
		return nil, false
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.backupCancel = cancel
	s.backupActive = true
	s.backupTrigger = fallbackBackupTrigger(trigger)
	return ctx, true
}

func fallbackBackupTrigger(trigger string) string {
	trigger = strings.TrimSpace(trigger)
	if trigger == "" {
		return "手动"
	}
	return trigger
}

func (s *Server) currentBackupTrigger() string {
	s.backupMu.RLock()
	defer s.backupMu.RUnlock()
	return fallbackBackupTrigger(s.backupTrigger)
}

func (s *Server) finishBackupJob() {
	s.backupMu.Lock()
	defer s.backupMu.Unlock()

	s.backupCancel = nil
	s.backupActive = false
	s.backupTrigger = ""
}

func (s *Server) stopBackupJob() bool {
	s.backupMu.Lock()
	defer s.backupMu.Unlock()

	if !s.backupActive || s.backupCancel == nil {
		return false
	}

	s.backupCancel()
	return true
}

func (s *Server) IsBackupActive() bool {
	s.backupMu.RLock()
	defer s.backupMu.RUnlock()
	return s.backupActive
}

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/api/health" {
			c.Next()
			return
		}

		cfg := s.Config.Get()

		// Setup 未完成时：仅放行 setup 必需接口，拒绝其余
		if !s.Config.IsSetupComplete() {
			if isSetupWhitelisted(c.Request) {
				c.Next()
			} else {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "初始化配置尚未完成，请先完成设置向导"})
			}
			return
		}

		// Setup 已完成但未开启认证 → 放行
		if !cfg.Server.AuthEnabled {
			c.Next()
			return
		}

		if strings.TrimSpace(cfg.Server.AuthUser) == "" || cfg.Server.AuthPasswordHash == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "访问保护已启用，但管理员账号配置不完整"})
			return
		}

		user, pass, ok := c.Request.BasicAuth()
		if ok && subtle.ConstantTimeCompare([]byte(user), []byte(cfg.Server.AuthUser)) == 1 && config.VerifyPassword(cfg.Server.AuthPasswordHash, pass) {
			c.Next()
			return
		}

		c.Header("WWW-Authenticate", `Basic realm="immichto115"`)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "需要管理员账号密码"})
	}
}

// isSetupWhitelisted 判断请求是否属于 setup 阶段的白名单。
// 仅允许初始化必需的接口通过，其余全部拒绝。
func isSetupWhitelisted(r *http.Request) bool {
	path := r.URL.Path
	// 健康检查始终放行
	if path == "/api/health" {
		return true
	}
	// Setup 阶段允许的接口
	setupPaths := map[string][]string{
		"/api/v1/system/status":       {"GET"},
		"/api/v1/config":              {"GET", "POST"},
		"/api/v1/webdav/test":         {"POST"},
		"/api/v1/webdav/ls":           {"POST"},
		"/api/v1/local/ls":            {"GET"},
		"/api/v1/remote/ls":           {"GET"},
		"/api/v1/open115/auth/start":  {"POST"},
		"/api/v1/open115/auth/status": {"GET"},
		"/api/v1/open115/auth/finish": {"POST"},
		"/api/v1/open115/test":        {"POST"},
		"/api/v1/open115/ls":          {"GET"},
	}
	methods, ok := setupPaths[path]
	if !ok {
		// 前端静态资源与 WebSocket 也需要放行
		if !strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/ws/") {
			return true
		}
		return false
	}
	for _, m := range methods {
		if r.Method == m {
			return true
		}
	}
	return false
}

// SetupRouter 注册所有 API 路由。
func (s *Server) SetupRouter() *gin.Engine {
	r := gin.Default()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(s.authMiddleware())

	// --- Health Check (Docker / 监控探针) ---
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// --- API v1 ---
	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		// 系统状态
		v1.GET("/system/status", s.handleSystemStatus)

		// 配置管理
		v1.GET("/config", s.handleGetConfig)
		v1.POST("/config", s.handleSaveConfig)

		// WebDAV 测试
		v1.POST("/webdav/test", s.handleWebDAVTest)
		v1.POST("/webdav/ls", s.handleWebDAVList)

		// 115 Open 授权 / 连接测试
		v1.POST("/open115/auth/start", s.handleOpen115AuthStart)
		v1.GET("/open115/auth/status", s.handleOpen115AuthStatus)
		v1.POST("/open115/auth/finish", s.handleOpen115AuthFinish)
		v1.POST("/open115/test", s.handleOpen115Test)
		v1.GET("/open115/ls", s.handleOpen115List)
		v1.POST("/open115/debug/stream-measure", s.handleOpen115DebugStreamMeasure)
		v1.POST("/open115/debug/stream-upload", s.handleOpen115DebugStreamUpload)

		// 备份控制
		v1.POST("/backup/start", s.handleBackupStart)
		v1.POST("/backup/stop", s.handleBackupStop)

		// 云端文件浏览 (Restore Explorer)
		v1.GET("/remote/ls", s.handleRemoteList)

		// 本地文件浏览 (向导路径选择)
		v1.GET("/local/ls", s.handleLocalList)

		// 通知测试
		v1.POST("/notify/test", s.handleNotifyTest)
	}

	// --- WebSocket（使用 auth 中间件保护）---
	r.GET("/ws/logs", HandleWebSocket(s.Hub))

	return r
}

// InitCron 根据配置初始化定时任务（在服务启动时调用）。
func (s *Server) InitCron() {
	cfg := s.Config.Get()
	if cfg.Cron.Enabled && cfg.Cron.Expression != "" {
		// 对于非标准5段cron，尝试补前导0秒
		expr := normalizeCronExpression(cfg.Cron.Expression)
		if err := s.Scheduler.Start(expr); err != nil {
			log.Printf("[cron] failed to start scheduler: %v", err)
		}
	}
}

// ===== Handler 实现 =====

func (s *Server) handleSystemStatus(c *gin.Context) {
	version, err := rclone.GetVersion()
	rcloneInstalled := err == nil

	status := "idle"
	if s.IsBackupActive() {
		status = "running"
	}

	nextRun := s.Scheduler.NextRun()
	cronRunning := s.Scheduler.IsRunning()

	provider := strings.TrimSpace(s.Config.Get().Provider)
	if provider == "" {
		provider = "webdav"
	}
	c.JSON(http.StatusOK, gin.H{
		"provider":         provider,
		"rclone_installed": rcloneInstalled,
		"rclone_version":   strings.TrimSpace(version),
		"backup_status":    status,
		"cron_enabled":     cronRunning,
		"next_run":         nextRun,
		"setup_complete":   s.Config.IsSetupComplete(),
	})
}

func (s *Server) handleGetConfig(c *gin.Context) {
	cfg := s.Config.Get()
	// 隐藏敏感信息
	maskSecret(&cfg.WebDAV.Password)
	maskSecret(&cfg.Open115.AccessToken)
	maskSecret(&cfg.Open115.RefreshToken)
	maskSecret(&cfg.Encrypt.Password)
	maskSecret(&cfg.Encrypt.Salt)
	maskSecret(&cfg.Open115Encrypt.Password)
	maskSecret(&cfg.Open115Encrypt.Salt)
	if cfg.Server.AuthPasswordHash != "" {
		cfg.Server.AuthPassword = maskedSecret
	}
	c.JSON(http.StatusOK, cfg)
}

func (s *Server) handleSaveConfig(c *gin.Context) {
	var newCfg config.AppConfig
	if err := c.ShouldBindJSON(&newCfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newCfg.Provider = strings.ToLower(strings.TrimSpace(newCfg.Provider))
	newCfg.WebDAV.URL = strings.TrimSpace(newCfg.WebDAV.URL)
	newCfg.WebDAV.User = strings.TrimSpace(newCfg.WebDAV.User)
	newCfg.WebDAV.Vendor = strings.TrimSpace(newCfg.WebDAV.Vendor)
	newCfg.Open115.ClientID = strings.TrimSpace(newCfg.Open115.ClientID)
	newCfg.Open115.AccessToken = strings.TrimSpace(newCfg.Open115.AccessToken)
	newCfg.Open115.RefreshToken = strings.TrimSpace(newCfg.Open115.RefreshToken)
	newCfg.Open115.RootID = strings.TrimSpace(newCfg.Open115.RootID)
	newCfg.Backup.LibraryDir = strings.TrimSpace(newCfg.Backup.LibraryDir)
	newCfg.Backup.BackupsDir = strings.TrimSpace(newCfg.Backup.BackupsDir)
	newCfg.Backup.RemoteDir = normalizeRemoteDir(newCfg.Backup.RemoteDir)
	newCfg.Backup.Mode = strings.ToLower(strings.TrimSpace(newCfg.Backup.Mode))
	newCfg.Backup.ManifestPath = strings.TrimSpace(newCfg.Backup.ManifestPath)
	newCfg.Open115Encrypt.Mode = strings.ToLower(strings.TrimSpace(newCfg.Open115Encrypt.Mode))
	newCfg.Open115Encrypt.FilenameMode = strings.ToLower(strings.TrimSpace(newCfg.Open115Encrypt.FilenameMode))
	newCfg.Open115Encrypt.Algorithm = strings.TrimSpace(newCfg.Open115Encrypt.Algorithm)
	newCfg.Open115Encrypt.TempDir = strings.TrimSpace(newCfg.Open115Encrypt.TempDir)
	newCfg.Cron.Expression = normalizeCronExpression(newCfg.Cron.Expression)
	newCfg.Notify.BarkURL = strings.TrimSpace(newCfg.Notify.BarkURL)

	// 如果前端传了 "********" 则保留旧密钥；显式清空字符串则视为用户要清除该值。
	oldCfg := s.Config.Get()
	restoreSecret(&newCfg.WebDAV.Password, oldCfg.WebDAV.Password)
	restoreSecret(&newCfg.Open115.AccessToken, oldCfg.Open115.AccessToken)
	restoreSecret(&newCfg.Open115.RefreshToken, oldCfg.Open115.RefreshToken)
	restoreSecret(&newCfg.Encrypt.Password, oldCfg.Encrypt.Password)
	restoreSecret(&newCfg.Encrypt.Salt, oldCfg.Encrypt.Salt)
	restoreSecret(&newCfg.Open115Encrypt.Password, oldCfg.Open115Encrypt.Password)
	restoreSecret(&newCfg.Open115Encrypt.Salt, oldCfg.Open115Encrypt.Salt)

	newCfg.Server.AuthUser = strings.TrimSpace(newCfg.Server.AuthUser)
	if newCfg.Server.AuthPassword == maskedSecret || newCfg.Server.AuthPassword == "" {
		newCfg.Server.AuthPasswordHash = oldCfg.Server.AuthPasswordHash
	} else {
		hash, err := config.HashPassword(newCfg.Server.AuthPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		newCfg.Server.AuthPasswordHash = hash
	}
	newCfg.Server.AuthPassword = ""

	if newCfg.Server.AuthEnabled {
		if newCfg.Server.AuthUser == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "启用访问保护时必须填写管理员用户名"})
			return
		}
		if newCfg.Server.AuthPasswordHash == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "启用访问保护时必须填写管理员密码"})
			return
		}
	}

	if err := s.Config.Update(newCfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 配置更新后重置 115 Open 客户端缓存，确保后续使用新 token
	s.Open115.ResetClient()

	// 更新定时任务
	if newCfg.Cron.Enabled && newCfg.Cron.Expression != "" {
		expr := normalizeCronExpression(newCfg.Cron.Expression)
		if err := s.Scheduler.Start(expr); err != nil {
			c.JSON(http.StatusOK, gin.H{"message": "配置已保存，但定时任务启动失败：" + err.Error()})
			return
		}
	} else {
		s.Scheduler.Stop()
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置已保存"})
}

// WebDAVTestRequest 测试 WebDAV 连接的请求体。
type WebDAVTestRequest struct {
	URL      string `json:"url" binding:"required"`
	User     string `json:"user" binding:"required"`
	Password string `json:"password" binding:"required"`
	Vendor   string `json:"vendor"`
}

type WebDAVListRequest struct {
	URL      string `json:"url" binding:"required"`
	User     string `json:"user" binding:"required"`
	Password string `json:"password" binding:"required"`
	Vendor   string `json:"vendor"`
	Path     string `json:"path"`
}

func (s *Server) resolveWebDAVPassword(password string) string {
	if password == maskedSecret {
		return s.Config.Get().WebDAV.Password
	}
	return password
}

func (s *Server) handleWebDAVTest(c *gin.Context) {
	var req WebDAVTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.URL = strings.TrimSpace(req.URL)
	req.User = strings.TrimSpace(req.User)
	req.Vendor = strings.TrimSpace(req.Vendor)

	// 如果密码是遮蔽的，使用已保存的密码
	password := s.resolveWebDAVPassword(req.Password)

	// 先对密码做 obscure 处理，避免命令注入风险
	obscured, obscErr := config.ObscurePassword(password)
	if obscErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "处理 WebDAV 密码失败：" + obscErr.Error(),
		})
		return
	}

	// 带超时的 context 防止 rclone 挂起
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	vendor := req.Vendor
	if vendor == "" {
		vendor = "other"
	}

	cmd := exec.CommandContext(ctx, "rclone", "lsd", ":webdav:", "--webdav-url", req.URL,
		"--webdav-user", req.User, "--webdav-pass", obscured,
		"--webdav-vendor", vendor,
		"--max-depth", "1", "--contimeout", "10s")

	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail == "" {
			detail = err.Error()
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "WebDAV 连接失败：" + detail,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "WebDAV 连接成功",
	})
}

func (s *Server) handleWebDAVList(c *gin.Context) {
	var req WebDAVListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.URL = strings.TrimSpace(req.URL)
	req.User = strings.TrimSpace(req.User)
	req.Path = strings.TrimSpace(req.Path)
	req.Vendor = strings.TrimSpace(req.Vendor)

	password := s.resolveWebDAVPassword(req.Password)
	vendor := req.Vendor
	if vendor == "" {
		vendor = strings.TrimSpace(s.Config.Get().WebDAV.Vendor)
	}
	if vendor == "" {
		vendor = "other"
	}

	tmpCfg := config.AppConfig{
		WebDAV: config.WebDAVConfig{
			URL:      req.URL,
			User:     req.User,
			Password: password,
			Vendor:   vendor,
		},
	}

	confPath, err := config.GenerateRcloneConf(tmpCfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 rclone 配置失败：" + err.Error()})
		return
	}
	defer config.CleanupRcloneConf(confPath)

	cleanPath := path.Clean("/" + req.Path)
	if cleanPath == "." {
		cleanPath = "/"
	}
	remotePath := "webdav115:" + cleanPath

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rclone", "lsjson", remotePath, "--config", confPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail == "" {
			detail = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取 WebDAV 目录失败：" + detail})
		return
	}

	c.Data(http.StatusOK, "application/json", out)
}

func (s *Server) handleBackupStart(c *gin.Context) {
	// 使用 beginBackupJob 原子地检查并占位，避免 TOCTOU 竞态
	jobCtx, ok := s.beginBackupJob("手动")
	if !ok {
		c.JSON(http.StatusConflict, gin.H{"error": "已有备份任务正在运行"})
		return
	}

	go func() {
		defer s.finishBackupJob()
		s.runBackupBody(jobCtx)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "备份已开始，正在检查配置并准备同步任务"})
}

type open115AuthStartRequest struct {
	ClientID string `json:"client_id" binding:"required"`
}

type open115AuthFinishRequest struct {
	UID string `json:"uid" binding:"required"`
}

func (s *Server) storeAuthSession(session *open115.AuthSession) {
	if s == nil || session == nil || strings.TrimSpace(session.UID) == "" {
		return
	}
	s.authSessionMu.Lock()
	defer s.authSessionMu.Unlock()
	s.authSessions[session.UID] = session
}

func (s *Server) loadAuthSession(uid string) (*open115.AuthSession, bool) {
	s.authSessionMu.RLock()
	defer s.authSessionMu.RUnlock()
	session, ok := s.authSessions[strings.TrimSpace(uid)]
	return session, ok
}

func (s *Server) deleteAuthSession(uid string) {
	s.authSessionMu.Lock()
	defer s.authSessionMu.Unlock()
	delete(s.authSessions, strings.TrimSpace(uid))
}

func (s *Server) cleanupExpiredAuthSessions() {
	s.authSessionMu.Lock()
	defer s.authSessionMu.Unlock()
	deadline := time.Now().Add(-10 * time.Minute)
	for uid, session := range s.authSessions {
		if session == nil || session.CreatedAt.Before(deadline) {
			delete(s.authSessions, uid)
		}
	}
}

// StartAuthCleanup 启动后台 goroutine 定期清理过期的 auth session。
// 当传入的 ctx 被取消时 goroutine 会退出，支持优雅关闭。
func (s *Server) StartAuthCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.cleanupExpiredAuthSessions()
			}
		}
	}()
}

func (s *Server) handleOpen115AuthStart(c *gin.Context) {
	var req open115AuthStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.cleanupExpiredAuthSessions()
	session, err := s.Open115.StartAuth(c.Request.Context(), req.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated := s.Config.Get()
	updated.Open115.ClientID = strings.TrimSpace(req.ClientID)
	if err := s.Config.Update(updated); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.storeAuthSession(session)
	c.JSON(http.StatusOK, gin.H{
		"uid":        session.UID,
		"time":       session.Time,
		"sign":       session.Sign,
		"qrcode":     session.QRCode,
		"created_at": session.CreatedAt,
	})
}

func (s *Server) handleOpen115AuthStatus(c *gin.Context) {
	uid := strings.TrimSpace(c.Query("uid"))
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "uid 不能为空"})
		return
	}
	s.cleanupExpiredAuthSessions()
	session, ok := s.loadAuthSession(uid)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "auth session 不存在或已过期"})
		return
	}
	status, err := s.Open115.CheckAuthStatus(c.Request.Context(), session)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (s *Server) handleOpen115AuthFinish(c *gin.Context) {
	var req open115AuthFinishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	session, ok := s.loadAuthSession(req.UID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "auth session 不存在或已过期"})
		return
	}
	state, err := s.Open115.FinishAuth(c.Request.Context(), session)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	s.deleteAuthSession(req.UID)
	c.JSON(http.StatusOK, gin.H{
		"message": "authorized",
		"state":   state,
	})
}

func (s *Server) handleOpen115Test(c *gin.Context) {
	if err := s.Open115.TestConnection(c.Request.Context()); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "115 Open 连接成功"})
}

func (s *Server) handleOpen115List(c *gin.Context) {
	remotePath := strings.TrimSpace(c.Query("path"))
	if remotePath == "" {
		remotePath = "/"
	}
	s.listOpen115Entries(c, remotePath)
}

// listOpen115Entries 是 handleOpen115List 和 handleRemoteList(open115分支) 的共用逻辑。
func (s *Server) listOpen115Entries(c *gin.Context, remotePath string) {
	backend := backup.NewOpen115Backend(s.Open115)
	items, err := backend.ListRemote(c.Request.Context(), remotePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	entries := make([]gin.H, 0, len(items))
	for _, item := range items {
		entries = append(entries, gin.H{
			"Name":    item.Name,
			"Path":    item.Path,
			"IsDir":   item.IsDir,
			"Size":    item.Size,
			"ModTime": time.Unix(item.ModTime, 0).Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, entries)
}

type open115DebugStreamMeasureRequest struct {
	LocalPath string `json:"local_path" binding:"required"`
}

type open115DebugStreamUploadRequest struct {
	LocalPath  string `json:"local_path" binding:"required"`
	RemotePath string `json:"remote_path" binding:"required"`
}

func (s *Server) handleOpen115DebugStreamMeasure(c *gin.Context) {
	var req open115DebugStreamMeasureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cfg := s.Config.Get()
	encCfg := open115crypt.FromAppConfig(cfg)
	if !encCfg.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "open115_encrypt 未启用"})
		return
	}
	if strings.TrimSpace(encCfg.Mode) != "stream" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前 open115_encrypt.mode 不是 stream，不能使用流式测量 debug 接口"})
		return
	}
	info, err := open115crypt.DebugMeasure(strings.TrimSpace(req.LocalPath), encCfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (s *Server) handleOpen115DebugStreamUpload(c *gin.Context) {
	var req open115DebugStreamUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cfg := s.Config.Get()
	encCfg := open115crypt.FromAppConfig(cfg)
	if !encCfg.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "open115_encrypt 未启用"})
		return
	}
	if strings.TrimSpace(encCfg.Mode) != "stream" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前 open115_encrypt.mode 不是 stream，不能使用流式上传 debug 接口"})
		return
	}
	uploader := open115.NewUploader(s.Open115)
	result, err := uploader.DebugStreamUpload(c.Request.Context(), strings.TrimSpace(req.LocalPath), strings.TrimSpace(req.RemotePath), encCfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "result": result})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleBackupStop(c *gin.Context) {
	jobStopped := s.stopBackupJob()
	runnerErr := s.Runner.Stop()
	if runnerErr != nil && !jobStopped {
		errMsg := runnerErr.Error()
		if errMsg == "no rclone process is running" {
			errMsg = "当前没有正在运行的备份任务"
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已发送停止指令，当前任务会在安全收尾后退出"})
}

// RemoteListRequest 云端文件浏览请求。
type RemoteListRequest struct {
	Path string `form:"path"`
}

func (s *Server) handleRemoteList(c *gin.Context) {
	var req RemoteListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg := s.Config.Get()
	provider := strings.TrimSpace(cfg.Provider)
	if provider == "" {
		provider = "webdav"
	}
	if provider == "open115" {
		remotePath := strings.TrimSpace(req.Path)
		if remotePath == "" {
			remotePath = "/"
		}
		s.listOpen115Entries(c, remotePath)
		return
	}

	confPath, err := config.GenerateRcloneConf(cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 rclone 配置失败：" + err.Error()})
		return
	}
	defer config.CleanupRcloneConf(confPath)

	remotePath := config.BuildRemotePath(cfg, req.Path)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rclone", "lsjson", remotePath, "--config", confPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail == "" {
			detail = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取远端目录失败：" + detail})
		return
	}

	c.Data(http.StatusOK, "application/json", out)
}

// LocalListRequest 本地文件浏览请求。
type LocalListRequest struct {
	Path string `form:"path"`
}

func (s *Server) handleLocalList(c *gin.Context) {
	var req LocalListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	localPath := req.Path
	if localPath == "" {
		if runtime.GOOS == "windows" {
			localPath = "C:\\"
		} else {
			localPath = "/"
		}
	}

	// 安全校验：在路径规范化之前先检查原始输入
	rawPath := strings.ReplaceAll(localPath, "\\", "/")
	for _, segment := range strings.Split(rawPath, "/") {
		if segment == ".." {
			c.JSON(http.StatusBadRequest, gin.H{"error": "路径中不允许包含父级路径段 '..'"})
			return
		}
	}

	absPath, err := filepath.Abs(localPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效路径: " + err.Error()})
		return
	}
	localPath = absPath

	entries, err := os.ReadDir(localPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取本地目录失败：" + err.Error()})
		return
	}

	type localEntry struct {
		Path    string `json:"Path"`
		Name    string `json:"Name"`
		IsDir   bool   `json:"IsDir"`
		Size    int64  `json:"Size,omitempty"`
		ModTime string `json:"ModTime,omitempty"`
	}

	result := make([]localEntry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		fullPath := filepath.Join(localPath, entry.Name())
		item := localEntry{
			Path:    fullPath,
			Name:    entry.Name(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime().Format(time.RFC3339),
		}
		if !entry.IsDir() {
			item.Size = info.Size()
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, result)
}

// sendBackupNotify 在备份完成时发送 Bark 推送通知。
func (s *Server) sendBackupNotify(cfg config.AppConfig, info notify.BackupNotification) {
	if !cfg.Notify.Enabled || cfg.Notify.BarkURL == "" {
		return
	}
	go notify.NotifyBackupResult(cfg.Notify.BarkURL, info)
}

// handleNotifyTest 测试 Bark 推送通知。
func (s *Server) handleNotifyTest(c *gin.Context) {
	cfg := s.Config.Get()
	if !cfg.Notify.Enabled || cfg.Notify.BarkURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "通知未启用或 Bark 地址为空，请先保存通知配置"})
		return
	}
	err := notify.SendBark(cfg.Notify.BarkURL, "🔔 测试通知", "应用：ImmichTo115\n结果：通知服务连接正常\n说明：后续会推送备份成功、失败、手动停止等状态")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "推送失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "测试通知已发送"})
}
