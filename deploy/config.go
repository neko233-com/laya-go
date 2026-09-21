package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// AutoUpdateConfig controls GitHub release checks and optional apply.
type AutoUpdateConfig struct {
	Enabled     bool   `json:"enabled"`
	AutoApply   bool   `json:"auto_apply"`
	IntervalMin int    `json:"interval_minutes"`
	Repo        string `json:"repo"`
	Prerelease  bool   `json:"prerelease"`
}

// Config is the deploy server runtime configuration.
type Config struct {
	Addr       string           `json:"addr"`
	ConfigPath string           `json:"config_path,omitempty"`
	AutoUpdate AutoUpdateConfig `json:"auto_update"`
}

func defaultConfig() Config {
	return Config{
		Addr: "127.0.0.1:7400",
		AutoUpdate: AutoUpdateConfig{
			Enabled:     true,
			AutoApply:   true,
			IntervalMin: 360,
			Repo:        "neko233-com/laya-go",
			Prerelease:  false,
		},
	}
}

// resolveConfigPath picks the config file location.
// Priority: LAYA_CONFIG > ProgramData\Laya\config.json (if exists) > exeDir\laya-config.json
func resolveConfigPath() string {
	if p := os.Getenv("LAYA_CONFIG"); p != "" {
		return p
	}
	exe, err := os.Executable()
	var exeDir string
	if err == nil {
		exeDir = filepath.Dir(exe)
	}
	if runtime.GOOS == "windows" {
		if pd := os.Getenv("ProgramData"); pd != "" {
			p := filepath.Join(pd, "Laya", "config.json")
			if fileExists(p) {
				return p
			}
			if exeDir != "" {
				// Prefer service install layout when directory exists.
				if fileExists(filepath.Join(pd, "Laya", "laya-deploy"+exeSuffix())) {
					return p
				}
			}
		}
	}
	if exeDir != "" {
		return filepath.Join(exeDir, "laya-config.json")
	}
	return "laya-config.json"
}

func loadConfig(path string) Config {
	cfg := defaultConfig()
	cfg.ConfigPath = path
	if path == "" || !fileExists(path) {
		return cfg
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	var disk Config
	if err := json.Unmarshal(raw, &disk); err != nil {
		return cfg
	}
	if disk.Addr != "" {
		cfg.Addr = disk.Addr
	}
	// Overlay auto_update; unmarshal zero values are meaningful for bools,
	// so only fill struct after full replace when auto_update object present.
	if hasJSONKey(raw, "auto_update") {
		cfg.AutoUpdate = disk.AutoUpdate
	}
	if cfg.AutoUpdate.IntervalMin <= 0 {
		cfg.AutoUpdate.IntervalMin = 360
	}
	if cfg.AutoUpdate.Repo == "" {
		cfg.AutoUpdate.Repo = defaultConfig().AutoUpdate.Repo
	}
	if cfg.Addr == "" {
		cfg.Addr = defaultConfig().Addr
	}
	return cfg
}

func saveConfig(path string, cfg Config) error {
	if path == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out := cfg
	out.ConfigPath = ""
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func hasJSONKey(raw []byte, key string) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return false
	}
	_, ok := m[key]
	return ok
}
