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

func TestUpdateMonotonicVersion(t *testing.T) {
	cfgPath := t.TempDir() + "/config.yaml"
	mgr, err := NewManager(cfgPath)
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}

	// First update
	cfg := mgr.Get()
	cfg.Provider = "webdav"
	if err := mgr.Update(cfg); err != nil {
		t.Fatalf("first Update error: %v", err)
	}
	v1 := mgr.Get().UpdatedAt

	// Second update immediately (same second)
	cfg2 := mgr.Get()
	cfg2.Server.Port = 9999
	if err := mgr.Update(cfg2); err != nil {
		t.Fatalf("second Update error: %v", err)
	}
	v2 := mgr.Get().UpdatedAt

	if v2 <= v1 {
		t.Fatalf("expected monotonic increase: v1=%d, v2=%d", v1, v2)
	}

	// Third update immediately
	cfg3 := mgr.Get()
	cfg3.Server.Port = 8888
	if err := mgr.Update(cfg3); err != nil {
		t.Fatalf("third Update error: %v", err)
	}
	v3 := mgr.Get().UpdatedAt

	if v3 <= v2 {
		t.Fatalf("expected monotonic increase: v2=%d, v3=%d", v2, v3)
	}
}

func TestApplyUpdateStaleConflict(t *testing.T) {
	cfgPath := t.TempDir() + "/config.yaml"
	mgr, err := NewManager(cfgPath)
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}

	// Set up initial config
	cfg := mgr.Get()
	cfg.Provider = "webdav"
	cfg.WebDAV = WebDAVConfig{URL: "https://dav.example.com", User: "user", Password: "pass"}
	cfg.Backup = BackupConfig{RemoteDir: "/backup", LibraryDir: "/lib"}
	if err := mgr.Update(cfg); err != nil {
		t.Fatalf("initial Update error: %v", err)
	}
	staleVersion := mgr.Get().UpdatedAt

	// Simulate a background Update (e.g., token refresh) that bumps the version
	cfg2 := mgr.Get()
	cfg2.Open115.AccessToken = "refreshed-token"
	if err := mgr.Update(cfg2); err != nil {
		t.Fatalf("background Update error: %v", err)
	}
	newVersion := mgr.Get().UpdatedAt
	if newVersion <= staleVersion {
		t.Fatalf("background update should have bumped version: stale=%d, new=%d", staleVersion, newVersion)
	}

	// Now try ApplyUpdate with the stale version → must fail
	staleReq := ConfigUpdateRequest{
		Provider:  "webdav",
		Backup:    cfg.Backup,
		UpdatedAt: staleVersion,
	}
	_, err = mgr.ApplyUpdate(staleReq)
	if err == nil {
		t.Fatal("expected ApplyUpdate with stale version to fail")
	}
	if err.Error() != "配置已被其他页面修改，请刷新后重试" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}

	// ApplyUpdate with current version → must succeed
	currentReq := ConfigUpdateRequest{
		Provider:  "webdav",
		Backup:    cfg.Backup,
		UpdatedAt: newVersion,
	}
	_, err = mgr.ApplyUpdate(currentReq)
	if err != nil {
		t.Fatalf("ApplyUpdate with current version should succeed, got: %v", err)
	}
}
