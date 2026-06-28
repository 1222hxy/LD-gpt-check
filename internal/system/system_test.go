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
