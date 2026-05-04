package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	ClientID     string `json:"client_id,omitempty"`
	DefaultOwner string `json:"default_owner,omitempty"`
	DefaultRepo  string `json:"default_repo,omitempty"`
}

func dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(home, ".config", "ghtui")
	return d, os.MkdirAll(d, 0700)
}

func Load() (*Config, error) {
	d, err := dir()
	if err != nil {
		return &Config{}, nil
	}
	data, err := os.ReadFile(filepath.Join(d, "config.json"))
	if err != nil {
		return &Config{}, nil
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &Config{}, nil
	}
	// env var takes precedence
	if v := os.Getenv("GHTUI_CLIENT_ID"); v != "" {
		cfg.ClientID = v
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	d, err := dir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d, "config.json"), data, 0600)
}

func TokenPath() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "token"), nil
}
