package system

import (
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
