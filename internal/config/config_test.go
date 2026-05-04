package config

import (
	"os"
	"path/filepath"
	"testing"
)

func redirectHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	return tmp
}

func TestLoad_MissingFile_ReturnsEmptyConfig(t *testing.T) {
	redirectHome(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.ClientID != "" || cfg.DefaultOwner != "" || cfg.DefaultRepo != "" {
		t.Errorf("expected empty config, got %+v", cfg)
	}
}

func TestSaveAndLoad_Roundtrip(t *testing.T) {
	redirectHome(t)

	want := &Config{
		ClientID:     "test-client-id",
		DefaultOwner: "myorg",
		DefaultRepo:  "myrepo",
	}
	if err := Save(want); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if got.ClientID != want.ClientID {
		t.Errorf("ClientID: got %q, want %q", got.ClientID, want.ClientID)
	}
	if got.DefaultOwner != want.DefaultOwner {
		t.Errorf("DefaultOwner: got %q, want %q", got.DefaultOwner, want.DefaultOwner)
	}
	if got.DefaultRepo != want.DefaultRepo {
		t.Errorf("DefaultRepo: got %q, want %q", got.DefaultRepo, want.DefaultRepo)
	}
}

func TestLoad_ClientIDEnvOverride(t *testing.T) {
	redirectHome(t)
	t.Setenv("GHTUI_CLIENT_ID", "env-client-id")

	if err := Save(&Config{ClientID: "file-client-id"}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.ClientID != "env-client-id" {
		t.Errorf("expected env var to override file: got %q", cfg.ClientID)
	}
}

func TestTokenPath_UnderConfigDir(t *testing.T) {
	home := redirectHome(t)
	path, err := TokenPath()
	if err != nil {
		t.Fatalf("TokenPath() error: %v", err)
	}
	expected := filepath.Join(home, ".config", "ghtui", "token")
	if path != expected {
		t.Errorf("TokenPath() = %q, want %q", path, expected)
	}
}

func TestSave_CreatesDirectories(t *testing.T) {
	home := redirectHome(t)
	if err := Save(&Config{ClientID: "x"}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	cfgFile := filepath.Join(home, ".config", "ghtui", "config.json")
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		t.Errorf("expected config file to exist at %q", cfgFile)
	}
}
