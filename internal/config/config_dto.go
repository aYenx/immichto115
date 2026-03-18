package config

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// Configuration defaults
// ---------------------------------------------------------------------------

const (
	DefaultPort             = 8096
	DefaultProvider         = "webdav"
	DefaultRemoteDir        = "/immich-backup"
	DefaultPhotoRemoteDir   = "/摄影"
	DefaultPhotoDateFormat  = "2006/01/02"
	DefaultBackupMode       = "copy"
)

// ---------------------------------------------------------------------------
// Safe response types (sent to frontend — no secrets)
// ---------------------------------------------------------------------------

// SafeConfigResponse 返回给前端的配置视图，敏感字段用布尔值替代。
type SafeConfigResponse struct {
	Provider       string                   `json:"provider"`
	Server         SafeServerConfig         `json:"server"`
	WebDAV         SafeWebDAVConfig         `json:"webdav"`
	Open115        SafeOpen115Config        `json:"open115"`
	Open115Encrypt SafeOpen115EncryptConfig `json:"open115_encrypt"`
	Backup         BackupConfig             `json:"backup"`
	Encrypt        SafeEncryptConfig        `json:"encrypt"`
	Cron           CronConfig               `json:"cron"`
	Notify         NotifyConfig             `json:"notify"`
	PhotoUpload    PhotoUploadConfig        `json:"photo_upload"`
	UpdatedAt      int64                    `json:"updated_at"`
}

type SafeServerConfig struct {
	Port           int    `json:"port"`
	AuthEnabled    bool   `json:"auth_enabled"`
	AuthUser       string `json:"auth_user"`
	HasAuthPassword bool  `json:"has_auth_password"`
}

type SafeWebDAVConfig struct {
	URL         string `json:"url"`
	User        string `json:"user"`
	HasPassword bool   `json:"has_password"`
	Vendor      string `json:"vendor"`
}

type SafeOpen115Config struct {
	Enabled         bool   `json:"enabled"`
	ClientID        string `json:"client_id"`
	HasAccessToken  bool   `json:"has_access_token"`
	HasRefreshToken bool   `json:"has_refresh_token"`
	RootID          string `json:"root_id"`
	TokenExpiresAt  int64  `json:"token_expires_at"`
	UserID          string `json:"user_id"`
}

type SafeOpen115EncryptConfig struct {
	Enabled        bool   `json:"enabled"`
	HasPassword    bool   `json:"has_password"`
	HasSalt        bool   `json:"has_salt"`
	Mode           string `json:"mode"`
	FilenameMode   string `json:"filename_mode"`
	Algorithm      string `json:"algorithm"`
	TempDir        string `json:"temp_dir"`
	MinFreeSpaceMB int64  `json:"min_free_space_mb"`
}

type SafeEncryptConfig struct {
	Enabled     bool `json:"enabled"`
	HasPassword bool `json:"has_password"`
	HasSalt     bool `json:"has_salt"`
}

