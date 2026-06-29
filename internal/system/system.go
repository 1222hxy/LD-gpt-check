package system

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type CodexConfig struct {
	Model           string
	ModelProvider   string
	ProviderHost    string
	ProviderBaseURL string
	ConfigPath      string
}

type CCSwitchResolution struct {
	LocalBaseURL    string
	ProviderBaseURL string
	ProviderHost    string
	ConfigDir       string
	ModelAPIKey     string
	APIFormat       string
}

type CCSwitchProviderConfig struct {
	BaseURL   string
	APIKey    string
	APIFormat string
	ConfigDir string
}

type ccSwitchConfigLocation struct {
	Dir         string
	DBPaths     []string
	ConfigPaths []string
}

func CodexPath() (string, error) {
	for _, name := range codexExecutableNames() {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	for _, p := range candidateCodexExecutablePaths() {
		if executableExists(p) {
			return p, nil
		}
	}
	return exec.LookPath("codex")
}

func codexExecutableNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"codex.cmd", "codex.exe", "codex"}
	}
	return []string{"codex"}
}

func candidateCodexExecutablePaths() []string {
	home, _ := os.UserHomeDir()
	var out []string
	switch runtime.GOOS {
	case "darwin":
		if home != "" {
			out = append(out,
				filepath.Join(home, ".local", "bin", "codex"),
				filepath.Join(home, "bin", "codex"),
			)
		}
		out = append(out,
			"/opt/homebrew/bin/codex",
			"/usr/local/bin/codex",
			"/Applications/Codex.app/Contents/MacOS/codex",
			"/Applications/Codex.app/Contents/MacOS/Codex",
		)
		if home != "" {
			out = append(out,
				filepath.Join(home, "Applications", "Codex.app", "Contents", "MacOS", "codex"),
				filepath.Join(home, "Applications", "Codex.app", "Contents", "MacOS", "Codex"),
			)
		}
	case "linux":
		out = append(out, "/usr/local/bin/codex", "/usr/bin/codex")
	case "windows":
		for _, env := range []string{"APPDATA", "LOCALAPPDATA"} {
			base := strings.TrimSpace(os.Getenv(env))
			if base == "" {
				continue
			}
			out = append(out,
				filepath.Join(base, "Codex", "codex.cmd"),
				filepath.Join(base, "Codex", "codex.exe"),
				filepath.Join(base, "OpenAI", "Codex", "codex.cmd"),
				filepath.Join(base, "OpenAI", "Codex", "codex.exe"),
			)
		}
	}
	return out
}

func executableExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0111 != 0
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
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
	switch strings.TrimSpace(codexSandbox) {
	case "api":
		return "api"
	case "auth-json":
		return "auth-json"
	}
	return CodexVersion()
}

func CodexConfigPath() string {
	for _, path := range CandidateCodexConfigPaths() {
		if fileExists(path) {
			return path
		}
	}
	candidates := CandidateCodexConfigPaths()
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}

func CandidateCodexConfigPaths() []string {
	if v := strings.TrimSpace(os.Getenv("CODEX_HOME")); v != "" {
		return []string{filepath.Join(v, "config.toml")}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	out := codexDefaultConfigPaths(home, runtime.GOOS)
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" && runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		out = append(out, filepath.Join(xdg, "Codex", "config.toml"))
	}
	seen := map[string]bool{}
	deduped := make([]string, 0, len(out))
	for _, path := range out {
		key := filepath.Clean(path)
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, path)
	}
	return deduped
}

