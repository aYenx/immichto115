package config

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// AppConfig 是应用的完整配置结构。
type AppConfig struct {
	Server  ServerConfig  `json:"server" mapstructure:"server"`
	WebDAV  WebDAVConfig  `json:"webdav" mapstructure:"webdav"`
	Encrypt EncryptConfig `json:"encrypt" mapstructure:"encrypt"`
	Backup  BackupConfig  `json:"backup" mapstructure:"backup"`
	Cron    CronConfig    `json:"cron" mapstructure:"cron"`
}

// ServerConfig 服务端口配置。
type ServerConfig struct {
	Port int `json:"port" mapstructure:"port"`
}

// WebDAVConfig WebDAV 连接配置。
type WebDAVConfig struct {
	URL      string `json:"url" mapstructure:"url"`
	User     string `json:"user" mapstructure:"user"`
	Password string `json:"password" mapstructure:"password"`
}

// EncryptConfig rclone crypt 加密配置。
type EncryptConfig struct {
	Enabled  bool   `json:"enabled" mapstructure:"enabled"`
	Password string `json:"password" mapstructure:"password"`
	Salt     string `json:"salt" mapstructure:"salt"`
}

// BackupConfig 备份目录配置。
type BackupConfig struct {
	LibraryDir string `json:"library_dir" mapstructure:"library_dir"`
	BackupsDir string `json:"backups_dir" mapstructure:"backups_dir"`
}

// CronConfig 定时任务配置。
type CronConfig struct {
	Enabled    bool   `json:"enabled" mapstructure:"enabled"`
	Expression string `json:"expression" mapstructure:"expression"`
}

// Manager 管理配置的读写和持久化。
type Manager struct {
	mu       sync.RWMutex
	config   AppConfig
	filePath string
	v        *viper.Viper
}

// NewManager 创建配置管理器并从文件加载配置。
// 如果配置文件不存在，则使用默认配置。
func NewManager(path string) (*Manager, error) {
	v := viper.New()

	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]

	v.SetConfigName(name)
	v.SetConfigType("yaml")
	v.AddConfigPath(dir)

	// 设置默认值
	v.SetDefault("server.port", 8096)

	m := &Manager{
		filePath: path,
		v:        v,
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("[config] config file not found at %s, using defaults", path)
		} else if os.IsNotExist(err) {
			log.Printf("[config] config file not found at %s, using defaults", path)
		} else {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	m.config = cfg

	log.Printf("[config] loaded config from %s", path)
	return m, nil
}

// Get 返回当前配置的副本。
func (m *Manager) Get() AppConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Update 更新配置并持久化到文件。
func (m *Manager) Update(cfg AppConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = cfg

	// 确保配置目录存在
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	m.v.Set("server", cfg.Server)
	m.v.Set("webdav", cfg.WebDAV)
	m.v.Set("encrypt", cfg.Encrypt)
	m.v.Set("backup", cfg.Backup)
	m.v.Set("cron", cfg.Cron)

	if err := m.v.WriteConfigAs(m.filePath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	log.Printf("[config] config saved to %s", m.filePath)
	return nil
}

// IsSetupComplete 检查是否已完成最基本的配置。
func (m *Manager) IsSetupComplete() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cfg := m.config
	return cfg.WebDAV.URL != "" &&
		cfg.WebDAV.User != "" &&
		cfg.WebDAV.Password != "" &&
		(cfg.Backup.LibraryDir != "" || cfg.Backup.BackupsDir != "")
}

// GenerateRcloneConf 根据当前配置生成临时 rclone.conf 文件。
// 返回临时文件路径，调用者需在用完后调用 CleanupRcloneConf 清理。
func GenerateRcloneConf(cfg AppConfig) (string, error) {
	obscuredPass, err := ObscurePassword(cfg.WebDAV.Password)
	if err != nil {
		return "", fmt.Errorf("failed to obscure webdav password: %w", err)
	}

	// 构建 rclone.conf 内容
	conf := "[webdav]\ntype = webdav\n"
	conf += "url = " + cfg.WebDAV.URL + "\n"
	conf += "user = " + cfg.WebDAV.User + "\n"
	conf += "pass = " + obscuredPass + "\n"

	if cfg.Encrypt.Enabled {
		encPass, err := ObscurePassword(cfg.Encrypt.Password)
		if err != nil {
			return "", fmt.Errorf("failed to obscure encrypt password: %w", err)
		}

		conf += "\n[crypt]\ntype = crypt\n"
		conf += "remote = webdav:\n"
		conf += "password = " + encPass + "\n"

		if cfg.Encrypt.Salt != "" {
			encSalt, err := ObscurePassword(cfg.Encrypt.Salt)
			if err != nil {
				return "", fmt.Errorf("failed to obscure encrypt salt: %w", err)
			}
			conf += "password2 = " + encSalt + "\n"
		}
	}

	tmpFile, err := os.CreateTemp("", "rclone-*.conf")
	if err != nil {
		return "", fmt.Errorf("failed to create temp rclone config: %w", err)
	}

	if _, err := tmpFile.WriteString(conf); err != nil {
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

// GetRemoteName 根据配置返回 rclone 远端名称。
// 如果启用了加密，返回 "crypt:"，否则返回 "webdav:"。
func GetRemoteName(cfg AppConfig) string {
	if cfg.Encrypt.Enabled {
		return "crypt:"
	}
	return "webdav:"
}

// ObscurePassword 使用 rclone obscure 命令对密码进行混淆。
func ObscurePassword(password string) (string, error) {
	cmd := exec.Command("rclone", "obscure", password)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("rclone obscure failed: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// CleanupRcloneConf 删除临时 rclone 配置文件。
func CleanupRcloneConf(confPath string) {
	if confPath == "" {
		return
	}
	if err := os.Remove(confPath); err != nil && !os.IsNotExist(err) {
		log.Printf("[config] warning: failed to cleanup rclone config %s: %v", confPath, err)
	}
}
