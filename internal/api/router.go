package api

import (
	"context"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/aYenx/immichto115/internal/config"
	appCron "github.com/aYenx/immichto115/internal/cron"
	"github.com/aYenx/immichto115/internal/rclone"
)

// Server 持有所有 API 依赖。
type Server struct {
	Runner    *rclone.Runner
	Hub       *Hub
	Config    *config.Manager
	Scheduler *appCron.Scheduler
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
	if s.Runner.IsRunning() {
		log.Println("[backup] skipped: already running")
		return
	}

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
	}

	// 备份 Database Dumps 目录
	if cfg.Backup.BackupsDir != "" {
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
	}

	config.CleanupRcloneConf(confPath)
}

// SetupRouter 注册所有 API 路由。
func (s *Server) SetupRouter() *gin.Engine {
	r := gin.Default()

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

		// 备份控制
		v1.POST("/backup/start", s.handleBackupStart)
		v1.POST("/backup/stop", s.handleBackupStop)

		// 云端文件浏览 (Restore Explorer)
		v1.GET("/remote/ls", s.handleRemoteList)
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
	if s.Runner.IsRunning() {
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
		cfg.WebDAV.Password = "********"
	}
	if cfg.Encrypt.Password != "" {
		cfg.Encrypt.Password = "********"
	}
	if cfg.Encrypt.Salt != "" {
		cfg.Encrypt.Salt = "********"
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
	if newCfg.WebDAV.Password == "********" || newCfg.WebDAV.Password == "" {
		newCfg.WebDAV.Password = oldCfg.WebDAV.Password
	}
	if newCfg.Encrypt.Password == "********" || newCfg.Encrypt.Password == "" {
		newCfg.Encrypt.Password = oldCfg.Encrypt.Password
	}
	if newCfg.Encrypt.Salt == "********" || newCfg.Encrypt.Salt == "" {
		newCfg.Encrypt.Salt = oldCfg.Encrypt.Salt
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

func (s *Server) handleWebDAVTest(c *gin.Context) {
	var req WebDAVTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果密码是遮蔽的，使用已保存的密码
	password := req.Password
	if password == "********" {
		password = s.Config.Get().WebDAV.Password
	}

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


func (s *Server) handleBackupStart(c *gin.Context) {
	if s.Runner.IsRunning() {
		c.JSON(http.StatusConflict, gin.H{"error": "backup is already running"})
		return
	}

	go s.triggerBackup()

	c.JSON(http.StatusOK, gin.H{"message": "backup started"})
}

func (s *Server) handleBackupStop(c *gin.Context) {
	if err := s.Runner.Stop(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
