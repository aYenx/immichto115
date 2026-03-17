package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

// AppConfig 应用全局配置结构。
type AppConfig struct {
	Provider       string               `mapstructure:"provider" json:"provider" yaml:"provider"`
	Server         ServerConfig         `mapstructure:"server"   json:"server"   yaml:"server"`
	WebDAV         WebDAVConfig         `mapstructure:"webdav"   json:"webdav"   yaml:"webdav"`
	Open115        Open115Config        `mapstructure:"open115"  json:"open115"  yaml:"open115"`
	Open115Encrypt Open115EncryptConfig `mapstructure:"open115_encrypt" json:"open115_encrypt" yaml:"open115_encrypt"`
	Backup         BackupConfig         `mapstructure:"backup"   json:"backup"   yaml:"backup"`
	Encrypt        EncryptConfig        `mapstructure:"encrypt"  json:"encrypt"  yaml:"encrypt"`
	Cron           CronConfig           `mapstructure:"cron"     json:"cron"     yaml:"cron"`
	Notify         NotifyConfig         `mapstructure:"notify"   json:"notify"   yaml:"notify"`
	PhotoUpload    PhotoUploadConfig    `mapstructure:"photo_upload" json:"photo_upload" yaml:"photo_upload"`
}

// ServerConfig 服务器配置。
type ServerConfig struct {
	Port             int    `mapstructure:"port" json:"port" yaml:"port"`
	AuthEnabled      bool   `mapstructure:"auth_enabled" json:"auth_enabled" yaml:"auth_enabled"`
	AuthUser         string `mapstructure:"auth_user" json:"auth_user" yaml:"auth_user"`
	AuthPasswordHash string `mapstructure:"auth_password_hash" json:"-" yaml:"auth_password_hash"`
	AuthPassword     string `mapstructure:"-" json:"auth_password,omitempty" yaml:"-"`
}

// WebDAVConfig WebDAV 连接配置。
type WebDAVConfig struct {
	URL      string `mapstructure:"url"      json:"url"      yaml:"url"`
	User     string `mapstructure:"user"     json:"user"     yaml:"user"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
	Vendor   string `mapstructure:"vendor"   json:"vendor"   yaml:"vendor"` // 如 "other"
}

// Open115Config 115 Open 接入配置。
type Open115Config struct {
	Enabled        bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	ClientID       string `mapstructure:"client_id" json:"client_id" yaml:"client_id"`
	AccessToken    string `mapstructure:"access_token" json:"access_token" yaml:"access_token"`
	RefreshToken   string `mapstructure:"refresh_token" json:"refresh_token" yaml:"refresh_token"`
	RootID         string `mapstructure:"root_id" json:"root_id" yaml:"root_id"`
	TokenExpiresAt int64  `mapstructure:"token_expires_at" json:"token_expires_at" yaml:"token_expires_at"`
	UserID         string `mapstructure:"user_id" json:"user_id" yaml:"user_id"`
}

// Open115EncryptConfig Open115 模式本地加密配置。
type Open115EncryptConfig struct {
	Enabled        bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Password       string `mapstructure:"password" json:"password" yaml:"password"`
	Salt           string `mapstructure:"salt" json:"salt" yaml:"salt"`
	Mode           string `mapstructure:"mode" json:"mode" yaml:"mode"`
	FilenameMode   string `mapstructure:"filename_mode" json:"filename_mode" yaml:"filename_mode"`
	Algorithm      string `mapstructure:"algorithm" json:"algorithm" yaml:"algorithm"`
	TempDir        string `mapstructure:"temp_dir" json:"temp_dir" yaml:"temp_dir"`
	MinFreeSpaceMB int64  `mapstructure:"min_free_space_mb" json:"min_free_space_mb" yaml:"min_free_space_mb"`
}

// BackupConfig 备份源和目标配置。
type BackupConfig struct {
	LibraryDir        string `mapstructure:"library_dir" json:"library_dir" yaml:"library_dir"`
	BackupsDir        string `mapstructure:"backups_dir" json:"backups_dir" yaml:"backups_dir"`
	RemoteDir         string `mapstructure:"remote_dir"  json:"remote_dir"  yaml:"remote_dir"`
	Mode              string `mapstructure:"mode"        json:"mode"        yaml:"mode"` // "copy" (增量) 或 "sync" (镜像)
	ManifestPath      string `mapstructure:"manifest_path" json:"manifest_path" yaml:"manifest_path"`
	AllowRemoteDelete bool   `mapstructure:"allow_remote_delete" json:"allow_remote_delete" yaml:"allow_remote_delete"`
}

// EncryptConfig Rclone Crypt 加密配置。
type EncryptConfig struct {
	Enabled  bool   `mapstructure:"enabled"  json:"enabled"  yaml:"enabled"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
	Salt     string `mapstructure:"salt"     json:"salt"     yaml:"salt"`
}

