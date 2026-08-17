package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds ZetaChat's runtime settings.
type Config struct {
	// AppID / AppHash are Telegram API credentials (from my.telegram.org).
	AppID   int
	AppHash string

	// DataDir is where the SQLite cache + sessions live.
	DataDir string

	// SessionPath is where the Telegram session file is stored.
	SessionPath string

	EnabledPlugins []string
}

// Default returns a config with sane defaults, creating dirs if needed.
func Default() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dataDir := filepath.Join(home, ".zetachat")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	sessionDir := filepath.Join(dataDir, "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, err
	}
	return &Config{
		DataDir:        dataDir,
		SessionPath:    filepath.Join(sessionDir, "telegram.json"),
		EnabledPlugins: []string{"telegram"},
	}, nil
}

// Load reads .env (if present) for TELEGRAM_API_ID / TELEGRAM_API_HASH,
// then falls back to environment variables APP_ID / APP_HASH for
// backward compatibility with the original .env format.
func Load() (*Config, error) {
	cfg, err := Default()
	if err != nil {
		return nil, err
	}

	env := findDotEnv()
	mergeEnv(env)

	appID, err := envInt("TELEGRAM_API_ID", "APP_ID")
	if err != nil {
		return nil, err
	}
	appHash := envStr("TELEGRAM_API_HASH", "APP_HASH")
	if appID == 0 || appHash == "" {
		return nil, fmt.Errorf(
			"telegram credentials missing.\n\n" +
				"Setup steps:\n" +
				"1. Go to https://my.telegram.org/apps\n" +
				"2. Sign in with your Telegram account.\n" +
				"3. Create an application and copy the API ID and API Hash.\n" +
				"4. Create a .env file in the ZetaChat project folder.\n" +
				"5. Add:\n\n" +
				"   TELEGRAM_API_ID=your_api_id\n" +
				"   TELEGRAM_API_HASH=your_api_hash\n",
		)
	}

	cfg.AppID = appID
	cfg.AppHash = appHash
	return cfg, nil
}

// findDotEnv locates a .env file: current directory first, then next to the
// executable, then in ~/.zetachat.
func findDotEnv() map[string]string {
	var candidates []string
	if pwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(pwd, ".env"))
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), ".env"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".zetachat", ".env"))
	}
	for _, p := range candidates {
		if m := readDotEnv(p); len(m) > 0 {
			return m
		}
	}
	return map[string]string{}
}

func readDotEnv(path string) map[string]string {
	env := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return env
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		env[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	return env
}

// mergeEnv folds .env values into the real environment only when the real
// environment variable is not already set (dotenv never overrides).
func mergeEnv(fromDotEnv map[string]string) {
	for k, v := range fromDotEnv {
		if _, ok := os.LookupEnv(k); !ok {
			os.Setenv(k, v)
		}
	}
}

func envInt(keys ...string) (int, error) {
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok && v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				return 0, fmt.Errorf("invalid %s: %q", k, v)
			}
			return n, nil
		}
	}
	return 0, nil
}

func envStr(keys ...string) string {
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok && v != "" {
			return v
		}
	}
	return ""
}