func codexDefaultConfigPaths(home, goos string) []string {
	if home == "" {
		return nil
	}
	out := []string{filepath.Join(home, ".codex", "config.toml")}
	switch goos {
	case "darwin":
		out = append(out,
			filepath.Join(home, "Library", "Application Support", "Codex", "config.toml"),
			filepath.Join(home, "Library", "Application Support", "OpenAI", "Codex", "config.toml"),
			filepath.Join(home, "Library", "Application Support", "com.openai.codex", "config.toml"),
		)
	case "windows":
		for _, env := range []string{"APPDATA", "LOCALAPPDATA"} {
			base := strings.TrimSpace(os.Getenv(env))
			if base == "" {
				continue
			}
			out = append(out,
				filepath.Join(base, "Codex", "config.toml"),
				filepath.Join(base, "OpenAI", "Codex", "config.toml"),
				filepath.Join(base, "com.openai.codex", "config.toml"),
			)
		}
	default:
		out = append(out,
			filepath.Join(home, ".config", "Codex", "config.toml"),
			filepath.Join(home, ".config", "OpenAI", "Codex", "config.toml"),
			filepath.Join(home, ".config", "openai-codex", "config.toml"),
			filepath.Join(home, ".config", "com.openai.codex", "config.toml"),
		)
	}
	return out
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
			return CodexConfig{ConfigPath: path, ProviderHost: "api.openai.com", ProviderBaseURL: "https://api.openai.com/v1"}, nil
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
	return CodexConfig{Model: model, ModelProvider: provider, ProviderHost: hostFromURL(baseURL), ProviderBaseURL: baseURL, ConfigPath: path}, nil
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
	u.RawPath = ""
	u.Path = strings.TrimRight(u.Path, `/\`)
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
	providerConfig := CCSwitchCodexProviderConfig()
	resolved := providerConfig.BaseURL
	configDir := firstNonEmptyString(providerConfig.ConfigDir, CCSwitchConfigDir())
	localBaseURL := ""
	path := CodexConfigPath()
	if path != "" {
		if b, err := os.ReadFile(path); err == nil {
			if values, err := parseSimpleTOMLStrings(b); err == nil {
				provider := strings.TrimSpace(values["model_provider"])
				if provider == "" {
					provider = strings.TrimSpace(values["provider"])
				}
				if provider != "" {
					if raw := providerBaseURLRaw(values, provider); IsPrivateProviderBaseURL(raw) {
						localBaseURL = raw
					}
				}
			}
		}
	}
	if resolved == "" {
		if localBaseURL == "" {
			if !ccSwitchConfigDirExists(configDir) {
				return CCSwitchResolution{}
			}
		}
		return CCSwitchResolution{LocalBaseURL: localBaseURL, ConfigDir: configDir}
	}
	return CCSwitchResolution{
		LocalBaseURL:    localBaseURL,
		ProviderBaseURL: resolved,
		ProviderHost:    hostFromURL(resolved),
		ConfigDir:       configDir,
		ModelAPIKey:     providerConfig.APIKey,
		APIFormat:       providerConfig.APIFormat,
	}
}

func CCSwitchConfigDir() string {
	locations := ccSwitchConfigLocations()
	for _, loc := range locations {
		if ccSwitchConfigLocationExists(loc) {
			return loc.Dir
		}
	}
	if len(locations) > 0 {
		return locations[0].Dir
	}
	return ""
}

func ccSwitchConfigLocations() []ccSwitchConfigLocation {
	var locations []ccSwitchConfigLocation
	for _, key := range []string{"LD_GPT_CHECK_CC_SWITCH_DIR", "CC_SWITCH_CONFIG_DIR", "CC_SWITCH_HOME"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			addCCSwitchConfigLocation(&locations, v)
		}
	}
	if v := strings.TrimSpace(os.Getenv("LD_GPT_CHECK_CC_SWITCH_DB")); v != "" {
		addCCSwitchConfigLocation(&locations, v)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return locations
	}
	for _, path := range ccSwitchDefaultConfigDirs(home, runtime.GOOS) {
		addCCSwitchConfigLocation(&locations, path)
	}
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		addCCSwitchConfigLocation(&locations, filepath.Join(xdg, "cc-switch"))
	}
	addCCSwitchConfigLocation(&locations, filepath.Join(home, ".config", "cc-switch"))
	return locations
}

func ccSwitchDefaultConfigDirs(home, goos string) []string {
	if home == "" {
		return nil
	}
	dirs := []string{filepath.Join(home, ".cc-switch")}
	if goos == "darwin" {
		appSupport := filepath.Join(home, "Library", "Application Support")
		for _, name := range []string{"cc-switch", "CC Switch", "CCSwitch", "ccswitch", "com.cc-switch", "com.ccswitch", "app.cc-switch"} {
			dirs = append(dirs, filepath.Join(appSupport, name))
		}
		for _, pattern := range []string{
			filepath.Join(home, "Library", "Containers", "*cc-switch*", "Data", "Library", "Application Support", "*"),
			filepath.Join(home, "Library", "Containers", "*ccswitch*", "Data", "Library", "Application Support", "*"),
			filepath.Join(home, "Library", "Containers", "*CC Switch*", "Data", "Library", "Application Support", "*"),
		} {
			matches, _ := filepath.Glob(pattern)
			for _, match := range matches {
				dirs = append(dirs, match)
			}
		}
	}
	return dirs
}

func addCCSwitchConfigLocation(locations *[]ccSwitchConfigLocation, raw string) {
	path := expandHomePath(strings.TrimSpace(raw))
	if path == "" {
		return
	}
	path = filepath.Clean(path)
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".db") || strings.HasSuffix(lower, ".sqlite") || strings.HasSuffix(lower, ".sqlite3") {
		addCCSwitchConfigLocationValue(locations, ccSwitchConfigLocation{
			Dir:         filepath.Dir(path),
			DBPaths:     []string{path},
			ConfigPaths: ccSwitchConfigPaths(filepath.Dir(path)),
		})
		return
	}
	addCCSwitchConfigLocationValue(locations, ccSwitchConfigLocation{
		Dir:         path,
		DBPaths:     ccSwitchDBPaths(path),
		ConfigPaths: ccSwitchConfigPaths(path),
	})
}

func addCCSwitchConfigLocationValue(locations *[]ccSwitchConfigLocation, loc ccSwitchConfigLocation) {
	if loc.Dir == "" {
		return
	}
	for _, existing := range *locations {
		if existing.Dir == loc.Dir {
			return
		}
	}
	*locations = append(*locations, loc)
}

func ccSwitchDBPaths(dir string) []string {
	names := []string{"cc-switch.db", "cc_switch.db", "ccswitch.db", "database.db", "data.db", "app.db", "db.sqlite", "database.sqlite", "cc-switch.sqlite", "ccswitch.sqlite"}
	paths := make([]string, 0, len(names))
	for _, name := range names {
		paths = append(paths, filepath.Join(dir, name))
	}
	return paths
}

func ccSwitchConfigPaths(dir string) []string {
	names := []string{"config.json", "providers.json", "settings.json", "cc-switch.json"}
	paths := make([]string, 0, len(names))
	for _, name := range names {
		paths = append(paths, filepath.Join(dir, name))
	}
	return paths
}

func expandHomePath(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func ccSwitchConfigDirExists(dir string) bool {
	if dir == "" {
		return false
	}
	return ccSwitchConfigLocationExists(ccSwitchConfigLocation{Dir: dir, DBPaths: ccSwitchDBPaths(dir), ConfigPaths: ccSwitchConfigPaths(dir)})
}

func ccSwitchConfigLocationExists(loc ccSwitchConfigLocation) bool {
	if loc.Dir != "" {
		if _, err := os.Stat(loc.Dir); err == nil {
			return true
		}
	}
	for _, path := range append(append([]string{}, loc.DBPaths...), loc.ConfigPaths...) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func CCSwitchCodexProviderBaseURL() string {
	return CCSwitchCodexProviderConfig().BaseURL
}

func CCSwitchCodexProviderConfig() CCSwitchProviderConfig {
	for _, loc := range ccSwitchConfigLocations() {
		if !ccSwitchConfigLocationExists(loc) {
			continue
		}
		if cfg := ccSwitchCodexProviderConfigFromLocation(loc); !cfg.empty() {
			return cfg
		}
	}
	return CCSwitchProviderConfig{}
}

func (c CCSwitchProviderConfig) empty() bool {
	return c.BaseURL == "" && c.APIKey == "" && c.APIFormat == ""
}

func ccSwitchCodexProviderConfigFromLocation(loc ccSwitchConfigLocation) CCSwitchProviderConfig {
	currentID := ccSwitchCurrentCodexProviderIDFromLocation(loc)
	for _, dbPath := range loc.DBPaths {
		if cfg := ccSwitchCodexProviderConfigFromSQLite(dbPath, currentID); !cfg.empty() {
			cfg.ConfigDir = loc.Dir
			return cfg
		}
	}
	for _, dbPath := range loc.DBPaths {
		if cfg := ccSwitchCodexProviderConfigFromDBText(dbPath, currentID); !cfg.empty() {
			cfg.ConfigDir = loc.Dir
			return cfg
		}
	}
	if cfg := ccSwitchCodexProviderConfigFromFiles(loc.Dir, currentID); !cfg.empty() {
		cfg.ConfigDir = loc.Dir
		return cfg
	}
	return CCSwitchProviderConfig{}
}

func ccSwitchCurrentCodexProviderIDFromLocation(loc ccSwitchConfigLocation) string {
	for _, path := range loc.ConfigPaths {
		if v := ccSwitchCurrentCodexProviderID(path); v != "" {
			return v
		}
	}
	for _, dbPath := range loc.DBPaths {
		if v := ccSwitchCurrentCodexProviderIDFromSQLite(dbPath); v != "" {
			return v
		}
	}
	return ""
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

func ccSwitchCurrentCodexProviderIDFromSQLite(dbPath string) string {
	if _, err := os.Stat(dbPath); err != nil {
		return ""
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return ""
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return ""
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		keyLower := strings.ToLower(key)
		if strings.Contains(keyLower, "codex") && strings.Contains(keyLower, "provider") {
			if v := strings.Trim(strings.TrimSpace(value), `"' ,`); v != "" && !strings.HasPrefix(v, "{") && !strings.HasPrefix(v, "[") {
				return v
			}
		}
		if v := ccSwitchCurrentCodexProviderIDFromText(key + "\n" + value); v != "" {
			return v
		}
	}
	return ""
}