// ToSafe 将 AppConfig 转换为前端安全视图。
func (c AppConfig) ToSafe() SafeConfigResponse {
	return SafeConfigResponse{
		Provider: c.Provider,
		Server: SafeServerConfig{
			Port:            c.Server.Port,
			AuthEnabled:     c.Server.AuthEnabled,
			AuthUser:        c.Server.AuthUser,
			HasAuthPassword: c.Server.AuthPasswordHash != "",
		},
		WebDAV: SafeWebDAVConfig{
			URL:         c.WebDAV.URL,
			User:        c.WebDAV.User,
			HasPassword: c.WebDAV.Password != "",
			Vendor:      c.WebDAV.Vendor,
		},
		Open115: SafeOpen115Config{
			Enabled:         c.Open115.Enabled,
			ClientID:        c.Open115.ClientID,
			HasAccessToken:  c.Open115.AccessToken != "",
			HasRefreshToken: c.Open115.RefreshToken != "",
			RootID:          c.Open115.RootID,
			TokenExpiresAt:  c.Open115.TokenExpiresAt,
			UserID:          c.Open115.UserID,
		},
		Open115Encrypt: SafeOpen115EncryptConfig{
			Enabled:        c.Open115Encrypt.Enabled,
			HasPassword:    c.Open115Encrypt.Password != "",
			HasSalt:        c.Open115Encrypt.Salt != "",
			Mode:           c.Open115Encrypt.Mode,
			FilenameMode:   c.Open115Encrypt.FilenameMode,
			Algorithm:      c.Open115Encrypt.Algorithm,
			TempDir:        c.Open115Encrypt.TempDir,
			MinFreeSpaceMB: c.Open115Encrypt.MinFreeSpaceMB,
		},
		Backup: c.Backup,
		Encrypt: SafeEncryptConfig{
			Enabled:     c.Encrypt.Enabled,
			HasPassword: c.Encrypt.Password != "",
			HasSalt:     c.Encrypt.Salt != "",
		},
		Cron:        c.Cron,
		Notify:      c.Notify,
		PhotoUpload: c.PhotoUpload,
		UpdatedAt:   c.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// Update request types (received from frontend — secrets are optional)
// ---------------------------------------------------------------------------

// ConfigUpdateRequest 前端保存配置的请求体。
// 敏感字段用 *string：nil=保留旧值，""=清空，非空=设置新值。
type ConfigUpdateRequest struct {
	Provider       string                         `json:"provider"`
	Server         ServerUpdateRequest            `json:"server"`
	WebDAV         WebDAVUpdateRequest            `json:"webdav"`
	Open115        Open115UpdateRequest           `json:"open115"`
	Open115Encrypt Open115EncryptUpdateRequest    `json:"open115_encrypt"`
	Backup         BackupConfig                   `json:"backup"`
	Encrypt        EncryptUpdateRequest           `json:"encrypt"`
	Cron           CronConfig                     `json:"cron"`
	Notify         NotifyConfig                   `json:"notify"`
	PhotoUpload    PhotoUploadConfig              `json:"photo_upload"`
	UpdatedAt      int64                          `json:"updated_at"`
}

type ServerUpdateRequest struct {
	Port        int     `json:"port"`
	AuthEnabled bool    `json:"auth_enabled"`
	AuthUser    string  `json:"auth_user"`
	Password    *string `json:"password"` // nil=keep, ""=clear, value=set
}

type WebDAVUpdateRequest struct {
	URL      string  `json:"url"`
	User     string  `json:"user"`
	Password *string `json:"password"`
	Vendor   string  `json:"vendor"`
}

type Open115UpdateRequest struct {
	Enabled        bool    `json:"enabled"`
	ClientID       string  `json:"client_id"`
	AccessToken    *string `json:"access_token"`
	RefreshToken   *string `json:"refresh_token"`
	RootID         string  `json:"root_id"`
	TokenExpiresAt int64   `json:"token_expires_at"`
	UserID         string  `json:"user_id"`
}

type Open115EncryptUpdateRequest struct {
	Enabled        bool    `json:"enabled"`
	Password       *string `json:"password"`
	Salt           *string `json:"salt"`
	Mode           string  `json:"mode"`
	FilenameMode   string  `json:"filename_mode"`
	Algorithm      string  `json:"algorithm"`
	TempDir        string  `json:"temp_dir"`
	MinFreeSpaceMB int64   `json:"min_free_space_mb"`
}

type EncryptUpdateRequest struct {
	Enabled  bool    `json:"enabled"`
	Password *string `json:"password"`
	Salt     *string `json:"salt"`
}

// ---------------------------------------------------------------------------
// Validation (centralizes trim + defaults + business rules)
// ---------------------------------------------------------------------------

// NormalizeRemoteDir 规范化远端目录路径。
func NormalizeRemoteDir(remoteDir string) string {
	cleaned := CleanRemotePath(remoteDir)
	if cleaned == "." || cleaned == "" {
		return "/"
	}
	return cleaned
}

// NormalizeCronExpression 规范化 cron 表达式（5段→6段）。
func NormalizeCronExpression(expr string) string {
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

// Validate 校验并规范化更新请求的所有字段。
func (r *ConfigUpdateRequest) Validate() error {
	r.Provider = strings.ToLower(strings.TrimSpace(r.Provider))
	if r.Provider == "" {
		r.Provider = DefaultProvider
	}
	if r.Provider != "webdav" && r.Provider != "open115" {
		return fmt.Errorf("不支持的 provider: %s", r.Provider)
	}

	// Server
	r.Server.AuthUser = strings.TrimSpace(r.Server.AuthUser)
	if r.Server.Port <= 0 {
		r.Server.Port = DefaultPort
	}
	if r.Server.AuthEnabled && r.Server.AuthUser == "" {
		return fmt.Errorf("启用访问保护时必须填写管理员用户名")
	}

	// WebDAV
	r.WebDAV.URL = strings.TrimSpace(r.WebDAV.URL)
	r.WebDAV.User = strings.TrimSpace(r.WebDAV.User)
	r.WebDAV.Vendor = strings.TrimSpace(r.WebDAV.Vendor)

	// Open115
	r.Open115.ClientID = strings.TrimSpace(r.Open115.ClientID)
	r.Open115.RootID = strings.TrimSpace(r.Open115.RootID)

	// Open115 Encrypt
	r.Open115Encrypt.Mode = strings.ToLower(strings.TrimSpace(r.Open115Encrypt.Mode))
	r.Open115Encrypt.FilenameMode = strings.ToLower(strings.TrimSpace(r.Open115Encrypt.FilenameMode))
	r.Open115Encrypt.Algorithm = strings.TrimSpace(r.Open115Encrypt.Algorithm)
	r.Open115Encrypt.TempDir = strings.TrimSpace(r.Open115Encrypt.TempDir)

	// Backup
	r.Backup.LibraryDir = strings.TrimSpace(r.Backup.LibraryDir)
	r.Backup.BackupsDir = strings.TrimSpace(r.Backup.BackupsDir)
	r.Backup.RemoteDir = NormalizeRemoteDir(r.Backup.RemoteDir)
	r.Backup.Mode = strings.ToLower(strings.TrimSpace(r.Backup.Mode))
	r.Backup.ManifestPath = strings.TrimSpace(r.Backup.ManifestPath)

	// Cron
	r.Cron.Expression = NormalizeCronExpression(r.Cron.Expression)

	// Notify
	r.Notify.BarkURL = strings.TrimSpace(r.Notify.BarkURL)

	// PhotoUpload
	r.PhotoUpload.WatchDir = strings.TrimSpace(r.PhotoUpload.WatchDir)
	r.PhotoUpload.RemoteDir = strings.TrimSpace(r.PhotoUpload.RemoteDir)
	r.PhotoUpload.Extensions = strings.TrimSpace(r.PhotoUpload.Extensions)
	r.PhotoUpload.DateFormat = strings.TrimSpace(r.PhotoUpload.DateFormat)
	if r.PhotoUpload.RemoteDir == "" {
		r.PhotoUpload.RemoteDir = DefaultPhotoRemoteDir
	}
	if r.PhotoUpload.DateFormat == "" {
		r.PhotoUpload.DateFormat = DefaultPhotoDateFormat
	}

	return nil
}

// ---------------------------------------------------------------------------
// Merge logic
// ---------------------------------------------------------------------------

// applyOptionalSecret 将 *string 语义应用到目标字段：
// nil → 保留旧值；"" → 清空；非空 → 设置新值。
func applyOptionalSecret(ptr *string, old string) string {
	if ptr == nil {
		return old
	}
	return *ptr
}

// ApplyUpdate 将 ConfigUpdateRequest 合并到当前配置并持久化。
// 如果 req.UpdatedAt 与当前不匹配则返回并发冲突错误。
func (m *Manager) ApplyUpdate(req ConfigUpdateRequest) (AppConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	old := *m.cfg

	// 并发编辑保护：前端传回的 updated_at 必须与当前值匹配
	if req.UpdatedAt != 0 && req.UpdatedAt != old.UpdatedAt {
		return AppConfig{}, fmt.Errorf("配置已被其他页面修改，请刷新后重试")
	}

	// 构建合并后的配置
	merged := old

	merged.Provider = req.Provider

	// Server
	merged.Server.Port = req.Server.Port
	merged.Server.AuthEnabled = req.Server.AuthEnabled
	merged.Server.AuthUser = req.Server.AuthUser
	if req.Server.Password != nil {
		pw := *req.Server.Password
		if pw == "" {
			merged.Server.AuthPasswordHash = ""
		} else {
			hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
			if err != nil {
				return AppConfig{}, fmt.Errorf("密码加密失败: %w", err)
			}
			merged.Server.AuthPasswordHash = string(hash)
		}
	}
	// 验证：启用 auth 但无密码
	if merged.Server.AuthEnabled && merged.Server.AuthPasswordHash == "" {
		return AppConfig{}, fmt.Errorf("启用访问保护时必须填写管理员密码")
	}

	// WebDAV
	merged.WebDAV.URL = req.WebDAV.URL
	merged.WebDAV.User = req.WebDAV.User
	merged.WebDAV.Password = applyOptionalSecret(req.WebDAV.Password, old.WebDAV.Password)
	merged.WebDAV.Vendor = req.WebDAV.Vendor

	// Open115
	merged.Open115.Enabled = req.Open115.Enabled
	merged.Open115.ClientID = req.Open115.ClientID
	merged.Open115.AccessToken = applyOptionalSecret(req.Open115.AccessToken, old.Open115.AccessToken)
	merged.Open115.RefreshToken = applyOptionalSecret(req.Open115.RefreshToken, old.Open115.RefreshToken)
	merged.Open115.RootID = req.Open115.RootID
	merged.Open115.TokenExpiresAt = req.Open115.TokenExpiresAt
	merged.Open115.UserID = req.Open115.UserID

	// Open115 Encrypt
	merged.Open115Encrypt.Enabled = req.Open115Encrypt.Enabled
	merged.Open115Encrypt.Password = applyOptionalSecret(req.Open115Encrypt.Password, old.Open115Encrypt.Password)
	merged.Open115Encrypt.Salt = applyOptionalSecret(req.Open115Encrypt.Salt, old.Open115Encrypt.Salt)
	merged.Open115Encrypt.Mode = req.Open115Encrypt.Mode
	merged.Open115Encrypt.FilenameMode = req.Open115Encrypt.FilenameMode
	merged.Open115Encrypt.Algorithm = req.Open115Encrypt.Algorithm
	merged.Open115Encrypt.TempDir = req.Open115Encrypt.TempDir
	merged.Open115Encrypt.MinFreeSpaceMB = req.Open115Encrypt.MinFreeSpaceMB

	// Backup
	merged.Backup = req.Backup

	// Encrypt
	merged.Encrypt.Enabled = req.Encrypt.Enabled
	merged.Encrypt.Password = applyOptionalSecret(req.Encrypt.Password, old.Encrypt.Password)
	merged.Encrypt.Salt = applyOptionalSecret(req.Encrypt.Salt, old.Encrypt.Salt)

	// Cron
	merged.Cron = req.Cron

	// Notify
	merged.Notify = req.Notify

	// PhotoUpload
	merged.PhotoUpload = req.PhotoUpload

	// 更新时间戳（单调递增，与 Manager.Update 共用 nextUpdatedAt）
	merged.UpdatedAt = nextUpdatedAt(old.UpdatedAt)

	// 持久化（内部调用，已持有锁，直接操作 viper）
	if err := m.persistLocked(merged); err != nil {
		return AppConfig{}, err
	}

	m.cfg = &merged
	return merged, nil
}
