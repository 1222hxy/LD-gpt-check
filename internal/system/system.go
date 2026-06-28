package system

import (
	"bufio"
	"bytes"
	"context"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type CodexConfig struct {
	Model         string
	ModelProvider string
	ProviderHost  string
}

func CodexPath() (string, error) {
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("codex.cmd"); err == nil {
			return p, nil
		}
	}
	return exec.LookPath("codex")
}

func CodexVersion() string {
	path, err := CodexPath()
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}

func CodexConfigPath() string {
	if v := strings.TrimSpace(os.Getenv("CODEX_HOME")); v != "" {
		return filepath.Join(v, "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codex", "config.toml")
}

func CodexConfiguredModel() (string, error) {
	info, err := CodexConfigInfo()
	if err != nil {
		return "", err
	}
	if !ConcreteCodexModel(info.Model) {
		return "", nil
	}
	return info.Model, nil
}

func CodexConfigInfo() (CodexConfig, error) {
	path := CodexConfigPath()
	if path == "" {
		return CodexConfig{}, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return CodexConfig{}, nil
		}
		return CodexConfig{}, err
	}
	values, err := parseSimpleTOMLStrings(b)
	if err != nil {
		return CodexConfig{}, err
	}
	model := strings.TrimSpace(values["model"])
	provider := strings.TrimSpace(values["model_provider"])
	if provider == "" {
		provider = strings.TrimSpace(values["provider"])
	}
	host := ""
	if provider != "" {
		host = providerHost(values, provider)
	}
	return CodexConfig{Model: model, ModelProvider: provider, ProviderHost: host}, nil
}

func ConcreteCodexModel(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "", "default", "codex-default", "local-codex-config", "codex-local-config", "unknown", "unknown-codex-model":
		return false
	default:
		return true
	}
}

func rootTOMLString(b []byte, want string) (string, error) {
	values, err := parseSimpleTOMLStrings(b)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(values[want]), nil
}

func parseSimpleTOMLStrings(b []byte) (map[string]string, error) {
	values := make(map[string]string)
	section := ""
	scanner := bufio.NewScanner(strings.NewReader(string(b)))
	for scanner.Scan() {
		line := stripTOMLComment(strings.TrimSpace(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		s, ok := parseTOMLStringValue(strings.TrimSpace(value))
		if !ok {
			continue
		}
		normalizedKey := strings.TrimSpace(key)
		if section != "" {
			normalizedKey = section + "." + normalizedKey
		}
		values[normalizedKey] = strings.TrimSpace(s)
	}
	return values, scanner.Err()
}

func parseTOMLStringValue(value string) (string, bool) {
	if value == "" {
		return "", false
	}
	if strings.HasPrefix(value, `"`) {
		s, err := strconv.Unquote(value)
		if err != nil {
			return "", false
		}
		return strings.TrimSpace(s), true
	}
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") && len(value) >= 2 {
		return strings.TrimSpace(strings.Trim(value, "'")), true
	}
	return "", false
}

func providerHost(values map[string]string, provider string) string {
	for _, key := range providerBaseURLKeys(provider) {
		if host := hostFromURL(values[key]); host != "" {
			return host
		}
	}
	return ""
}

func providerBaseURLKeys(provider string) []string {
	provider = strings.Trim(strings.TrimSpace(provider), `"`)
	return []string{
		"model_providers." + provider + ".base_url",
		"model_providers.\"" + provider + "\".base_url",
		"providers." + provider + ".base_url",
		"providers.\"" + provider + "\".base_url",
	}
}

func hostFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Host
}

func stripTOMLComment(line string) string {
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

func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
