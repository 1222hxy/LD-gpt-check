package wizard

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/haowang02/ld-gpt-check/internal/i18n"
)

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