// CronConfig 定时任务配置。
type CronConfig struct {
	Enabled    bool   `mapstructure:"enabled"    json:"enabled"    yaml:"enabled"`
	Expression string `mapstructure:"expression" json:"expression" yaml:"expression"` // 如 "0 2 * * *"
}

// NotifyConfig 通知推送配置。
type NotifyConfig struct {
	Enabled bool   `mapstructure:"enabled"  json:"enabled"  yaml:"enabled"`
	BarkURL string `mapstructure:"bark_url" json:"bark_url" yaml:"bark_url"` // Bark 推送地址，如 https://api.day.app/YOUR_KEY
}

// PhotoUploadConfig 摄影文件自动上传配置。
type PhotoUploadConfig struct {
	Enabled           bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	WatchDir          string `mapstructure:"watch_dir" json:"watch_dir" yaml:"watch_dir"`                       // 本地监控目录
	RemoteDir         string `mapstructure:"remote_dir" json:"remote_dir" yaml:"remote_dir"`                    // 115上的目标根目录
	Extensions        string `mapstructure:"extensions" json:"extensions" yaml:"extensions"`                    // 逗号分隔的扩展名
	DateFormat        string `mapstructure:"date_format" json:"date_format" yaml:"date_format"`                 // 目录分类格式，默认 "2006/01/02"
	DeleteAfterUpload bool   `mapstructure:"delete_after_upload" json:"delete_after_upload" yaml:"delete_after_upload"`
}

// Manager 配置管理器，线程安全。
type Manager struct {
	mu       sync.RWMutex
	cfg      *AppConfig
	filePath string
	v        *viper.Viper
}

// NewManager 创建配置管理器并加载或初始化配置文件。
func NewManager(configPath string) (*Manager, error) {
	v := viper.New()

	m := &Manager{
		filePath: configPath,
		v:        v,
	}

	// 确保配置目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config dir: %w", err)
	}

	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 设置默认值
	v.SetDefault("provider", "webdav")
	v.SetDefault("server.port", 8096)
	v.SetDefault("server.auth_enabled", false)
	v.SetDefault("webdav.vendor", "other")
	v.SetDefault("open115.enabled", false)
	v.SetDefault("open115.root_id", "0")
	v.SetDefault("open115.token_expires_at", 0)
	v.SetDefault("open115_encrypt.enabled", false)
	v.SetDefault("open115_encrypt.mode", "temp")
	v.SetDefault("open115_encrypt.filename_mode", "plain")
	v.SetDefault("open115_encrypt.algorithm", "aes256gcm-v1")
	v.SetDefault("open115_encrypt.temp_dir", "")
	v.SetDefault("open115_encrypt.min_free_space_mb", 1024)
	v.SetDefault("backup.remote_dir", "/immich-backup")
	v.SetDefault("backup.mode", "copy")
	v.SetDefault("backup.manifest_path", "")
	v.SetDefault("backup.allow_remote_delete", false)
	v.SetDefault("cron.expression", "0 2 * * *")
	v.SetDefault("cron.enabled", false)
	v.SetDefault("encrypt.enabled", false)
	v.SetDefault("notify.enabled", false)
	v.SetDefault("photo_upload.enabled", false)
	v.SetDefault("photo_upload.extensions", "cr2,cr3,nef,arw,dng,raf,rw2,orf,pef,srw,jpg,jpeg")
	v.SetDefault("photo_upload.date_format", "2006/01/02")
	v.SetDefault("photo_upload.delete_after_upload", true)
	v.SetDefault("photo_upload.remote_dir", "/摄影")

	// 尝试读取已有配置
	if err := v.ReadInConfig(); err != nil {
		// SetConfigFile 时文件不存在返回 os.PathError，而非 viper.ConfigFileNotFoundError
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) || os.IsNotExist(err) {
			// 文件不存在，写入默认配置
			if writeErr := v.SafeWriteConfig(); writeErr != nil {
				if writeErr2 := v.WriteConfig(); writeErr2 != nil {
					return nil, fmt.Errorf("failed to write default config: %w", writeErr2)
				}
			}
		} else {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	cfg := &AppConfig{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	m.cfg = cfg

	return m, nil
}

// Get 返回当前配置的副本。
func (m *Manager) Get() AppConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.cfg
}

func (m *Manager) FilePath() string {
	if m == nil {
		return ""
	}
	return m.filePath
}