func ccSwitchCurrentCodexProviderIDFromText(text string) string {
	var data any
	if json.Unmarshal([]byte(text), &data) == nil {
		for _, key := range []string{"currentProviderCodex", "current_provider_codex", "currentCodexProvider", "current_codex_provider", "currentCodexProviderID", "current_codex_provider_id"} {
			if v := jsonStringAtPath(data, key); v != "" {
				return v
			}
		}
	}
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "codex") || !strings.Contains(lower, "provider") {
		return ""
	}
	for _, line := range strings.Split(text, "\n") {
		lineLower := strings.ToLower(line)
		if !strings.Contains(lineLower, "codex") || !strings.Contains(lineLower, "provider") {
			continue
		}
		if _, value, ok := strings.Cut(line, "="); ok {
			return strings.Trim(strings.TrimSpace(value), `"' ,`)
		}
		if _, value, ok := strings.Cut(line, ":"); ok {
			return strings.Trim(strings.TrimSpace(value), `"' ,`)
		}
	}
	return ""
}

func ccSwitchCodexProviderBaseURLFromSQLite(dbPath, currentID string) string {
	return ccSwitchCodexProviderConfigFromSQLite(dbPath, currentID).BaseURL
}

func ccSwitchCodexProviderConfigFromSQLite(dbPath, currentID string) CCSwitchProviderConfig {
	if cfg := ccSwitchCodexProviderConfigFromSQLiteDB(dbPath, currentID); !cfg.empty() {
		return cfg
	}
	return ccSwitchCodexProviderConfigFromSQLiteCLI(dbPath, currentID)
}

