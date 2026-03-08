package config

import "testing"

func TestIsSetupComplete(t *testing.T) {
	base := &Manager{cfg: &AppConfig{
		Provider: "webdav",
		WebDAV: WebDAVConfig{
			URL:      "https://dav.example.com",
			User:     "user",
			Password: "secret",
		},
		Backup: BackupConfig{
			RemoteDir:  "/immich-backup",
			LibraryDir: "/data/library",
		},
	}}

	if !base.IsSetupComplete() {
		t.Fatalf("expected setup to be complete for valid minimal config")
	}

	cases := []struct {
		name string
		mutate func(cfg *AppConfig)
	}{
		{
			name: "missing webdav url",
			mutate: func(cfg *AppConfig) { cfg.WebDAV.URL = "" },
		},
		{
			name: "missing remote dir",
			mutate: func(cfg *AppConfig) { cfg.Backup.RemoteDir = "" },
		},
		{
			name: "missing all backup paths",
			mutate: func(cfg *AppConfig) {
				cfg.Backup.LibraryDir = ""
				cfg.Backup.BackupsDir = ""
			},
		},
		{
			name: "encrypt enabled without password",
			mutate: func(cfg *AppConfig) {
				cfg.Encrypt.Enabled = true
				cfg.Encrypt.Password = ""
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgCopy := *base.cfg
			tc.mutate(&cfgCopy)
			mgr := &Manager{cfg: &cfgCopy}
			if mgr.IsSetupComplete() {
				t.Fatalf("expected setup to be incomplete for case %q", tc.name)
			}
		})
	}

	t.Run("open115 minimal config", func(t *testing.T) {
		mgr := &Manager{cfg: &AppConfig{
			Provider: "open115",
			Open115: Open115Config{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				RootID:       "0",
			},
			Backup: BackupConfig{
				RemoteDir:  "/immich-backup",
				LibraryDir: "/data/library",
			},
		}}
		if !mgr.IsSetupComplete() {
			t.Fatalf("expected open115 setup to be complete for valid minimal config")
		}
	})

	t.Run("open115 missing refresh token", func(t *testing.T) {
		mgr := &Manager{cfg: &AppConfig{
			Provider: "open115",
			Open115: Open115Config{
				AccessToken: "access-token",
			},
			Backup: BackupConfig{
				RemoteDir:  "/immich-backup",
				LibraryDir: "/data/library",
			},
		}}
		if mgr.IsSetupComplete() {
			t.Fatalf("expected open115 setup to be incomplete when refresh token is missing")
		}
	})
}
