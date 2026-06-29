package updater

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultGitHubLatestURL = "https://github.com/1222hxy/LD-gpt-check/releases/latest"
	DefaultR2BaseURL       = "https://download.yhklab.com/ld-gpt-check/latest"
	EnvNoUpdate            = "LD_GPT_CHECK_NO_UPDATE"
	CheckInterval          = 24 * time.Hour
	userAgent              = "ld-gpt-check updater"
)

type Client struct {
	CurrentVersion  string
	AssetSuffix     string
	GOOS            string
	GOARCH          string
	Executable      string
	StatePath       string
	GitHubLatestURL string
	R2BaseURL       string
	HTTP            *http.Client
	Now             func() time.Time
}

type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	TagName         string
	AssetName       string
	GitHubAssetURL  string
	GitHubSHAURL    string
	R2AssetURL      string
	R2SHAURL        string
	UpdateAvailable bool
}

type State struct {
	LastCheckedAt time.Time `json:"last_checked_at"`
}

type githubRelease struct {
	TagName    string        `json:"tag_name"`
	Prerelease bool          `json:"prerelease"`
	HTMLURL    string        `json:"html_url"`
	Assets     []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (c Client) Check(ctx context.Context) (CheckResult, error) {
	c = c.withDefaults()
	assetName, err := AssetName(c.GOOS, c.GOARCH, c.AssetSuffix)
	if err != nil {
		return CheckResult{}, err
	}
	release, err := c.fetchLatest(ctx)
	if err != nil {
		return CheckResult{}, err
	}
	latest := NormalizeVersion(release.TagName)
	current := NormalizeVersion(c.CurrentVersion)
	result := CheckResult{
		CurrentVersion: current,
		LatestVersion:  latest,
		TagName:        release.TagName,
		AssetName:      assetName,
		R2AssetURL:     joinURL(c.R2BaseURL, assetName),
		R2SHAURL:       joinURL(c.R2BaseURL, "SHA256SUMS.txt"),
	}
	for _, a := range release.Assets {
		switch a.Name {
		case assetName:
			result.GitHubAssetURL = a.BrowserDownloadURL
		case "SHA256SUMS.txt":
			result.GitHubSHAURL = a.BrowserDownloadURL
		}
	}
	if result.GitHubAssetURL == "" {
		result.GitHubAssetURL = githubDownloadURL(release.TagName, assetName)
	}
	if result.GitHubSHAURL == "" {
		result.GitHubSHAURL = githubDownloadURL(release.TagName, "SHA256SUMS.txt")
	}
	cmp, ok := CompareVersions(latest, current)
	result.UpdateAvailable = ok && cmp > 0
	return result, nil
}

func (c Client) Install(ctx context.Context, result CheckResult) (string, error) {
	c = c.withDefaults()
	if result.AssetName == "" {
		return "", errors.New("asset name is required")
	}
	exe := c.Executable
	if exe == "" {
		var err error
		exe, err = os.Executable()
		if err != nil {
			return "", err
		}
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	shaBody, shaSource, err := c.downloadBytesFallback(ctx, result.GitHubSHAURL, result.R2SHAURL, 1<<20)
	if err != nil {
		return "", fmt.Errorf("download checksums: %w", err)
	}
	want, err := ParseChecksum(shaBody, result.AssetName)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", shaSource, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(exe), "."+filepath.Base(exe)+".update-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	keepTmp := false
	defer func() {
		_ = tmp.Close()
		if !keepTmp {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err = c.downloadFileFallback(ctx, result.GitHubAssetURL, result.R2AssetURL, tmp); err != nil {
		return "", fmt.Errorf("download binary: %w", err)
	}
	if _, err = tmp.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	sum := sha256.New()
	if _, err = io.Copy(sum, tmp); err != nil {
		return "", err
	}
	if got := hex.EncodeToString(sum.Sum(nil)); !strings.EqualFold(got, want) {
		return "", fmt.Errorf("checksum mismatch for %s", result.AssetName)
	}
	mode := os.FileMode(0755)
	if st, err := os.Stat(exe); err == nil {
		mode = st.Mode().Perm()
	}
	if err = tmp.Chmod(mode); err != nil {
		return "", err
	}
	if err = tmp.Close(); err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		keepTmp = true
		return "pending", scheduleWindowsReplace(exe, tmpName)
	}
	if err = replaceExecutable(exe, tmpName); err != nil {
		return "", err
	}
	keepTmp = true
	return "installed", nil
}

func (c Client) ShouldCheck() bool {
	if NoUpdateDisabled() {
		return false
	}
	state, err := LoadState(c.statePath())
	if err != nil {
		return true
	}
	return c.now().Sub(state.LastCheckedAt) >= CheckInterval
}

func (c Client) MarkChecked() {
	_ = SaveState(c.statePath(), State{LastCheckedAt: c.now()})
}

func (c Client) fetchLatest(ctx context.Context) (githubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.GitHubLatestURL, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return githubRelease{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return githubRelease{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return githubRelease{}, fmt.Errorf("GitHub release check failed: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var release githubRelease
	if json.Valid(body) {
		if err := json.Unmarshal(body, &release); err != nil {
			return githubRelease{}, err
		}
	} else {
		release.TagName = inferTagFromURL(resp.Request.URL)
	}
	if strings.TrimSpace(release.TagName) == "" {
		return githubRelease{}, errors.New("release response missing tag_name")
	}
	if release.Prerelease {
		return githubRelease{}, errors.New("latest release is prerelease")
	}
	return release, nil
}

func inferTagFromURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "tag" {
			return parts[i+1]
		}
	}
	return ""
}

func githubDownloadURL(tag, asset string) string {
	return "https://github.com/1222hxy/LD-gpt-check/releases/download/" + url.PathEscape(tag) + "/" + strings.TrimLeft(asset, "/")
}

func (c Client) downloadBytesFallback(ctx context.Context, primary, fallback string, max int64) ([]byte, string, error) {
	var last error
	for _, raw := range []string{primary, fallback} {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		b, err := c.downloadBytes(ctx, raw, max)
		if err == nil {
			return b, raw, nil
		}
		last = err
	}
	if last == nil {
		last = errors.New("no download URL")
	}
	return nil, "", last
}

func (c Client) downloadBytes(ctx context.Context, raw string, max int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > max {
		return nil, errors.New("response too large")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func (c Client) downloadFileFallback(ctx context.Context, primary, fallback string, dst *os.File) (string, error) {
	var last error
	for _, raw := range []string{primary, fallback} {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		if err := dst.Truncate(0); err != nil {
			return "", err
		}
		if _, err := dst.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
		err := c.downloadFile(ctx, raw, dst)
		if err == nil {
			return raw, nil
		}
		last = err
	}
	if last == nil {
		last = errors.New("no download URL")
	}
	return "", last
}

func (c Client) downloadFile(ctx context.Context, raw string, dst io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	_, err = io.Copy(dst, resp.Body)
	return err
}

func (c Client) withDefaults() Client {
	if c.CurrentVersion == "" {
		c.CurrentVersion = "0.0.0"
	}
	if c.GOOS == "" {
		c.GOOS = runtime.GOOS
	}
	if c.GOARCH == "" {
		c.GOARCH = runtime.GOARCH
	}
	if c.GitHubLatestURL == "" {
		c.GitHubLatestURL = DefaultGitHubLatestURL
	}
	if c.R2BaseURL == "" {
		c.R2BaseURL = DefaultR2BaseURL
	}
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 20 * time.Second}
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return c
}

func (c Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c Client) statePath() string {
	if strings.TrimSpace(c.StatePath) != "" {
		return c.StatePath
	}
	return DefaultStatePath()
}

func DefaultStatePath() string {
	if dir, err := os.UserCacheDir(); err == nil && strings.TrimSpace(dir) != "" {
		return filepath.Join(dir, "ld-gpt-check", "update-state.json")
	}
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), "update-state.json")
	}
	return "update-state.json"
}

func LoadState(path string) (State, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(b, &state); err != nil {
		return State{}, err
	}
	if state.LastCheckedAt.IsZero() {
		return State{}, errors.New("last_checked_at is empty")
	}
	return state, nil
}

func SaveState(path string, state State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0600)
}

func NoUpdateDisabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(EnvNoUpdate)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "+-"); i >= 0 {
		v = v[:i]
	}
	return v
}