func ccSwitchCodexProviderConfigFromSQLiteDB(dbPath, currentID string) CCSwitchProviderConfig {
	if _, err := os.Stat(dbPath); err != nil {
		return CCSwitchProviderConfig{}
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return CCSwitchProviderConfig{}
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, `SELECT p.id, p.settings_config, p.is_current, COALESCE(e.url, '') AS endpoint_url
		FROM providers p
		LEFT JOIN provider_endpoints e ON e.provider_id = p.id AND e.app_type = p.app_type
		WHERE p.app_type = 'codex'
		ORDER BY p.is_current DESC, p.sort_index, e.id`)
	if err != nil {
		return CCSwitchProviderConfig{}
	}
	defer rows.Close()
	type providerRow struct {
		id             string
		settingsConfig string
		isCurrent      int
		endpointURL    string
	}
	var all []providerRow
	for rows.Next() {
		var row providerRow
		if err := rows.Scan(&row.id, &row.settingsConfig, &row.isCurrent, &row.endpointURL); err != nil {
			return CCSwitchProviderConfig{}
		}
		all = append(all, row)
	}
	if len(all) == 0 {
		return CCSwitchProviderConfig{}
	}
	for _, row := range all {
		if currentID != "" && row.id == currentID {
			if cfg := ccSwitchProviderConfigFromSQLiteRow(row.settingsConfig, row.endpointURL); !cfg.empty() {
				return cfg
			}
		}
	}
	for _, row := range all {
		if row.isCurrent != 0 {
			if cfg := ccSwitchProviderConfigFromSQLiteRow(row.settingsConfig, row.endpointURL); !cfg.empty() {
				return cfg
			}
		}
	}
	for _, row := range all {
		if cfg := ccSwitchProviderConfigFromSQLiteRow(row.settingsConfig, row.endpointURL); !cfg.empty() {
			return cfg
		}
	}
	return CCSwitchProviderConfig{}
}

