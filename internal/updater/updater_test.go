package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
		ok   bool
	}{
		{"v0.2.2", "0.2.1", 1, true},
		{"0.2.1", "v0.2.1", 0, true},
		{"0.2.0", "0.2.1", -1, true},
		{"0.2.1+build", "0.2.1", 0, true},
		{"bad", "0.2.1", 0, false},
	}
	for _, tt := range tests {
		got, ok := CompareVersions(tt.a, tt.b)
		if got != tt.want || ok != tt.ok {
			t.Fatalf("CompareVersions(%q, %q) = %d, %v; want %d, %v", tt.a, tt.b, got, ok, tt.want, tt.ok)
		}
	}
}

func TestAssetName(t *testing.T) {
	tests := []struct {
		goos, goarch, suffix, want string
	}{
		{"linux", "amd64", "", "ld-gpt-check_linux_amd64"},
		{"linux", "arm64", "", "ld-gpt-check_linux_arm64"},
		{"linux", "arm", "armv6", "ld-gpt-check_linux_armv6"},
		{"windows", "amd64", "", "ld-gpt-check_windows_amd64.exe"},
		{"darwin", "arm64", "", "ld-gpt-check_darwin_arm64"},
	}
	for _, tt := range tests {
		got, err := AssetName(tt.goos, tt.goarch, tt.suffix)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Fatalf("AssetName(%q,%q,%q) = %q, want %q", tt.goos, tt.goarch, tt.suffix, got, tt.want)
		}
	}
}

func TestParseChecksum(t *testing.T) {
	sum := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	body := []byte(fmt.Sprintf("%s  ld-gpt-check_linux_amd64\n", sum))
	got, err := ParseChecksum(body, "ld-gpt-check_linux_amd64")
	if err != nil {
		t.Fatal(err)
	}
	if got != sum {
		t.Fatalf("checksum = %q", got)
	}
}

func TestShouldCheckUsesDailyState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	c := Client{StatePath: path, Now: func() time.Time { return now }}
	if !c.ShouldCheck() {
		t.Fatal("missing state should check")
	}
	if err := SaveState(path, State{LastCheckedAt: now.Add(-time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if c.ShouldCheck() {
		t.Fatal("recent state should not check")
	}
	if err := SaveState(path, State{LastCheckedAt: now.Add(-25 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if !c.ShouldCheck() {
		t.Fatal("stale state should check")
	}
}

func TestCheckFindsLatestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"tag_name":"v0.2.2",
			"prerelease":false,
			"assets":[
				{"name":"ld-gpt-check_linux_amd64","browser_download_url":"https://example.test/bin"},
				{"name":"SHA256SUMS.txt","browser_download_url":"https://example.test/sha"}
			]
		}`)
	}))
	defer srv.Close()
	got, err := (Client{
		CurrentVersion:  "0.2.1",
		GOOS:            "linux",
		GOARCH:          "amd64",
		GitHubLatestURL: srv.URL,
		R2BaseURL:       "https://download.test/ld-gpt-check/latest",
	}).Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got.UpdateAvailable || got.LatestVersion != "0.2.2" || got.AssetName != "ld-gpt-check_linux_amd64" {
		t.Fatalf("check result = %#v", got)
	}
	if got.R2AssetURL != "https://download.test/ld-gpt-check/latest/ld-gpt-check_linux_amd64" {
		t.Fatalf("R2 url = %q", got.R2AssetURL)
	}
}

func TestCheckInfersLatestReleaseFromRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			http.Redirect(w, r, "/tag/v0.2.2", http.StatusFound)
		case "/tag/v0.2.2":
			fmt.Fprint(w, "<html>release</html>")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	got, err := (Client{
		CurrentVersion:  "0.2.1",
		GOOS:            "linux",
		GOARCH:          "amd64",
		GitHubLatestURL: srv.URL + "/latest",
	}).Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got.UpdateAvailable || got.TagName != "v0.2.2" || got.GitHubSHAURL == "" || got.GitHubAssetURL == "" {
		t.Fatalf("check result = %#v", got)
	}
}

func TestInstallFallsBackToR2AndReplacesExecutable(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "ld-gpt-check")
	if err := os.WriteFile(exe, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	next := []byte("new binary")
	sumRaw := sha256.Sum256(next)
	sum := hex.EncodeToString(sumRaw[:])
	var sawGitHubAsset bool
	var sawR2Asset bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/github/bin":
			sawGitHubAsset = true
			http.Error(w, "down", http.StatusBadGateway)
		case "/github/SHA256SUMS.txt":
			fmt.Fprintf(w, "%s  ld-gpt-check_linux_amd64\n", sum)
		case "/r2/ld-gpt-check_linux_amd64":
			sawR2Asset = true
			_, _ = w.Write(next)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	status, err := (Client{Executable: exe}).Install(context.Background(), CheckResult{
		AssetName:      "ld-gpt-check_linux_amd64",
		GitHubAssetURL: srv.URL + "/github/bin",
		GitHubSHAURL:   srv.URL + "/github/SHA256SUMS.txt",
		R2AssetURL:     srv.URL + "/r2/ld-gpt-check_linux_amd64",
		R2SHAURL:       srv.URL + "/r2/SHA256SUMS.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if status != "installed" {
		t.Fatalf("status = %q", status)
	}
	b, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != string(next) || !sawGitHubAsset || !sawR2Asset {
		t.Fatalf("install did not fallback and replace: got=%q github=%v r2=%v", b, sawGitHubAsset, sawR2Asset)
	}
}

func TestInstallRejectsChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "ld-gpt-check")
	if err := os.WriteFile(exe, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	badSum := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/SHA256SUMS.txt":
			fmt.Fprintf(w, "%s  ld-gpt-check_linux_amd64\n", badSum)
		case "/bin":
			_, _ = w.Write([]byte("new"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	_, err := (Client{Executable: exe}).Install(context.Background(), CheckResult{
		AssetName:      "ld-gpt-check_linux_amd64",
		GitHubAssetURL: srv.URL + "/bin",
		GitHubSHAURL:   srv.URL + "/SHA256SUMS.txt",
	})
	if err == nil {
		t.Fatal("expected checksum mismatch")
	}
	b, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "old" {
		t.Fatalf("executable changed after failed update: %q", b)
	}
}
