package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigFileName)
	t.Setenv(ConfigEnvVarName, path)

	want := Config{
		APIBaseURL: "https://example.com",
		Language:   "en",
		User: User{
			ID:       "8028dd70-a644-4138-9156-43e1bb228e5b",
			Username: "alice",
		},
		DeviceAuthorization: DeviceAuthorization{
			Secret: "device-secret",
		},
	}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, `api_base_url = "https://example.com"`) {
		t.Fatalf("saved config is not expected TOML:\n%s", text)
	}
	if !strings.Contains(text, `[device_authorization]`) || !strings.Contains(text, `secret = "device-secret"`) {
		t.Fatalf("saved config missing device authorization secret:\n%s", text)
	}
	if strings.Contains(text, "access_token") || strings.Contains(text, "[user]") || strings.Contains(text, want.User.ID) {
		t.Fatalf("saved config leaked legacy auth fields or user id:\n%s", text)
	}
	if strings.HasPrefix(strings.TrimSpace(text), "{") {
		t.Fatalf("saved config should not be JSON:\n%s", text)
	}

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.APIBaseURL != want.APIBaseURL || got.Language != want.Language {
		t.Fatalf("loaded config = %#v", got)
	}
	if got.AccessToken != "device-secret" || got.DeviceAuthorization.Secret != "device-secret" {
		t.Fatalf("loaded device authorization = %#v", got)
	}
	if got.User.ID != "" || got.User.Username != "" {
		t.Fatalf("saved file should not restore user identity: %#v", got.User)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("config permissions = %o", info.Mode().Perm())
	}
}

func TestLoadMissingConfigUsesProductionAPI(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigFileName)
	t.Setenv(ConfigEnvVarName, path)
	t.Setenv("LD_GPT_CHECK_API_BASE_URL", "")

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.APIBaseURL != DefaultAPIBase {
		t.Fatalf("APIBaseURL = %q, want %q", got.APIBaseURL, DefaultAPIBase)
	}
}

func TestLoadTOMLWithComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigFileName)
	t.Setenv(ConfigEnvVarName, path)
	err := os.WriteFile(path, []byte(`
api_base_url = "https://example.com" # comment
access_token = "token#not-comment"
language = "zh-CN"

[user]
id = "u1"
username = "bob"
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "token#not-comment" {
		t.Fatalf("access token = %q", got.AccessToken)
	}
}

func TestLoadLegacyAccessTokenAsDeviceAuthorization(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigFileName)
	t.Setenv(ConfigEnvVarName, path)
	err := os.WriteFile(path, []byte(`
api_base_url = "https://example.com"
access_token = "legacy-token"
language = "zh-CN"

[user]
id = "8028dd70-a644-4138-9156-43e1bb228e5b"
username = "bob"
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "legacy-token" || got.DeviceAuthorization.Secret != "legacy-token" {
		t.Fatalf("legacy token not normalized: %#v", got)
	}
}

func TestLoadTOMLRejectsUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigFileName)
	t.Setenv(ConfigEnvVarName, path)
	if err := os.WriteFile(path, []byte(`api_url = "https://example.com"`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("expected unknown key error")
	}
}

func TestLoadTOMLRejectsUnquotedValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigFileName)
	t.Setenv(ConfigEnvVarName, path)
	if err := os.WriteFile(path, []byte(`api_base_url = https://example.com`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("expected unquoted value error")
	}
}

func TestPathUsesEnvOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.toml")
	t.Setenv(ConfigEnvVarName, path)
	got, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("Path = %q, want %q", got, path)
	}
}
