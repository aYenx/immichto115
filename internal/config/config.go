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
	Server  ServerConfig  `mapstructure:"server"  json:"server"  yaml:"server"`
	WebDAV  WebDAVConfig  `mapstructure:"webdav"  json:"webdav"  yaml:"webdav"`
	Backup  BackupConfig  `mapstructure:"backup"  json:"backup"  yaml:"backup"`
	Encrypt EncryptConfig `mapstructure:"encrypt" json:"encrypt" yaml:"encrypt"`
	Cron    CronConfig    `mapstructure:"cron"    json:"cron"    yaml:"cron"`
	Notify  NotifyConfig  `mapstructure:"notify"  json:"notify"  yaml:"notify"`
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

// BackupConfig 备份源和目标配置。
type BackupConfig struct {
	LibraryDir string `mapstructure:"library_dir" json:"library_dir" yaml:"library_dir"`
	BackupsDir string `mapstructure:"backups_dir" json:"backups_dir" yaml:"backups_dir"`
	RemoteDir  string `mapstructure:"remote_dir"  json:"remote_dir"  yaml:"remote_dir"`
	Mode       string `mapstructure:"mode"        json:"mode"        yaml:"mode"` // "copy" (增量) 或 "sync" (镜像)
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

// Manager 配置管理器，线程安全。
type Manager struct {
	mu       sync.RWMutex
	cfg      *AppConfig
	filePath string
}

// NewManager 创建配置管理器并加载或初始化配置文件。
func NewManager(configPath string) (*Manager, error) {
	m := &Manager{
		filePath: configPath,
	}

	// 确保配置目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config dir: %w", err)
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置默认值
	viper.SetDefault("server.port", 8096)
	viper.SetDefault("server.auth_enabled", false)
	viper.SetDefault("webdav.vendor", "other")
	viper.SetDefault("backup.remote_dir", "/immich-backup")
	viper.SetDefault("backup.mode", "copy")
	viper.SetDefault("cron.expression", "0 2 * * *")
	viper.SetDefault("cron.enabled", false)
	viper.SetDefault("encrypt.enabled", false)
	viper.SetDefault("notify.enabled", false)

	// 尝试读取已有配置
	if err := viper.ReadInConfig(); err != nil {
		// SetConfigFile 时文件不存在返回 os.PathError，而非 viper.ConfigFileNotFoundError
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) || os.IsNotExist(err) {
			// 文件不存在，写入默认配置
			if writeErr := viper.SafeWriteConfig(); writeErr != nil {
				if writeErr2 := viper.WriteConfig(); writeErr2 != nil {
					return nil, fmt.Errorf("failed to write default config: %w", writeErr2)
				}
			}
		} else {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	cfg := &AppConfig{}
	if err := viper.Unmarshal(cfg); err != nil {
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

// Update 更新配置并持久化到文件。
func (m *Manager) Update(cfg AppConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	viper.Set("server.port", cfg.Server.Port)
	viper.Set("server.auth_enabled", cfg.Server.AuthEnabled)
	viper.Set("server.auth_user", cfg.Server.AuthUser)
	viper.Set("server.auth_password_hash", cfg.Server.AuthPasswordHash)

	viper.Set("webdav.url", cfg.WebDAV.URL)
	viper.Set("webdav.user", cfg.WebDAV.User)
	viper.Set("webdav.password", cfg.WebDAV.Password)
	viper.Set("webdav.vendor", cfg.WebDAV.Vendor)

	viper.Set("backup.library_dir", cfg.Backup.LibraryDir)
	viper.Set("backup.backups_dir", cfg.Backup.BackupsDir)
	viper.Set("backup.remote_dir", cfg.Backup.RemoteDir)
	viper.Set("backup.mode", cfg.Backup.Mode)

	viper.Set("encrypt.enabled", cfg.Encrypt.Enabled)
	viper.Set("encrypt.password", cfg.Encrypt.Password)
	viper.Set("encrypt.salt", cfg.Encrypt.Salt)

	viper.Set("cron.enabled", cfg.Cron.Enabled)
	viper.Set("cron.expression", cfg.Cron.Expression)

	viper.Set("notify.enabled", cfg.Notify.Enabled)
	viper.Set("notify.bark_url", cfg.Notify.BarkURL)

	if err := viper.WriteConfig(); err != nil {
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
	if strings.TrimSpace(cfg.WebDAV.URL) == "" || strings.TrimSpace(cfg.WebDAV.User) == "" || strings.TrimSpace(cfg.WebDAV.Password) == "" {
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
	return true
}
