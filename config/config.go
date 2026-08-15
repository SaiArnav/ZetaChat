package config

import (
	"os"
	"path/filepath"
)

// Config holds ZetaChat's runtime settings, loaded from
// ~/.config/zetachat/config.yaml (or similar) once implemented.
type Config struct {
	DataDir        string // where sqlite cache + sessions live
	EnabledPlugins []string
}

// Default returns a sane default config, creating the data dir if needed.
func Default() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dataDir := filepath.Join(home, ".zetachat")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	return &Config{
		DataDir:        dataDir,
		EnabledPlugins: []string{"telegram"},
	}, nil
}