func ccSwitchCodexProviderConfigFromSQLiteCLI(dbPath, currentID string) CCSwitchProviderConfig {
	if _, err := os.Stat(dbPath); err != nil {
		return CCSwitchProviderConfig{}
	}
	sqlitePath, err := exec.LookPath("sqlite3")
	if err != nil {
		return CCSwitchProviderConfig{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	query := `SELECT p.id, p.settings_config, p.is_current, COALESCE(e.url, '') AS endpoint_url
		FROM providers p
		LEFT JOIN provider_endpoints e ON e.provider_id = p.id AND e.app_type = p.app_type
		WHERE p.app_type = 'codex'
		ORDER BY p.is_current DESC, p.sort_index, e.id`
	cmd := exec.CommandContext(ctx, sqlitePath, "-readonly", "-json", dbPath, query)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return CCSwitchProviderConfig{}
	}
	var rows []struct {
		ID             string `json:"id"`
		SettingsConfig string `json:"settings_config"`
		IsCurrent      int    `json:"is_current"`
		EndpointURL    string `json:"endpoint_url"`
	}
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		return CCSwitchProviderConfig{}
	}
	for _, row := range rows {
		if currentID != "" && row.ID == currentID {
			if cfg := ccSwitchProviderConfigFromSQLiteRow(row.SettingsConfig, row.EndpointURL); !cfg.empty() {
				return cfg
			}
		}
	}
	for _, row := range rows {
		if row.IsCurrent != 0 {
			if cfg := ccSwitchProviderConfigFromSQLiteRow(row.SettingsConfig, row.EndpointURL); !cfg.empty() {
				return cfg
			}
		}
	}
	for _, row := range rows {
		if cfg := ccSwitchProviderConfigFromSQLiteRow(row.SettingsConfig, row.EndpointURL); !cfg.empty() {
			return cfg
		}
	}
	return CCSwitchProviderConfig{}
}

func ccSwitchProviderConfigFromSQLiteRow(settingsConfig, endpointURL string) CCSwitchProviderConfig {
	cfg := codexProviderConfigFromCCSwitchSettings(settingsConfig)
	if cfg.BaseURL == "" {
		cfg.BaseURL = normalizePublicProviderBaseURL(endpointURL)
	}
	return cfg
}

func ccSwitchCodexProviderBaseURLFromFiles(dir, currentID string) string {
	return ccSwitchCodexProviderConfigFromFiles(dir, currentID).BaseURL
}

func ccSwitchCodexProviderConfigFromFiles(dir, currentID string) CCSwitchProviderConfig {
	var paths []string
	for _, name := range []string{"config.json", "providers.json"} {
		paths = append(paths, filepath.Join(dir, name))
	}
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err == nil && strings.Count(rel, string(filepath.Separator)) > 3 {
			return nil
		}
		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".toml") ||
			strings.HasSuffix(name, ".db") || strings.HasSuffix(name, ".sqlite") || strings.HasSuffix(name, ".sqlite3") {
			paths = append(paths, path)
		}
		return nil
	})
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
		if cfg := codexProviderConfigFromCCSwitchBlob(b, currentID); !cfg.empty() {
			return cfg
		}
	}
	return CCSwitchProviderConfig{}
}

func ccSwitchCodexProviderBaseURLFromDBText(dbPath, currentID string) string {
	return ccSwitchCodexProviderConfigFromDBText(dbPath, currentID).BaseURL
}

func ccSwitchCodexProviderConfigFromDBText(dbPath, currentID string) CCSwitchProviderConfig {
	b, err := os.ReadFile(dbPath)
	if err != nil || len(b) > 64*1024*1024 {
		return CCSwitchProviderConfig{}
	}
	if currentID != "" {
		if idx := bytes.Index(b, []byte(currentID)); idx >= 0 {
			start := idx
			end := min(len(b), idx+32*1024)
			if cfg := codexProviderConfigFromCCSwitchBlob(b[start:end], currentID); !cfg.empty() {
				return cfg
			}
		}
	}
	return codexProviderConfigFromCCSwitchBlob(b, currentID)
}

func codexBaseURLFromCCSwitchBlob(b []byte, currentID string) string {
	return codexProviderConfigFromCCSwitchBlob(b, currentID).BaseURL
}

