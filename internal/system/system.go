package system

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
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
	Model           string
	ModelProvider   string
	ProviderHost    string
	ProviderBaseURL string
}

type CCSwitchResolution struct {
	LocalBaseURL    string
	ProviderBaseURL string
	ProviderHost    string
	ConfigDir       string
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

func UploadCodexVersion(codexSandbox string) string {
	if strings.TrimSpace(codexSandbox) == "api" {
		return "api"
	}
	return CodexVersion()
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
			return CodexConfig{ProviderHost: "api.openai.com", ProviderBaseURL: "https://api.openai.com/v1"}, nil
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
	baseURL := ""
	rawBaseURL := ""
	if provider != "" {
		rawBaseURL = providerBaseURLRaw(values, provider)
		baseURL = providerBaseURL(values, provider)
	}
	localPrivateBaseURL := IsPrivateProviderBaseURL(rawBaseURL) || IsPrivateProviderBaseURL(baseURL)
	if CCSwitchAutoResolveEnabled() && localPrivateBaseURL {
		if resolved := CCSwitchCodexProviderBaseURL(); resolved != "" {
			baseURL = resolved
		}
	}
	if baseURL == "" && !localPrivateBaseURL {
		baseURL = "https://api.openai.com/v1"
	}
	return CodexConfig{Model: model, ModelProvider: provider, ProviderHost: hostFromURL(baseURL), ProviderBaseURL: baseURL}, nil
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
	return hostFromURL(providerBaseURL(values, provider))
}

func providerBaseURL(values map[string]string, provider string) string {
	return NormalizeProviderBaseURL(providerBaseURLRaw(values, provider))
}

func providerBaseURLRaw(values map[string]string, provider string) string {
	for _, key := range providerBaseURLKeys(provider) {
		if baseURL := strings.TrimSpace(values[key]); baseURL != "" {
			return baseURL
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
	raw = NormalizeProviderBaseURL(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Host
}

func NormalizeProviderBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "https" {
		return ""
	}
	u.Host = strings.ToLower(u.Host)
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	u.Path = strings.TrimRight(u.Path, "/")
	if u.Path == "" {
		u.Path = ""
	}
	return u.String()
}

func IsPrivateProviderBaseURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return false
	}
	return privateHostname(u.Hostname())
}

func CCSwitchAutoResolveEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("LD_GPT_CHECK_DISABLE_CC_SWITCH")))
	return v != "1" && v != "true" && v != "yes"
}

func DetectCCSwitchCodexResolution() CCSwitchResolution {
	path := CodexConfigPath()
	if path == "" {
		return CCSwitchResolution{}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return CCSwitchResolution{}
	}
	values, err := parseSimpleTOMLStrings(b)
	if err != nil {
		return CCSwitchResolution{}
	}
	provider := strings.TrimSpace(values["model_provider"])
	if provider == "" {
		provider = strings.TrimSpace(values["provider"])
	}
	if provider == "" {
		return CCSwitchResolution{}
	}
	localBaseURL := providerBaseURLRaw(values, provider)
	if !IsPrivateProviderBaseURL(localBaseURL) {
		return CCSwitchResolution{}
	}
	resolved := CCSwitchCodexProviderBaseURL()
	if resolved == "" {
		return CCSwitchResolution{LocalBaseURL: localBaseURL, ConfigDir: CCSwitchConfigDir()}
	}
	return CCSwitchResolution{
		LocalBaseURL:    localBaseURL,
		ProviderBaseURL: resolved,
		ProviderHost:    hostFromURL(resolved),
		ConfigDir:       CCSwitchConfigDir(),
	}
}

