package open115crypt

import "github.com/aYenx/immichto115/internal/config"

type Config struct {
	Enabled        bool
	Password       string
	Salt           string
	Mode           string
	FilenameMode   string
	Algorithm      string
	TempDir        string
	MinFreeSpaceMB int64
}

func FromAppConfig(cfg config.AppConfig) Config {
	return Config{
		Enabled:        cfg.Open115Encrypt.Enabled,
		Password:       cfg.Open115Encrypt.Password,
		Salt:           cfg.Open115Encrypt.Salt,
		Mode:           cfg.Open115Encrypt.Mode,
		FilenameMode:   cfg.Open115Encrypt.FilenameMode,
		Algorithm:      cfg.Open115Encrypt.Algorithm,
		TempDir:        cfg.Open115Encrypt.TempDir,
		MinFreeSpaceMB: cfg.Open115Encrypt.MinFreeSpaceMB,
	}
}