func codexProviderConfigFromCCSwitchBlob(b []byte, currentID string) CCSwitchProviderConfig {
	var data any
	if json.Unmarshal(b, &data) == nil {
		if cfg := codexProviderConfigFromCCSwitchJSON(data, currentID); !cfg.empty() {
			return cfg
		}
	}
	return CCSwitchProviderConfig{
		BaseURL:   publicProviderBaseURLFromText(string(b)),
		APIKey:    apiKeyFromText(string(b)),
		APIFormat: apiFormatFromText(string(b)),
	}
}

func codexBaseURLFromCCSwitchSettings(raw string) string {
	return codexProviderConfigFromCCSwitchSettings(raw).BaseURL
}

func codexProviderConfigFromCCSwitchSettings(raw string) CCSwitchProviderConfig {
	var data any
	if err := json.Unmarshal([]byte(raw), &data); err == nil {
		if cfg := codexProviderConfigFromCCSwitchJSON(data, ""); !cfg.empty() {
			return cfg
		}
	}
	return CCSwitchProviderConfig{
		BaseURL:   publicProviderBaseURLFromText(raw),
		APIKey:    apiKeyFromText(raw),
		APIFormat: apiFormatFromText(raw),
	}
}

func codexBaseURLFromCCSwitchJSON(v any, currentID string) string {
	return codexProviderConfigFromCCSwitchJSON(v, currentID).BaseURL
}

func codexProviderConfigFromCCSwitchJSON(v any, currentID string) CCSwitchProviderConfig {
	if currentID != "" {
		if obj, ok := v.(map[string]any); ok {
			if provider := findJSONObjectByID(obj, currentID); provider != nil {
				if cfg := codexProviderConfigFromCCSwitchJSON(provider, ""); !cfg.empty() {
					return cfg
				}
			}
		}
	}
	cfg := CCSwitchProviderConfig{
		APIKey:    apiKeyFromJSON(v),
		APIFormat: apiFormatFromJSON(v),
	}
	if configText := jsonStringAtPath(v, "config"); configText != "" {
		cfg = mergeProviderConfig(cfg, providerConfigFromCodexConfig(configText))
	}
	for _, key := range []string{"base_url", "baseUrl", "api_base_url", "apiBaseUrl", "url"} {
		if baseURL := normalizePublicProviderBaseURL(jsonStringAtPath(v, key)); baseURL != "" {
			cfg.BaseURL = baseURL
			return cfg
		}
	}
	switch x := v.(type) {
	case map[string]any:
		for _, value := range x {
			if nested := codexProviderConfigFromCCSwitchJSON(value, ""); !nested.empty() {
				return mergeProviderConfig(cfg, nested)
			}
		}
	case []any:
		for _, value := range x {
			if nested := codexProviderConfigFromCCSwitchJSON(value, ""); !nested.empty() {
				return mergeProviderConfig(cfg, nested)
			}
		}
	}
	return cfg
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
	return providerConfigFromCodexConfig(configText).BaseURL
}

