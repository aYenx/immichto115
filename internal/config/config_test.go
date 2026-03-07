package config

import "testing"

func TestIsSetupComplete(t *testing.T) {
	base := &Manager{cfg: &AppConfig{
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
}
