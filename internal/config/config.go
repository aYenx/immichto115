package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

// AppConfig 应用全局配置结构。
type AppConfig struct {
	Server  ServerConfig  `mapstructure:"server"  json:"server"  yaml:"server"`
	WebDAV  WebDAVConfig  `mapstructure:"webdav"  json:"webdav"  yaml:"webdav"`
	Backup  BackupConfig  `mapstructure:"backup"  json:"backup"  yaml:"backup"`
	Encrypt EncryptConfig `mapstructure:"encrypt" json:"encrypt" yaml:"encrypt"`
	Cron    CronConfig    `mapstructure:"cron"    json:"cron"    yaml:"cron"`
}

// ServerConfig 服务器配置。
type ServerConfig struct {
	Port int `mapstructure:"port" json:"port" yaml:"port"`
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config dir: %w", err)
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置默认值
	viper.SetDefault("server.port", 8096)
	viper.SetDefault("webdav.vendor", "other")
	viper.SetDefault("backup.remote_dir", "/immich-backup")
	viper.SetDefault("cron.expression", "0 2 * * *")
	viper.SetDefault("cron.enabled", false)
	viper.SetDefault("encrypt.enabled", false)

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

	viper.Set("server", cfg.Server)
	viper.Set("webdav", cfg.WebDAV)
	viper.Set("backup", cfg.Backup)
	viper.Set("encrypt", cfg.Encrypt)
	viper.Set("cron", cfg.Cron)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	m.cfg = &cfg
	return nil
}

// IsSetupComplete 检查是否已完成初始配置。
func (m *Manager) IsSetupComplete() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.WebDAV.URL != "" && m.cfg.WebDAV.User != ""
}