func providerConfigFromCodexConfig(configText string) CCSwitchProviderConfig {
	values, err := parseSimpleTOMLStrings([]byte(configText))
	if err != nil {
		return CCSwitchProviderConfig{
			BaseURL:   publicProviderBaseURLFromText(configText),
			APIKey:    apiKeyFromText(configText),
			APIFormat: apiFormatFromText(configText),
		}
	}
	provider := strings.TrimSpace(values["model_provider"])
	if provider == "" {
		provider = strings.TrimSpace(values["provider"])
	}
	cfg := CCSwitchProviderConfig{
		APIKey:    firstNonEmptyString(apiKeyFromProviderValues(values, provider), apiKeyFromText(configText)),
		APIFormat: firstNonEmptyString(apiFormatFromProviderValues(values, provider), apiFormatFromText(configText)),
	}
	if provider != "" {
		if baseURL := normalizePublicProviderBaseURL(providerBaseURLRaw(values, provider)); baseURL != "" {
			cfg.BaseURL = baseURL
			return cfg
		}
	}
	cfg.BaseURL = normalizePublicProviderBaseURL(values["base_url"])
	return cfg
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

func mergeProviderConfig(base, override CCSwitchProviderConfig) CCSwitchProviderConfig {
	if base.BaseURL == "" {
		base.BaseURL = override.BaseURL
	}
	if base.APIKey == "" {
		base.APIKey = override.APIKey
	}
	if base.APIFormat == "" {
		base.APIFormat = override.APIFormat
	}
	return base
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func apiKeyFromProviderValues(values map[string]string, provider string) string {
	keys := []string{"api_key", "apiKey", "OPENAI_API_KEY", "openai_api_key", "key"}
	if provider != "" {
		for _, prefix := range providerBaseURLKeys(provider) {
			prefix = strings.TrimSuffix(prefix, ".base_url")
			for _, key := range keys {
				if v := strings.TrimSpace(values[prefix+"."+key]); v != "" {
					return v
				}
			}
			if envKey := strings.TrimSpace(values[prefix+".env_key"]); envKey != "" {
				if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
					return v
				}
			}
		}
	}
	for _, key := range keys {
		if v := strings.TrimSpace(values[key]); v != "" {
			return v
		}
	}
	if envKey := strings.TrimSpace(values["env_key"]); envKey != "" {
		return strings.TrimSpace(os.Getenv(envKey))
	}
	return ""
}

func apiFormatFromProviderValues(values map[string]string, provider string) string {
	keys := []string{"api_format", "apiFormat", "format", "wire_api", "wireApi"}
	if provider != "" {
		for _, prefix := range providerBaseURLKeys(provider) {
			prefix = strings.TrimSuffix(prefix, ".base_url")
			for _, key := range keys {
				if format := normalizeAPIFormatName(values[prefix+"."+key]); format != "" {
					return format
				}
			}
		}
	}
	for _, key := range keys {
		if format := normalizeAPIFormatName(values[key]); format != "" {
			return format
		}
	}
	return ""
}

func apiKeyFromJSON(v any) string {
	for _, key := range []string{"api_key", "apiKey", "openai_api_key", "OPENAI_API_KEY", "model_api_key", "modelApiKey", "key"} {
		if s := jsonStringAtPath(v, key); looksLikeSecret(s) {
			return s
		}
	}
	return ""
}

func apiFormatFromJSON(v any) string {
	for _, key := range []string{"api_format", "apiFormat", "format", "wire_api", "wireApi"} {
		if format := normalizeAPIFormatName(jsonStringAtPath(v, key)); format != "" {
			return format
		}
	}
	return ""
}

func apiKeyFromText(text string) string {
	for _, line := range strings.Split(text, "\n") {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "api_key") && !strings.Contains(lower, "apikey") &&
			!strings.Contains(lower, "openai_api_key") && !strings.Contains(lower, "model_api_key") {
			continue
		}
		if secret := secretValueFromLine(line); looksLikeSecret(secret) {
			return secret
		}
	}
	return ""
}

func apiFormatFromText(text string) string {
	for _, line := range strings.Split(text, "\n") {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "format") && !strings.Contains(lower, "wire_api") && !strings.Contains(lower, "wireapi") {
			continue
		}
		if format := normalizeAPIFormatName(line); format != "" {
			return format
		}
	}
	return ""
}

func secretValueFromLine(line string) string {
	lower := strings.ToLower(line)
	start := -1
	for _, key := range []string{"openai_api_key", "model_api_key", "api_key", "apikey", "apiKey"} {
		if idx := strings.Index(lower, strings.ToLower(key)); idx >= 0 {
			start = idx + len(key)
			break
		}
	}
	if start >= 0 {
		line = line[start:]
	}
	_, value, ok := strings.Cut(line, "=")
	if !ok {
		_, value, ok = strings.Cut(line, ":")
	}
	if !ok {
		return ""
	}
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"' ,`)
	if idx := strings.IndexAny(value, `"',}`); idx >= 0 {
		value = value[:idx]
	}
	fields := strings.Fields(value)
	if len(fields) > 0 {
		value = fields[0]
	}
	return strings.Trim(value, `"' ,`)
}

func looksLikeSecret(v string) bool {
	v = strings.TrimSpace(v)
	if len(v) < 8 {
		return false
	}
	lower := strings.ToLower(v)
	if strings.Contains(lower, "http://") || strings.Contains(lower, "https://") {
		return false
	}
	return !strings.ContainsAny(v, "\n\r\t ")
}

func normalizeAPIFormatName(v string) string {
	s := strings.ToLower(strings.TrimSpace(v))
	switch {
	case strings.Contains(s, "anthropic") || strings.Contains(s, "claude"):
		return "anthropic-messages"
	case strings.Contains(s, "response"):
		return "openai-responses"
	case strings.Contains(s, "chat") || strings.Contains(s, "completion"):
		return "openai-chat"
	default:
		return ""
	}
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