func CompareVersions(a, b string) (int, bool) {
	av, okA := parseVersion(a)
	bv, okB := parseVersion(b)
	if !okA || !okB {
		return 0, false
	}
	for i := 0; i < 3; i++ {
		if av[i] > bv[i] {
			return 1, true
		}
		if av[i] < bv[i] {
			return -1, true
		}
	}
	return 0, true
}

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	v = NormalizeVersion(v)
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func AssetName(goos, goarch, suffix string) (string, error) {
	goos = strings.TrimSpace(goos)
	goarch = strings.TrimSpace(goarch)
	suffix = strings.TrimSpace(suffix)
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	if suffix == "" {
		switch goarch {
		case "amd64", "arm64":
			suffix = goarch
		case "arm":
			suffix = "armv7"
		default:
			return "", fmt.Errorf("unsupported architecture %s/%s", goos, goarch)
		}
	}
	name := "ld-gpt-check_" + goos + "_" + suffix
	if goos == "windows" && !strings.HasSuffix(name, ".exe") {
		name += ".exe"
	}
	return name, nil
}

func ParseChecksum(b []byte, assetName string) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		name = filepath.Base(name)
		if name == assetName {
			sum := strings.ToLower(fields[0])
			if len(sum) != sha256.Size*2 {
				return "", fmt.Errorf("invalid checksum for %s", assetName)
			}
			if _, err := hex.DecodeString(sum); err != nil {
				return "", err
			}
			return sum, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("checksum for %s not found", assetName)
}

func joinURL(base, elem string) string {
	base = strings.TrimSpace(base)
	elem = strings.TrimLeft(strings.TrimSpace(elem), "/")
	if base == "" {
		return elem
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return strings.TrimRight(base, "/") + "/" + elem
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/" + elem
	return u.String()
}

func replaceExecutable(exe, next string) error {
	backup := exe + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(exe, backup); err != nil {
		return err
	}
	if err := os.Rename(next, exe); err != nil {
		_ = os.Rename(backup, exe)
		return err
	}
	_ = os.Remove(backup)
	return nil
}

func scheduleWindowsReplace(exe, next string) error {
	script := next + ".cmd"
	body := fmt.Sprintf("@echo off\r\nping 127.0.0.1 -n 2 > nul\r\nmove /Y %q %q > nul\r\ndel \"%%~f0\" > nul\r\n", next, exe)
	if err := os.WriteFile(script, []byte(body), 0600); err != nil {
		return err
	}
	return exec.Command("cmd", "/C", "start", "/B", "", script).Start()
}
