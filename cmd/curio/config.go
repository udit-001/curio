package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

func configDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "curio")
	case "windows":
		return filepath.Join(os.Getenv("AppData"), "curio")
	default: // linux, bsd, etc.
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "curio")
		}
		return filepath.Join(os.Getenv("HOME"), ".config", "curio")
	}
}

var configDirPath = configDir()

// ConfigPath is the TOML config file.
var ConfigPath = filepath.Join(configDirPath, "config.toml")

// EnvPath is the legacy .env file (migrated to TOML on first run).
var EnvPath = filepath.Join(configDirPath, ".env")

// configData holds the parsed TOML config, flattened to "section.key" → value.
var configData = loadConfig()

// loadConfig reads config.toml (or migrates from legacy .env or old image-fetcher dir).
// Keys are "section.key" (e.g. "smithsonian.api_key").
func loadConfig() map[string]string {
	// Migrate old image-fetcher config dir → curio config dir
	migrateOldConfigDir()

	cfg := map[string]string{}

	// Migrate legacy .env → config.toml if TOML doesn't exist yet
	if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
		migrateEnvToToml()
	}

	// Parse TOML
	if data, err := os.ReadFile(ConfigPath); err == nil {
		var raw map[string]map[string]string
		if err := toml.Unmarshal(data, &raw); err != nil {
			fmt.Fprintf(os.Stderr, "  ! warning: config.toml parse error: %v\n", err)
		} else {
			for section, keys := range raw {
				for k, v := range keys {
					cfg[section+"."+k] = v
				}
			}
		}
	}

	return cfg
}

// migrateOldConfigDir moves the old image-fetcher config dir to curio if curio doesn't exist yet.
func migrateOldConfigDir() {
	if _, err := os.Stat(ConfigPath); err == nil {
		return // curio config already exists
	}

	oldDir := filepath.Join(filepath.Dir(configDirPath), "image-fetcher")
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return // no old config to migrate
	}

	_ = os.MkdirAll(configDirPath, 0755)
	entries, _ := os.ReadDir(oldDir)
	for _, e := range entries {
		oldPath := filepath.Join(oldDir, e.Name())
		newPath := filepath.Join(configDirPath, e.Name())
		_ = os.Rename(oldPath, newPath)
	}
}

// migrateEnvToToml converts a legacy .env file to config.toml, preserving values.
// One-time migration — the mapping is inline because it's never used again.
func migrateEnvToToml() {
	data, err := os.ReadFile(EnvPath)
	if err != nil {
		return
	}

	mapping := map[string]string{
		"OPENVERSE_CLIENT_ID":     "openverse.client_id",
		"OPENVERSE_CLIENT_SECRET": "openverse.client_secret",
		"WIKIMEDIA_BOT_USER":      "wikimedia.bot_user",
		"WIKIMEDIA_BOT_PASS":      "wikimedia.bot_pass",
		"DATA_GOV_API_KEY":        "smithsonian.api_key",
		"EUROPEANA_API_KEY":       "europeana.api_key",
	}

	raw := map[string]map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			oldKey := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			newKey, ok := mapping[oldKey]
			if !ok {
				continue
			}
			parts := strings.SplitN(newKey, ".", 2)
			if raw[parts[0]] == nil {
				raw[parts[0]] = map[string]string{}
			}
			raw[parts[0]][parts[1]] = val
		}
	}

	if len(raw) > 0 {
		writeToml(raw)
		_ = os.Rename(EnvPath, EnvPath+".migrated")
	}
}

// configGet returns the value for a "section.key" key, or "" if not set.
func configGet(key string) string {
	return configData[key]
}

// configSet upserts a "section.key"=value into config.toml and refreshes the cache.
func configSet(key, value string) {
	_ = os.MkdirAll(configDirPath, 0755)
	if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
		_ = os.WriteFile(ConfigPath, []byte{}, 0600)
	}

	raw := map[string]map[string]string{}
	if data, err := os.ReadFile(ConfigPath); err == nil {
		toml.Unmarshal(data, &raw)
	}

	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return
	}
	section, field := parts[0], parts[1]
	if raw[section] == nil {
		raw[section] = map[string]string{}
	}
	raw[section][field] = value

	writeToml(raw)
	refreshConfig()
}

// writeToml marshals a map[string]map[string]string to config.toml.
func writeToml(raw map[string]map[string]string) {
	_ = os.MkdirAll(configDirPath, 0755)
	data, err := toml.Marshal(raw)
	if err != nil {
		return
	}
	_ = os.WriteFile(ConfigPath, data, 0600)
}

// refreshConfig reloads configData from disk (used by setup wizard after writes).
func refreshConfig() {
	configData = loadConfig()
}
