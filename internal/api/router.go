package api

import (
	"context"
	"crypto/subtle"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/aYenx/immichto115/internal/config"
	appCron "github.com/aYenx/immichto115/internal/cron"
	"github.com/aYenx/immichto115/internal/rclone"
)

const maskedSecret = "********"

// Server 持有所有 API 依赖。
type Server struct {
	Runner    *rclone.Runner
	Hub       *Hub
	Config    *config.Manager
	Scheduler *appCron.Scheduler

	backupMu     sync.Mutex
	backupCancel context.CancelFunc
	backupActive bool
}

// NewServer 创建 API Server 实例。
func NewServer(cfgMgr *config.Manager) *Server {
	s := &Server{
		Runner: rclone.NewRunner(),
		Hub:    NewHub(),
		Config: cfgMgr,
	}

	// 定时任务：触发时执行备份
	s.Scheduler = appCron.NewScheduler(func() {
		s.triggerBackup()
	})

	return s
}

// triggerBackup 是定时任务和手动触发共用的备份逻辑。
func (s *Server) triggerBackup() {
	jobCtx, ok := s.beginBackupJob()
	if !ok {
		log.Println("[backup] skipped: already running")
		return
	}
	defer s.finishBackupJob()

	cfg := s.Config.Get()

	// 生成临时 rclone.conf
	confPath, err := config.GenerateRcloneConf(cfg)
	if err != nil {
		log.Printf("[backup] failed to generate rclone.conf: %v", err)
		s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] failed to generate rclone config: " + err.Error()})
		return
	}

	remote := config.GetRemoteName(cfg)

	// 备份 Library 目录
	if cfg.Backup.LibraryDir != "" {
		if jobCtx.Err() != nil {
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] backup stopped before library sync started"})
			config.CleanupRcloneConf(confPath)
			return
		}
		dest := remote
		if !cfg.Encrypt.Enabled {
			dest = remote + "/library"
		}
		logCh, err := s.Runner.RunSync(cfg.Backup.LibraryDir, dest, nil, confPath)
		if err != nil {
			log.Printf("[backup] failed to start library backup: %v", err)
			config.CleanupRcloneConf(confPath)
			return
		}
		s.Hub.BroadcastFromChannel(logCh) // 阻塞直到完成
		if jobCtx.Err() != nil {
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] backup stopped after library sync"})
			config.CleanupRcloneConf(confPath)
			return
		}
	}

	// 备份 Database Dumps 目录
	if cfg.Backup.BackupsDir != "" {
		if jobCtx.Err() != nil {
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] backup stopped before backups sync started"})
			config.CleanupRcloneConf(confPath)
			return
		}
		dest := remote
		if !cfg.Encrypt.Enabled {
			dest = remote + "/backups"
		}
		logCh, err := s.Runner.RunSync(cfg.Backup.BackupsDir, dest, nil, confPath)
		if err != nil {
			log.Printf("[backup] failed to start backups backup: %v", err)
			config.CleanupRcloneConf(confPath)
			return
		}
		s.Hub.BroadcastFromChannel(logCh) // 阻塞直到完成
		if jobCtx.Err() != nil {
			s.Hub.Broadcast(rclone.LogLine{Stream: "stderr", Text: "[immichto115] backup stopped after backups sync"})
			config.CleanupRcloneConf(confPath)
			return
		}
	}

	config.CleanupRcloneConf(confPath)
}

func (s *Server) beginBackupJob() (context.Context, bool) {
	s.backupMu.Lock()
	defer s.backupMu.Unlock()

	if s.backupActive {
		return nil, false
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.backupCancel = cancel
	s.backupActive = true
	return ctx, true
}

func (s *Server) finishBackupJob() {
	s.backupMu.Lock()
	defer s.backupMu.Unlock()

	s.backupCancel = nil
	s.backupActive = false
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
	s.backupMu.Lock()
	defer s.backupMu.Unlock()
	return s.backupActive
}

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/api/health" {
			c.Next()
			return
		}

		cfg := s.Config.Get()
		if !cfg.Server.AuthEnabled || !s.Config.IsSetupComplete() {
			c.Next()
			return
		}

		if strings.TrimSpace(cfg.Server.AuthUser) == "" || cfg.Server.AuthPasswordHash == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "authentication is enabled but not configured correctly"})
			return
		}

		user, pass, ok := c.Request.BasicAuth()
		if ok && subtle.ConstantTimeCompare([]byte(user), []byte(cfg.Server.AuthUser)) == 1 && config.VerifyPassword(cfg.Server.AuthPasswordHash, pass) {
			c.Next()
			return
		}

		c.Header("WWW-Authenticate", `Basic realm="immichto115"`)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
	}
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

		// 备份控制
		v1.POST("/backup/start", s.handleBackupStart)
		v1.POST("/backup/stop", s.handleBackupStop)

		// 云端文件浏览 (Restore Explorer)
		v1.GET("/remote/ls", s.handleRemoteList)

		// 本地文件浏览 (向导路径选择)
		v1.GET("/local/ls", s.handleLocalList)
	}

	// --- WebSocket ---
	r.GET("/ws/logs", HandleWebSocket(s.Hub))

	return r
}

