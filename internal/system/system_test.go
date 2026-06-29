package system

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestCodexConfiguredModelReadsRootModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
model = "gpt-5.5"

[projects."/tmp"]
model = "other"
`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := CodexConfiguredModel()
	if err != nil {
		t.Fatal(err)
	}
	if got != "gpt-5.5" {
		t.Fatalf("model = %q", got)
	}
}

func TestCodexConfigInfoReadsProviderHost(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
model = "gpt-5.5"
model_provider = "linuxdo"

[model_providers.linuxdo]
base_url = "https://API.example.com/v1/?token=secret#fragment"
`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := CodexConfigInfo()
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "gpt-5.5" || got.ModelProvider != "linuxdo" || got.ProviderHost != "api.example.com" || got.ProviderBaseURL != "https://api.example.com/v1" {
		t.Fatalf("config = %#v", got)
	}
}

func TestCodexConfigInfoDefaultsProviderBaseURLToOpenAI(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`model = "gpt-5.5"`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := CodexConfigInfo()
	if err != nil {
		t.Fatal(err)
	}
	if got.ProviderBaseURL != "https://api.openai.com/v1" || got.ProviderHost != "api.openai.com" {
		t.Fatalf("config = %#v", got)
	}
}

func TestCodexDefaultConfigPathsIncludesMacApplicationSupport(t *testing.T) {
	home := filepath.Join("Users", "tester")
	paths := codexDefaultConfigPaths(home, "darwin")
	want := filepath.Join(home, "Library", "Application Support", "Codex", "config.toml")
	found := false
	for _, path := range paths {
		if path == want {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("mac Codex config path missing from %#v", paths)
	}
}

func TestCodexConfigInfoResolvesCCSwitchUpstreamForLocalProxy(t *testing.T) {
	home := t.TempDir()
	codexHome := filepath.Join(home, "codex")
	ccSwitchHome := filepath.Join(home, ".cc-switch")
	if err := os.MkdirAll(codexHome, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(ccSwitchHome, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("LD_GPT_CHECK_CC_SWITCH_DIR", ccSwitchHome)
	t.Setenv("LD_GPT_CHECK_DISABLE_CC_SWITCH", "")
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte(`
model = "gpt-5.5"
model_provider = "cc-switch"

[model_providers.cc-switch]
base_url = "http://127.0.0.1:18443/v1"
`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ccSwitchHome, "config.json"), []byte(`{
  "currentProviderCodex": "krill",
  "providers": {
    "krill": {
      "settingsConfig": {
        "config": "model_provider = \"krill\"\n[model_providers.krill]\nbase_url = \"https://api.cdn-krill-ai.com/codex/v1?token=secret#fragment\"\n"
      }
    }
  }
}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := CodexConfigInfo()
	if err != nil {
		t.Fatal(err)
	}
	if got.ProviderBaseURL != "https://api.cdn-krill-ai.com/codex/v1" || got.ProviderHost != "api.cdn-krill-ai.com" {
		t.Fatalf("config = %#v", got)
	}
	resolution := DetectCCSwitchCodexResolution()
	if resolution.LocalBaseURL != "http://127.0.0.1:18443/v1" || resolution.ProviderBaseURL != "https://api.cdn-krill-ai.com/codex/v1" {
		t.Fatalf("resolution = %#v", resolution)
	}
}

func TestDetectCCSwitchResolutionWithoutCodexProxy(t *testing.T) {
	home := t.TempDir()
	codexHome := filepath.Join(home, "codex")
	ccSwitchHome := filepath.Join(home, ".cc-switch")
	if err := os.MkdirAll(codexHome, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(ccSwitchHome, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("LD_GPT_CHECK_CC_SWITCH_DIR", ccSwitchHome)
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte(`model = "gpt-5.5"`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ccSwitchHome, "providers.json"), []byte(`{
  "providers": [{"id":"krill","baseUrl":"https://api.krill-ai.example/codex/v1"}]
}`), 0600); err != nil {
		t.Fatal(err)
	}
	resolution := DetectCCSwitchCodexResolution()
	if resolution.LocalBaseURL != "" || resolution.ProviderBaseURL != "https://api.krill-ai.example/codex/v1" {
		t.Fatalf("resolution = %#v", resolution)
	}
}

func TestCCSwitchProviderBaseURLScansDBWithoutSQLiteCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("LD_GPT_CHECK_CC_SWITCH_DIR", home)
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte(`{"currentProviderCodex":"krill"}`), 0600); err != nil {
		t.Fatal(err)
	}
	dbLike := []byte(`noise old https://api.old.example/v1 krill {"base_url":"https://api.krill-ai.example/codex/v1","api_key":"sk-ccswitch-secret","wire_api":"responses"} tail`)
	if err := os.WriteFile(filepath.Join(home, "cc-switch.db"), dbLike, 0600); err != nil {
		t.Fatal(err)
	}
	cfg := CCSwitchCodexProviderConfig()
	if cfg.BaseURL != "https://api.krill-ai.example/codex/v1" || cfg.APIKey != "sk-ccswitch-secret" || cfg.APIFormat != "openai-responses" {
		t.Fatalf("provider config = %#v", cfg)
	}
}

func TestCCSwitchProviderConfigMergesConfigAndNestedAuthKey(t *testing.T) {
	raw := `{
  "config": "model_provider = \"custom\"\n[model_providers.custom]\nbase_url = \"https://api.krill-ai.example/codex/v1\"\nwire_api = \"responses\"\n",
  "auth": {
    "OPENAI_API_KEY": "sk-realistic-secret"
  }
}`
	cfg := codexProviderConfigFromCCSwitchSettings(raw)
	if cfg.BaseURL != "https://api.krill-ai.example/codex/v1" || cfg.APIKey != "sk-realistic-secret" || cfg.APIFormat != "openai-responses" {
		t.Fatalf("provider config = %#v", cfg)
	}
}

func TestCCSwitchSQLiteReaderSelectsCodexNotClaude(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cc-switch.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE providers (
  id TEXT NOT NULL,
  app_type TEXT NOT NULL,
  name TEXT NOT NULL,
  settings_config TEXT NOT NULL,
  is_current BOOLEAN NOT NULL DEFAULT 0,
  sort_index INTEGER,
  PRIMARY KEY (id, app_type)
);
CREATE TABLE provider_endpoints (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  provider_id TEXT NOT NULL,
  app_type TEXT NOT NULL,
  url TEXT NOT NULL
);
INSERT INTO providers(id, app_type, name, settings_config, is_current, sort_index) VALUES
  ('claude-krill', 'claude', 'Krill Claude', '{"env":{"ANTHROPIC_BASE_URL":"https://claude-wrong.example/v1","ANTHROPIC_AUTH_TOKEN":"claude-secret"}}', 1, 1),
  ('codex-krill', 'codex', 'Krill Codex', '{"config":"model_provider = \"custom\"\n[model_providers.custom]\nbase_url = \"https://codex-right.example/v1%5C\"\nwire_api = \"responses\"\n","auth":{"OPENAI_API_KEY":"sk-codex-secret"}}', 1, 1);
INSERT INTO provider_endpoints(provider_id, app_type, url) VALUES
  ('claude-krill', 'claude', 'https://claude-wrong.example/v1'),
  ('codex-krill', 'codex', 'https://codex-right.example/v1');
`)
	if err != nil {
		t.Fatal(err)
	}
	cfg := ccSwitchCodexProviderConfigFromSQLiteDB(dbPath, "")
	if cfg.BaseURL != "https://codex-right.example/v1" || cfg.APIKey != "sk-codex-secret" || cfg.APIFormat != "openai-responses" {
		t.Fatalf("provider config = %#v", cfg)
	}
}

func TestCCSwitchProviderConfigReadsDirectDBPath(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "custom-cc-switch.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE providers (
  id TEXT NOT NULL,
  app_type TEXT NOT NULL,
  name TEXT NOT NULL,
  settings_config TEXT NOT NULL,
  is_current BOOLEAN NOT NULL DEFAULT 0,
  sort_index INTEGER,
  PRIMARY KEY (id, app_type)
);
CREATE TABLE provider_endpoints (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  provider_id TEXT NOT NULL,
  app_type TEXT NOT NULL,
  url TEXT NOT NULL
);
INSERT INTO providers(id, app_type, name, settings_config, is_current, sort_index) VALUES
  ('codex-krill', 'codex', 'Krill Codex', '{"config":"base_url = \"https://mac-krill.example/codex/v1\"\nwire_api = \"responses\"\n","auth":{"OPENAI_API_KEY":"sk-mac-secret"}}', 1, 1);
`)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LD_GPT_CHECK_CC_SWITCH_DIR", "")
	t.Setenv("CC_SWITCH_CONFIG_DIR", "")
	t.Setenv("CC_SWITCH_HOME", "")
	t.Setenv("LD_GPT_CHECK_CC_SWITCH_DB", dbPath)
	cfg := CCSwitchCodexProviderConfig()
	if cfg.BaseURL != "https://mac-krill.example/codex/v1" || cfg.APIKey != "sk-mac-secret" ||
		cfg.APIFormat != "openai-responses" || cfg.ConfigDir != dir {
		t.Fatalf("provider config = %#v", cfg)
	}
}

func TestCCSwitchDefaultConfigDirsIncludesMacApplicationSupport(t *testing.T) {
	home := filepath.Join("Users", "tester")
	dirs := ccSwitchDefaultConfigDirs(home, "darwin")
	want := filepath.Join(home, "Library", "Application Support", "CC Switch")
	found := false
	for _, dir := range dirs {
		if dir == want {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("mac application support dir missing from %#v", dirs)
	}
}

func TestCodexConfigInfoDoesNotDefaultLocalProxyToOpenAI(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	t.Setenv("LD_GPT_CHECK_CC_SWITCH_DIR", filepath.Join(home, ".cc-switch-missing"))
	t.Setenv("LD_GPT_CHECK_DISABLE_CC_SWITCH", "")
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
model = "gpt-5.5"
model_provider = "cc-switch"

[model_providers.cc-switch]
base_url = "http://127.0.0.1:18443/v1"
`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := CodexConfigInfo()
	if err != nil {
		t.Fatal(err)
	}
	if got.ProviderBaseURL != "" || got.ProviderHost != "" {
		t.Fatalf("config = %#v, want empty provider for unresolved local proxy", got)
	}
}

func TestNormalizeProviderBaseURL(t *testing.T) {
	tests := map[string]string{
		" https://API.EXAMPLE.com/v1/ ":                     "https://api.example.com/v1",
		"https://api.example.com/tenant/path/?token=secret": "https://api.example.com/tenant/path",
		"https://user:pass@api.example.com/v1":              "https://api.example.com/v1",
		"https://api.example.com":                           "https://api.example.com",
		"https://api.example.com/v1%5C":                     "https://api.example.com/v1",
		`https://api.example.com/v1\`:                       "https://api.example.com/v1",
		"http://api.example.com/v1":                         "",
		"not a url":                                         "",
	}
	for input, want := range tests {
		if got := NormalizeProviderBaseURL(input); got != want {
			t.Fatalf("NormalizeProviderBaseURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCodexConfigInfoSkipsNonStringValues(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
model = "gpt-5.5"
approval_policy = "never"
disable_response_storage = true
experimental = ["a", "b"]
`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := CodexConfigInfo()
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "gpt-5.5" {
		t.Fatalf("config = %#v", got)
	}
}

func TestCodexConfiguredModelIgnoresDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`model = "default"`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := CodexConfiguredModel()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("model = %q, want empty", got)
	}
}

func TestConcreteCodexModel(t *testing.T) {
	if !ConcreteCodexModel("gpt-5.5") {
		t.Fatal("gpt-5.5 should be concrete")
	}
	for _, v := range []string{"", "default", "codex-default", "unknown-codex-model"} {
		if ConcreteCodexModel(v) {
			t.Fatalf("%q should not be concrete", v)
		}
	}
}

func TestUploadCodexVersionUsesAPIPlaceholder(t *testing.T) {
	if got := UploadCodexVersion("api"); got != "api" {
		t.Fatalf("UploadCodexVersion(api) = %q, want api", got)
	}
}