// Update 更新配置并持久化到文件。
func (m *Manager) Update(cfg AppConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	v := m.v

	v.Set("provider", cfg.Provider)
	v.Set("server.port", cfg.Server.Port)
	v.Set("server.auth_enabled", cfg.Server.AuthEnabled)
	v.Set("server.auth_user", cfg.Server.AuthUser)
	v.Set("server.auth_password_hash", cfg.Server.AuthPasswordHash)

	v.Set("webdav.url", cfg.WebDAV.URL)
	v.Set("webdav.user", cfg.WebDAV.User)
	v.Set("webdav.password", cfg.WebDAV.Password)
	v.Set("webdav.vendor", cfg.WebDAV.Vendor)

	v.Set("open115.enabled", cfg.Open115.Enabled)
	v.Set("open115.client_id", cfg.Open115.ClientID)
	v.Set("open115.access_token", cfg.Open115.AccessToken)
	v.Set("open115.refresh_token", cfg.Open115.RefreshToken)
	v.Set("open115.root_id", cfg.Open115.RootID)
	v.Set("open115.token_expires_at", cfg.Open115.TokenExpiresAt)
	v.Set("open115.user_id", cfg.Open115.UserID)

	v.Set("open115_encrypt.enabled", cfg.Open115Encrypt.Enabled)
	v.Set("open115_encrypt.password", cfg.Open115Encrypt.Password)
	v.Set("open115_encrypt.salt", cfg.Open115Encrypt.Salt)
	v.Set("open115_encrypt.mode", cfg.Open115Encrypt.Mode)
	v.Set("open115_encrypt.filename_mode", cfg.Open115Encrypt.FilenameMode)
	v.Set("open115_encrypt.algorithm", cfg.Open115Encrypt.Algorithm)
	v.Set("open115_encrypt.temp_dir", cfg.Open115Encrypt.TempDir)
	v.Set("open115_encrypt.min_free_space_mb", cfg.Open115Encrypt.MinFreeSpaceMB)

	v.Set("backup.library_dir", cfg.Backup.LibraryDir)
	v.Set("backup.backups_dir", cfg.Backup.BackupsDir)
	v.Set("backup.remote_dir", cfg.Backup.RemoteDir)
	v.Set("backup.mode", cfg.Backup.Mode)
	v.Set("backup.manifest_path", cfg.Backup.ManifestPath)
	v.Set("backup.allow_remote_delete", cfg.Backup.AllowRemoteDelete)

	v.Set("encrypt.enabled", cfg.Encrypt.Enabled)
	v.Set("encrypt.password", cfg.Encrypt.Password)
	v.Set("encrypt.salt", cfg.Encrypt.Salt)

	v.Set("cron.enabled", cfg.Cron.Enabled)
	v.Set("cron.expression", cfg.Cron.Expression)

	v.Set("notify.enabled", cfg.Notify.Enabled)
	v.Set("notify.bark_url", cfg.Notify.BarkURL)

	v.Set("photo_upload.enabled", cfg.PhotoUpload.Enabled)
	v.Set("photo_upload.watch_dir", cfg.PhotoUpload.WatchDir)
	v.Set("photo_upload.remote_dir", cfg.PhotoUpload.RemoteDir)
	v.Set("photo_upload.extensions", cfg.PhotoUpload.Extensions)
	v.Set("photo_upload.date_format", cfg.PhotoUpload.DateFormat)
	v.Set("photo_upload.delete_after_upload", cfg.PhotoUpload.DeleteAfterUpload)

	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	m.cfg = &cfg
	return nil
}

func HashPassword(plaintext string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

func VerifyPassword(hash string, plaintext string) bool {
	if hash == "" || plaintext == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)) == nil
}

// IsSetupComplete 检查是否已完成初始配置。
func (m *Manager) IsSetupComplete() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cfg := m.cfg
	provider := strings.TrimSpace(cfg.Provider)
	if provider == "" {
		provider = "webdav"
	}

	if provider == "webdav" {
		if strings.TrimSpace(cfg.WebDAV.URL) == "" || strings.TrimSpace(cfg.WebDAV.User) == "" || strings.TrimSpace(cfg.WebDAV.Password) == "" {
			return false
		}
	} else if provider == "open115" {
		if strings.TrimSpace(cfg.Open115.AccessToken) == "" || strings.TrimSpace(cfg.Open115.RefreshToken) == "" {
			return false
		}
	} else {
		return false
	}
	if strings.TrimSpace(cfg.Backup.RemoteDir) == "" {
		return false
	}
	if strings.TrimSpace(cfg.Backup.LibraryDir) == "" && strings.TrimSpace(cfg.Backup.BackupsDir) == "" {
		return false
	}
	if cfg.Encrypt.Enabled && strings.TrimSpace(cfg.Encrypt.Password) == "" {
		return false
	}
	if provider == "open115" && cfg.Open115Encrypt.Enabled && strings.TrimSpace(cfg.Open115Encrypt.Password) == "" {
		return false
	}
	return true
}