// InitCron 根据配置初始化定时任务（在服务启动时调用）。
func (s *Server) InitCron() {
	cfg := s.Config.Get()
	if cfg.Cron.Enabled && cfg.Cron.Expression != "" {
		// 对于非标准5段cron，尝试补前导0秒
		expr := cfg.Cron.Expression
		parts := strings.Fields(expr)
		if len(parts) == 5 {
			expr = "0 " + expr // 补秒字段
		}
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

	c.JSON(http.StatusOK, gin.H{
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
	if cfg.WebDAV.Password != "" {
		cfg.WebDAV.Password = maskedSecret
	}
	if cfg.Encrypt.Password != "" {
		cfg.Encrypt.Password = maskedSecret
	}
	if cfg.Encrypt.Salt != "" {
		cfg.Encrypt.Salt = maskedSecret
	}
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

	// 如果前端传了 "********" 则保留旧密码
	oldCfg := s.Config.Get()
	if newCfg.WebDAV.Password == maskedSecret || newCfg.WebDAV.Password == "" {
		newCfg.WebDAV.Password = oldCfg.WebDAV.Password
	}
	if newCfg.Encrypt.Password == maskedSecret || newCfg.Encrypt.Password == "" {
		newCfg.Encrypt.Password = oldCfg.Encrypt.Password
	}
	if newCfg.Encrypt.Salt == maskedSecret || newCfg.Encrypt.Salt == "" {
		newCfg.Encrypt.Salt = oldCfg.Encrypt.Salt
	}

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
			c.JSON(http.StatusBadRequest, gin.H{"error": "auth_user is required when authentication is enabled"})
			return
		}
		if newCfg.Server.AuthPasswordHash == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "auth_password is required when authentication is enabled"})
			return
		}
	}

	if err := s.Config.Update(newCfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 更新定时任务
	if newCfg.Cron.Enabled && newCfg.Cron.Expression != "" {
		expr := newCfg.Cron.Expression
		parts := strings.Fields(expr)
		if len(parts) == 5 {
			expr = "0 " + expr
		}
		if err := s.Scheduler.Start(expr); err != nil {
			c.JSON(http.StatusOK, gin.H{"message": "config saved, but cron failed: " + err.Error()})
			return
		}
	} else {
		s.Scheduler.Stop()
	}

	c.JSON(http.StatusOK, gin.H{"message": "config saved"})
}

// WebDAVTestRequest 测试 WebDAV 连接的请求体。
type WebDAVTestRequest struct {
	URL      string `json:"url" binding:"required"`
	User     string `json:"user" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type WebDAVListRequest struct {
	URL      string `json:"url" binding:"required"`
	User     string `json:"user" binding:"required"`
	Password string `json:"password" binding:"required"`
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

	// 如果密码是遮蔽的，使用已保存的密码
	password := s.resolveWebDAVPassword(req.Password)

	// 先对密码做 obscure 处理，避免命令注入风险
	obscured, obscErr := config.ObscurePassword(password)
	if obscErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to obscure password: " + obscErr.Error(),
		})
		return
	}

	// 带超时的 context 防止 rclone 挂起
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rclone", "lsd", ":webdav:", "--webdav-url", req.URL,
		"--webdav-user", req.User, "--webdav-pass", obscured,
		"--max-depth", "1", "--contimeout", "10s")

	out, err := cmd.CombinedOutput()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "connection failed: " + string(out),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "WebDAV connection successful",
	})
}

func (s *Server) handleWebDAVList(c *gin.Context) {
	var req WebDAVListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	password := s.resolveWebDAVPassword(req.Password)
	vendor := s.Config.Get().WebDAV.Vendor
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate rclone config: " + err.Error()})
		return
	}
	defer config.CleanupRcloneConf(confPath)

	remotePath := "webdav115:"
	if req.Path != "" && req.Path != "/" {
		remotePath += req.Path
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rclone", "lsjson", remotePath, "--config", confPath)
	out, err := cmd.Output()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list webdav directory: " + err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/json", out)
}


func (s *Server) handleBackupStart(c *gin.Context) {
	if s.IsBackupActive() {
		c.JSON(http.StatusConflict, gin.H{"error": "backup is already running"})
		return
	}

	go s.triggerBackup()

	c.JSON(http.StatusOK, gin.H{"message": "backup started"})
}

func (s *Server) handleBackupStop(c *gin.Context) {
	jobStopped := s.stopBackupJob()
	runnerErr := s.Runner.Stop()
	if runnerErr != nil && !jobStopped {
		c.JSON(http.StatusBadRequest, gin.H{"error": runnerErr.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "backup stop signal sent"})
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

	confPath, err := config.GenerateRcloneConf(cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate rclone config: " + err.Error()})
		return
	}
	defer config.CleanupRcloneConf(confPath)

	remote := config.GetRemoteName(cfg)
	remotePath := remote + req.Path

	// 带超时的 context 防止 rclone 挂起
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rclone", "lsjson", remotePath, "--config", confPath)
	out, err := cmd.Output()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list remote: " + err.Error()})
		return
	}

	// lsjson 返回的就是 JSON 数组，直接透传
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rclone", "lsjson", localPath)
	out, err := cmd.Output()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list local directory: " + string(out) + " " + err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/json", out)
}