func CCSwitchConfigDir() string {
	for _, key := range []string{"LD_GPT_CHECK_CC_SWITCH_DIR", "CC_SWITCH_CONFIG_DIR", "CC_SWITCH_HOME"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cc-switch")
}

func CCSwitchCodexProviderBaseURL() string {
	dir := CCSwitchConfigDir()
	if dir == "" {
		return ""
	}
	currentID := ccSwitchCurrentCodexProviderID(filepath.Join(dir, "config.json"))
	if baseURL := ccSwitchCodexProviderBaseURLFromSQLite(filepath.Join(dir, "cc-switch.db"), currentID); baseURL != "" {
		return baseURL
	}
	return ccSwitchCodexProviderBaseURLFromFiles(dir, currentID)
}

func ccSwitchCurrentCodexProviderID(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var data any
	if err := json.Unmarshal(b, &data); err != nil {
		return ""
	}
	for _, key := range []string{"currentProviderCodex", "current_provider_codex"} {
		if v := jsonStringAtPath(data, key); v != "" {
			return v
		}
	}
	return ""
}

func ccSwitchCodexProviderBaseURLFromSQLite(dbPath, currentID string) string {
	if _, err := os.Stat(dbPath); err != nil {
		return ""
	}
	sqlitePath, err := exec.LookPath("sqlite3")
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	query := `SELECT id, settings_config, is_current FROM providers WHERE app_type = 'codex'`
	cmd := exec.CommandContext(ctx, sqlitePath, "-readonly", "-json", dbPath, query)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	var rows []struct {
		ID             string `json:"id"`
		SettingsConfig string `json:"settings_config"`
		IsCurrent      int    `json:"is_current"`
	}
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		return ""
	}
	for _, row := range rows {
		if currentID != "" && row.ID == currentID {
			if baseURL := codexBaseURLFromCCSwitchSettings(row.SettingsConfig); baseURL != "" {
				return baseURL
			}
		}
	}
	for _, row := range rows {
		if row.IsCurrent != 0 {
			if baseURL := codexBaseURLFromCCSwitchSettings(row.SettingsConfig); baseURL != "" {
				return baseURL
			}
		}
	}
	for _, row := range rows {
		if baseURL := codexBaseURLFromCCSwitchSettings(row.SettingsConfig); baseURL != "" {
			return baseURL
		}
	}
	return ""
}

func ccSwitchCodexProviderBaseURLFromFiles(dir, currentID string) string {
	var paths []string
	for _, name := range []string{"config.json", "providers.json"} {
		paths = append(paths, filepath.Join(dir, name))
	}
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := strings.ToLower(entry.Name())
			if strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".toml") {
				paths = append(paths, filepath.Join(dir, entry.Name()))
			}
		}
	}
	seen := map[string]struct{}{}
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		b, err := os.ReadFile(path)
		if err != nil || len(b) > 4*1024*1024 {
			continue
		}
		if baseURL := codexBaseURLFromCCSwitchBlob(b, currentID); baseURL != "" {
			return baseURL
		}
	}
	return ""
}

func codexBaseURLFromCCSwitchBlob(b []byte, currentID string) string {
	var data any
	if json.Unmarshal(b, &data) == nil {
		if baseURL := codexBaseURLFromCCSwitchJSON(data, currentID); baseURL != "" {
			return baseURL
		}
	}
	return publicProviderBaseURLFromText(string(b))
}

func codexBaseURLFromCCSwitchSettings(raw string) string {
	var data any
	if err := json.Unmarshal([]byte(raw), &data); err == nil {
		if configText := jsonStringAtPath(data, "config"); configText != "" {
			if baseURL := publicProviderBaseURLFromCodexConfig(configText); baseURL != "" {
				return baseURL
			}
		}
		if baseURL := codexBaseURLFromCCSwitchJSON(data, ""); baseURL != "" {
			return baseURL
		}
	}
	return publicProviderBaseURLFromText(raw)
}

func codexBaseURLFromCCSwitchJSON(v any, currentID string) string {
	if currentID != "" {
		if obj, ok := v.(map[string]any); ok {
			if provider := findJSONObjectByID(obj, currentID); provider != nil {
				if baseURL := codexBaseURLFromCCSwitchJSON(provider, ""); baseURL != "" {
					return baseURL
				}
			}
		}
	}
	if configText := jsonStringAtPath(v, "config"); configText != "" {
		if baseURL := publicProviderBaseURLFromCodexConfig(configText); baseURL != "" {
			return baseURL
		}
	}
	for _, key := range []string{"base_url", "baseUrl", "api_base_url", "apiBaseUrl", "url"} {
		if baseURL := normalizePublicProviderBaseURL(jsonStringAtPath(v, key)); baseURL != "" {
			return baseURL
		}
	}
	switch x := v.(type) {
	case map[string]any:
		for _, value := range x {
			if baseURL := codexBaseURLFromCCSwitchJSON(value, ""); baseURL != "" {
				return baseURL
			}
		}
	case []any:
		for _, value := range x {
			if baseURL := codexBaseURLFromCCSwitchJSON(value, ""); baseURL != "" {
				return baseURL
			}
		}
	}
	return ""
}

