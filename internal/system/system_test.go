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

func TestNormalizeProviderBaseURL(t *testing.T) {
	tests := map[string]string{
		" https://API.EXAMPLE.com/v1/ ":                     "https://api.example.com/v1",
		"https://api.example.com/tenant/path/?token=secret": "https://api.example.com/tenant/path",
		"https://api.example.com":                           "https://api.example.com",
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
