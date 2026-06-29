package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/1222hxy/LD-gpt-check/internal/i18n"
)

const (
	AppName          = "ld-gpt-check"
	DefaultAPIBase   = "https://codexgo.yhklab.com"
	ConfigFileName   = "ld-gpt-check.toml"
	ConfigEnvVarName = "LD_GPT_CHECK_CONFIG"
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type DeviceAuthorization struct {
	Secret string `json:"secret"`
}

type Config struct {
	APIBaseURL          string              `json:"api_base_url"`
	AccessToken         string              `json:"access_token"`
	Language            string              `json:"language"`
	User                User                `json:"user"`
	DeviceAuthorization DeviceAuthorization `json:"device_authorization"`
}

func DefaultAPIBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("LD_GPT_CHECK_API_BASE_URL")); v != "" {
		return v
	}
	return DefaultAPIBase
}

func Path() (string, error) {
	if v := strings.TrimSpace(os.Getenv(ConfigEnvVarName)); v != "" {
		if filepath.IsAbs(v) {
			return v, nil
		}
		return filepath.Abs(v)
	}

	exe, err := os.Executable()
	if err != nil {
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			return "", err
		}
		return filepath.Join(wd, ConfigFileName), nil
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Join(filepath.Dir(exe), ConfigFileName), nil
}

func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return normalize(Config{APIBaseURL: DefaultAPIBaseURL(), Language: string(i18n.ZH)}), nil
		}
		return Config{}, err
	}
	cfg, err := parseConfig(b)
	if err != nil {
		return Config{}, err
	}
	return normalize(cfg), nil
}

func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	cfg = normalize(cfg)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(marshalTOML(cfg)); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	if err := os.Chmod(path, 0600); err != nil {
		return err
	}
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

func DeleteToken() error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg.AccessToken = ""
	cfg.DeviceAuthorization = DeviceAuthorization{}
	cfg.User = User{}
	return Save(cfg)
}

func normalize(cfg Config) Config {
	if strings.TrimSpace(cfg.APIBaseURL) == "" {
		cfg.APIBaseURL = DefaultAPIBaseURL()
	} else {
		cfg.APIBaseURL = strings.TrimSpace(cfg.APIBaseURL)
	}
	cfg.Language = string(i18n.Normalize(cfg.Language))
	if strings.TrimSpace(cfg.DeviceAuthorization.Secret) == "" {
		cfg.DeviceAuthorization.Secret = strings.TrimSpace(cfg.AccessToken)
	} else {
		cfg.DeviceAuthorization.Secret = strings.TrimSpace(cfg.DeviceAuthorization.Secret)
	}
	cfg.AccessToken = cfg.DeviceAuthorization.Secret
	return cfg
}

func parseConfig(b []byte) (Config, error) {
	if strings.HasPrefix(strings.TrimSpace(string(b)), "{") {
		var cfg Config
		if err := json.Unmarshal(b, &cfg); err != nil {
			return Config{}, err
		}
		return cfg, nil
	}
	return parseTOML(b)
}

func parseTOML(b []byte) (Config, error) {
	var cfg Config
	section := ""
	scanner := bufio.NewScanner(strings.NewReader(string(b)))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := stripComment(strings.TrimSpace(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			if section != "user" && section != "device_authorization" {
				return Config{}, fmt.Errorf("invalid config line %d: unknown section [%s]", lineNo, section)
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return Config{}, fmt.Errorf("invalid config line %d: expected key = \"value\"", lineNo)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		s, err := strconv.Unquote(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid config line %d: values must be quoted strings", lineNo)
		}
		switch section + "." + key {
		case ".api_base_url":
			cfg.APIBaseURL = s
		case ".access_token":
			cfg.AccessToken = s
		case ".language":
			cfg.Language = s
		case "device_authorization.secret":
			cfg.DeviceAuthorization.Secret = s
		case "user.id":
			cfg.User.ID = s
		case "user.username":
			cfg.User.Username = s
		default:
			if section == "" {
				return Config{}, fmt.Errorf("invalid config line %d: unknown key %q", lineNo, key)
			}
			return Config{}, fmt.Errorf("invalid config line %d: unknown key %q in section [%s]", lineNo, key, section)
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func marshalTOML(cfg Config) string {
	var b strings.Builder
	b.WriteString("# LD-gpt-check local config. Keep this file private.\n")
	b.WriteString("api_base_url = " + strconv.Quote(cfg.APIBaseURL) + "\n")
	b.WriteString("language = " + strconv.Quote(cfg.Language) + "\n\n")
	b.WriteString("[device_authorization]\n")
	b.WriteString("secret = " + strconv.Quote(cfg.DeviceAuthorization.Secret) + "\n")
	return b.String()
}

func stripComment(line string) string {
	inString := false
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inString {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if r == '#' && !inString {
			return strings.TrimSpace(line[:i])
		}
	}
	return line
}
