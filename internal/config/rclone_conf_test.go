package config

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateRcloneConfDefaultsVendorToOther(t *testing.T) {
	confPath, err := GenerateRcloneConf(AppConfig{
		WebDAV: WebDAVConfig{
			URL:      "https://dav.example.com",
			User:     "user",
			Password: "secret",
			Vendor:   "",
		},
	})
	if err != nil {
		t.Fatalf("GenerateRcloneConf error: %v", err)
	}
	defer CleanupRcloneConf(confPath)

	content, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if !strings.Contains(string(content), "vendor = other") {
		t.Fatalf("expected generated config to default vendor to other, got:\n%s", string(content))
	}
}

func TestBuildRemotePath(t *testing.T) {
	tests := []struct {
		name string
		cfg  AppConfig
		path string
		want string
	}{
		{
			name: "plain root",
			cfg: AppConfig{Backup: BackupConfig{RemoteDir: "/immich-backup"}},
			path: "/",
			want: "webdav115:/immich-backup",
		},
		{
			name: "plain nested path",
			cfg: AppConfig{Backup: BackupConfig{RemoteDir: "/immich-backup"}},
			path: "/albums/2026",
			want: "webdav115:/immich-backup/albums/2026",
		},
		{
			name: "crypt root",
			cfg: AppConfig{Encrypt: EncryptConfig{Enabled: true}, Backup: BackupConfig{RemoteDir: "/immich-backup"}},
			path: "/",
			want: "crypt115:",
		},
		{
			name: "crypt nested path",
			cfg: AppConfig{Encrypt: EncryptConfig{Enabled: true}, Backup: BackupConfig{RemoteDir: "/immich-backup"}},
			path: "/albums/2026",
			want: "crypt115:albums/2026",
		},
		{
			name: "plain empty path treated as root",
			cfg: AppConfig{Backup: BackupConfig{RemoteDir: "/immich-backup"}},
			path: "",
			want: "webdav115:/immich-backup",
		},
		{
			name: "plain path is normalized",
			cfg: AppConfig{Backup: BackupConfig{RemoteDir: "/immich-backup"}},
			path: "//albums///2026//",
			want: "webdav115:/immich-backup/albums/2026",
		},
		{
			name: "crypt path is normalized",
			cfg: AppConfig{Encrypt: EncryptConfig{Enabled: true}, Backup: BackupConfig{RemoteDir: "/immich-backup"}},
			path: "//albums///2026//",
			want: "crypt115:albums/2026",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildRemotePath(tt.cfg, tt.path)
			if got != tt.want {
				t.Fatalf("BuildRemotePath() = %q, want %q", got, tt.want)
			}
		})
	}
}