func findJSONObjectByID(obj map[string]any, id string) map[string]any {
	for key, value := range obj {
		if key == id {
			if nested, ok := value.(map[string]any); ok {
				return nested
			}
		}
		if nested, ok := value.(map[string]any); ok {
			if jsonStringAtPath(nested, "id") == id {
				return nested
			}
			if found := findJSONObjectByID(nested, id); found != nil {
				return found
			}
		}
		if arr, ok := value.([]any); ok {
			for _, item := range arr {
				if nested, ok := item.(map[string]any); ok {
					if jsonStringAtPath(nested, "id") == id {
						return nested
					}
					if found := findJSONObjectByID(nested, id); found != nil {
						return found
					}
				}
			}
		}
	}
	return nil
}

func jsonStringAtPath(v any, key string) string {
	switch x := v.(type) {
	case map[string]any:
		if value, ok := x[key]; ok {
			if s, ok := value.(string); ok {
				return strings.TrimSpace(s)
			}
		}
		for _, value := range x {
			if s := jsonStringAtPath(value, key); s != "" {
				return s
			}
		}
	case []any:
		for _, value := range x {
			if s := jsonStringAtPath(value, key); s != "" {
				return s
			}
		}
	}
	return ""
}

func publicProviderBaseURLFromCodexConfig(configText string) string {
	values, err := parseSimpleTOMLStrings([]byte(configText))
	if err != nil {
		return publicProviderBaseURLFromText(configText)
	}
	provider := strings.TrimSpace(values["model_provider"])
	if provider == "" {
		provider = strings.TrimSpace(values["provider"])
	}
	if provider != "" {
		if baseURL := normalizePublicProviderBaseURL(providerBaseURLRaw(values, provider)); baseURL != "" {
			return baseURL
		}
	}
	return normalizePublicProviderBaseURL(values["base_url"])
}

func publicProviderBaseURLFromText(text string) string {
	for _, line := range strings.Split(text, "\n") {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "base_url") && !strings.Contains(lower, "baseurl") && !strings.Contains(lower, "api_base") && !strings.Contains(lower, "apibase") {
			continue
		}
		for _, raw := range extractHTTPSURLs(line) {
			if baseURL := normalizePublicProviderBaseURL(raw); baseURL != "" {
				return baseURL
			}
		}
	}
	for _, raw := range extractHTTPSURLs(text) {
		if baseURL := normalizePublicProviderBaseURL(raw); baseURL != "" {
			return baseURL
		}
	}
	return ""
}

func extractHTTPSURLs(text string) []string {
	var urls []string
	for _, field := range strings.FieldsFunc(text, func(r rune) bool {
		return r <= ' ' || strings.ContainsRune(`"'<>[]{}(),`, r)
	}) {
		field = strings.TrimSpace(field)
		if strings.HasPrefix(strings.ToLower(field), "https://") {
			urls = append(urls, field)
		}
	}
	return urls
}

func normalizePublicProviderBaseURL(raw string) string {
	baseURL := NormalizeProviderBaseURL(raw)
	if baseURL == "" || IsPrivateProviderBaseURL(baseURL) {
		return ""
	}
	return baseURL
}

func privateHostname(hostname string) bool {
	h := strings.ToLower(strings.Trim(hostname, "[]"))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") || strings.HasSuffix(h, ".local") {
		return true
	}
	if parts := strings.Split(h, "."); len(parts) == 4 {
		nums := make([]int, 4)
		for i, part := range parts {
			n, err := strconv.Atoi(part)
			if err != nil || n < 0 || n > 255 {
				return false
			}
			nums[i] = n
		}
		return nums[0] == 10 ||
			nums[0] == 127 ||
			(nums[0] == 169 && nums[1] == 254) ||
			(nums[0] == 172 && nums[1] >= 16 && nums[1] <= 31) ||
			(nums[0] == 192 && nums[1] == 168) ||
			nums[0] == 0 ||
			nums[0] >= 224
	}
	return h == "::1" || strings.HasPrefix(h, "fc") || strings.HasPrefix(h, "fd") || strings.HasPrefix(h, "fe80:")
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
