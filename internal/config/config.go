package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// AppConfig 应用全部配置。
type AppConfig struct {
	Server  ServerConfig  `json:"server" mapstructure:"server"`
	WebDAV  WebDAVConfig  `json:"webdav" mapstructure:"webdav"`
	Backup  BackupConfig  `json:"backup" mapstructure:"backup"`
	Encrypt EncryptConfig `json:"encrypt" mapstructure:"encrypt"`
	Cron    CronConfig    `json:"cron" mapstructure:"cron"`
}

// ServerConfig 服务器配置。
type ServerConfig struct {
	Port int `json:"port" mapstructure:"port"`
}

// WebDAVConfig WebDAV 连接配置。
type WebDAVConfig struct {
	URL      string `json:"url" mapstructure:"url"`
	User     string `json:"user" mapstructure:"user"`
	Password string `json:"password" mapstructure:"password"`
}

// BackupConfig 备份路径配置。
type BackupConfig struct {
	LibraryDir string `json:"library_dir" mapstructure:"library_dir"`
	BackupsDir string `json:"backups_dir" mapstructure:"backups_dir"`
}

// EncryptConfig 加密配置。
type EncryptConfig struct {
	Enabled  bool   `json:"enabled" mapstructure:"enabled"`
	Password string `json:"password" mapstructure:"password"`
	Salt     string `json:"salt" mapstructure:"salt"`
}

// CronConfig 定时任务配置。
type CronConfig struct {
	Enabled    bool   `json:"enabled" mapstructure:"enabled"`
	Expression string `json:"expression" mapstructure:"expression"`
}

// Manager 管理配置的读写。
type Manager struct {
	mu       sync.RWMutex
	cfg      AppConfig
	filePath string
}

// NewManager 创建一个新的配置管理器，从指定路径加载配置文件。
// 如果配置文件不存在，则使用默认配置并创建文件。
func NewManager(cfgPath string) (*Manager, error) {
	m := &Manager{
		filePath: cfgPath,
	}

	// 确保配置文件目录存在
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(cfgPath)
	v.SetConfigType("yaml")

	// 默认值
	v.SetDefault("server.port", 8096)

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundErr viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundErr) {
			log.Printf("[config] config file not found, using defaults: %s", cfgPath)
		} else if os.IsNotExist(err) {
			log.Printf("[config] config file not found, using defaults: %s", cfgPath)
		} else {
			// 文件存在但解析失败才报错
			if _, statErr := os.Stat(cfgPath); statErr == nil {
				return nil, fmt.Errorf("failed to read config: %w", err)
			}
			log.Printf("[config] config file not found, using defaults: %s", cfgPath)
		}
	}

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	m.cfg = cfg
	return m, nil
}

// Get 返回当前配置的副本。
func (m *Manager) Get() AppConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

// Update 更新并持久化配置。
func (m *Manager) Update(newCfg AppConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	v := viper.New()
	v.SetConfigFile(m.filePath)
	v.SetConfigType("yaml")

	v.Set("server", newCfg.Server)
	v.Set("webdav", newCfg.WebDAV)
	v.Set("backup", newCfg.Backup)
	v.Set("encrypt", newCfg.Encrypt)
	v.Set("cron", newCfg.Cron)

	if err := v.WriteConfigAs(m.filePath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	m.cfg = newCfg
	return nil
}

// IsSetupComplete 检查配置是否已完成初始设置（WebDAV 已配置且至少有一个备份路径）。
func (m *Manager) IsSetupComplete() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.cfg.WebDAV.URL != "" &&
		m.cfg.WebDAV.User != "" &&
		m.cfg.WebDAV.Password != "" &&
		(m.cfg.Backup.LibraryDir != "" || m.cfg.Backup.BackupsDir != "")
}

// GenerateRcloneConf 根据配置生成临时的 rclone.conf 文件，返回文件路径。
func GenerateRcloneConf(cfg AppConfig) (string, error) {
	obscured, err := ObscurePassword(cfg.WebDAV.Password)
	if err != nil {
		return "", fmt.Errorf("failed to obscure webdav password: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("[webdav]\n")
	sb.WriteString("type = webdav\n")
	sb.WriteString(fmt.Sprintf("url = %s\n", cfg.WebDAV.URL))
	sb.WriteString(fmt.Sprintf("user = %s\n", cfg.WebDAV.User))
	sb.WriteString(fmt.Sprintf("pass = %s\n", obscured))

	if cfg.Encrypt.Enabled {
		encPass, err := ObscurePassword(cfg.Encrypt.Password)
		if err != nil {
			return "", fmt.Errorf("failed to obscure encrypt password: %w", err)
		}
		encSalt, err := ObscurePassword(cfg.Encrypt.Salt)
		if err != nil {
			return "", fmt.Errorf("failed to obscure encrypt salt: %w", err)
		}
		sb.WriteString("\n[crypt]\n")
		sb.WriteString("type = crypt\n")
		sb.WriteString("remote = webdav:\n")
		sb.WriteString(fmt.Sprintf("password = %s\n", encPass))
		sb.WriteString(fmt.Sprintf("password2 = %s\n", encSalt))
	}

	tmpFile, err := os.CreateTemp("", "rclone-*.conf")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := tmpFile.WriteString(sb.String()); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write rclone config: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to close rclone config: %w", err)
	}

	return tmpFile.Name(), nil
}

// GetRemoteName 根据加密配置返回 rclone remote 名称。
func GetRemoteName(cfg AppConfig) string {
	if cfg.Encrypt.Enabled {
		return "crypt:"
	}
	return "webdav:"
}

// CleanupRcloneConf 删除临时生成的 rclone.conf 文件。
func CleanupRcloneConf(confPath string) {
	if confPath == "" {
		return
	}
	if err := os.Remove(confPath); err != nil && !os.IsNotExist(err) {
		log.Printf("[config] failed to cleanup rclone config %s: %v", confPath, err)
	}
}

// ObscurePassword 使用 rclone obscure 命令对密码进行加密处理。
func ObscurePassword(password string) (string, error) {
	if password == "" {
		return "", nil
	}
	cmd := exec.Command("rclone", "obscure", password)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("rclone obscure failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
