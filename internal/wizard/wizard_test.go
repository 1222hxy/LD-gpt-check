package wizard

import (
	"bufio"
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/haowang02/ld-gpt-check/internal/config"
	"github.com/haowang02/ld-gpt-check/internal/i18n"
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
	if !strings.Contains(text, "正在使用 API："+config.DefaultAPIBase) {
		t.Fatalf("wizard did not show default API:\n%s", text)
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
