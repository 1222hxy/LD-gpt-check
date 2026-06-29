package wizard

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/1222hxy/LD-gpt-check/internal/config"
	"github.com/1222hxy/LD-gpt-check/internal/i18n"
	"github.com/1222hxy/LD-gpt-check/internal/runner"
)

func TestRunWizardUsesProductionAPIWithoutAsking(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), config.ConfigFileName)
	t.Setenv(config.ConfigEnvVarName, configPath)

	var out bytes.Buffer
	err := Run(context.Background(), Options{
		Version: "test",
		Lang:    i18n.ZH,
		Stdin:   strings.NewReader("否\n否\n"),
		Stdout:  &out,
	})
	if err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if strings.Contains(text, "Worker API 地址") || strings.Contains(text, "界面语言") {
		t.Fatalf("wizard should not ask setup questions:\n%s", text)
	}
	if !strings.Contains(text, config.DefaultAPIBase) {
		t.Fatalf("wizard did not show default API:\n%s", text)
	}
	if !strings.Contains(text, "检查配置") || !strings.Contains(text, "登录状态") || !strings.Contains(text, "运行测试") {
		t.Fatalf("wizard did not show step sections:\n%s", text)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIBaseURL != config.DefaultAPIBase {
		t.Fatalf("saved APIBaseURL = %q", cfg.APIBaseURL)
	}
}

func TestPromptBoolDefaultsAndChineseInput(t *testing.T) {
	var out bytes.Buffer
	l := i18n.New(i18n.ZH)
	ok, err := promptBool(bufio.NewReader(strings.NewReader("\n")), &out, l, "登录", true)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected default true")
	}

	out.Reset()
	ok, err = promptBool(bufio.NewReader(strings.NewReader("否\n")), &out, l, "上传", true)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false for Chinese no")
	}
}

func TestPromptBackendSelectsAPIWhenCodexAvailable(t *testing.T) {
	var out bytes.Buffer
	got, err := promptBackend(bufio.NewReader(strings.NewReader("2\n")), &out, i18n.New(i18n.ZH), false, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != runner.BackendAPI {
		t.Fatalf("backend = %q", got)
	}
	if !strings.Contains(out.String(), "API 模式") {
		t.Fatalf("missing API option:\n%s", out.String())
	}
}

func TestPromptBackendDefaultsToAPIWithoutCodex(t *testing.T) {
	var out bytes.Buffer
	got, err := promptBackend(bufio.NewReader(strings.NewReader("\n")), &out, i18n.New(i18n.ZH), false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != runner.BackendAPI {
		t.Fatalf("backend = %q", got)
	}
}

func TestPromptEffortRetriesInvalidValue(t *testing.T) {
	var out bytes.Buffer
	got, err := promptEffort(bufio.NewReader(strings.NewReader("bad\nxhigh\n")), &out, i18n.New(i18n.ZH), "推理强度", "medium")
	if err != nil {
		t.Fatal(err)
	}
	if got != "xhigh" {
		t.Fatalf("effort = %q", got)
	}
}

func TestPromptOptionalStringAllowsDefaultCodexConfig(t *testing.T) {
	var out bytes.Buffer
	got, err := promptOptionalString(bufio.NewReader(strings.NewReader("\n")), &out, "模型", "Codex 本机配置")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("optional model = %q", got)
	}
}

func TestPromptModelUsesDetectedCodexConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`model = "gpt-5.5"`), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	got, err := promptModel(bufio.NewReader(strings.NewReader("\n")), &out, i18n.New(i18n.ZH), false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("model = %q, want empty to keep Codex config", got)
	}
	if !strings.Contains(out.String(), "gpt-5.5") {
		t.Fatalf("missing detected model:\n%s", out.String())
	}
}

func TestPromptModelRequiresChoiceWhenCodexConfigIsDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`model = "default"`), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	got, err := promptModel(bufio.NewReader(strings.NewReader("2\n")), &out, i18n.New(i18n.ZH), false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "gpt-5.4" {
		t.Fatalf("model = %q", got)
	}
	if !strings.Contains(out.String(), "无法确认") {
		t.Fatalf("missing choice prompt:\n%s", out.String())
	}
}

func TestPromptDuration(t *testing.T) {
	var out bytes.Buffer
	got, err := promptDuration(bufio.NewReader(strings.NewReader("90s\n")), &out, i18n.New(i18n.ZH), "超时", 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got != 90*time.Second {
		t.Fatalf("duration = %s", got)
	}
}

func TestWizardRunsAPIModeWithoutSavingKey(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), config.ConfigFileName)
	t.Setenv(config.ConfigEnvVarName, configPath)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LD_GPT_CHECK_MODEL_API_KEY", "")

	oldRunBenchmark := runBenchmark
	defer func() { runBenchmark = oldRunBenchmark }()
	var captured runner.Options
	runBenchmark = func(ctx context.Context, opts runner.Options) (runner.Summary, error) {
		captured = opts
		return runner.Summary{
			Model:        opts.Model,
			Tests:        1,
			Correct:      1,
			Accuracy:     100,
			CodexSandbox: "api",
			Cases: []runner.CaseResult{{
				Index:         1,
				OK:            true,
				Status:        "completed",
				AnswerPreview: "21",
			}},
		}, nil
	}

	input := strings.Join([]string{
		"否",
		"是",
		"",
		"1",
		"https://api.example.com/v1",
		"wizard-key",
		"",
		"",
		"1",
		"5s",
		"否",
		"",
	}, "\n")
	var out bytes.Buffer
	if err := Run(context.Background(), Options{
		Version: "test",
		Lang:    i18n.ZH,
		Stdin:   strings.NewReader(input),
		Stdout:  &out,
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "wizard-key") {
		t.Fatalf("API key was written to config:\n%s", raw)
	}
	if captured.Backend != runner.BackendAPI || captured.APIFormat != runner.APIFormatOpenAIChat ||
		captured.ModelAPIBaseURL != "https://api.example.com/v1" || captured.ModelAPIKey != "wizard-key" {
		t.Fatalf("captured options = %#v", captured)
	}
	if !strings.Contains(out.String(), "API 模式") || !strings.Contains(out.String(), "临时 API Key") {
		t.Fatalf("missing API prompts:\n%s", out.String())
	}
}
