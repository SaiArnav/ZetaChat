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

	// DiscordToken is a Discord bot token (from discord.com/developers).
	// Empty means the Discord platform is disabled.
	DiscordToken string

	// WhatsAppEnabled can be set to "0"/"false" to skip the WhatsApp
	// adapter. WhatsApp needs no credentials — it links by QR code.
	WhatsAppEnabled bool

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

	telegramConfigured := appID != 0 && appHash != ""

	switch strings.ToLower(envStr("WHATSAPP_ENABLED")) {
	case "0", "false", "no", "off":
		cfg.WhatsAppEnabled = false
	default:
		cfg.WhatsAppEnabled = true
	}

	if !telegramConfigured && !cfg.WhatsAppEnabled {
		return nil, fmt.Errorf(
			"no platform enabled.\n\n" +
				"Configure at least one platform in .env:\n\n" +
				"Telegram (https://my.telegram.org/apps):\n" +
				"   TELEGRAM_API_ID=your_api_id\n" +
				"   TELEGRAM_API_HASH=your_api_hash\n\n" +
				"WhatsApp needs no credentials (QR pairing) — it is on by\n" +
				"default; disable with WHATSAPP_ENABLED=0.\n",
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
