package runner

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/1222hxy/LD-gpt-check/internal/i18n"
	"github.com/1222hxy/LD-gpt-check/internal/questions"
)

func TestParseEventsExtractsMessageAndUsage(t *testing.T) {
	input := strings.Join([]string{
		`not-json`,
		`{"type":"item.completed","item":{"role":"assistant","content":[{"type":"output_text","text":"最少需要取出 **21 个**。"}]}}`,
		`{"type":"turn.completed","usage":{"input_tokens":"10163","output_tokens":4873,"output_tokens_details":{"reasoning_tokens":4660}}}`,
	}, "\n")

	parsed, err := parseEvents(strings.NewReader(input), i18n.ZH)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.ToolUsed {
		t.Fatal("did not expect tool use")
	}
	if parsed.FinalAnswer != "最少需要取出 **21 个**。" {
		t.Fatalf("answer = %q", parsed.FinalAnswer)
	}
	if parsed.InputTokens != 10163 || parsed.OutputTokens != 4873 || parsed.ReasoningTokens != 4660 {
		t.Fatalf("usage = %d %d %d", parsed.InputTokens, parsed.OutputTokens, parsed.ReasoningTokens)
	}
	if !questions.Grade(questions.Builtin()[0], parsed.FinalAnswer).OK {
		t.Fatal("expected grader to match independent 21")
	}
}

func TestParseEventsHandlesNestedEventShapes(t *testing.T) {
	input := strings.Join([]string{
		`{"event":"codex.item.completed","payload":{"item":{"role":"assistant","content":"答案：121 不是。最终是 21。"}}}`,
		`{"name":"codex.turn.completed","payload":{"usage":{"input_tokens":10,"output_tokens":20,"completion_tokens_details":{"reasoning_tokens":7}}}}`,
	}, "\n")

	parsed, err := parseEvents(strings.NewReader(input), i18n.ZH)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.FinalAnswer != "答案：121 不是。最终是 21。" {
		t.Fatalf("answer = %q", parsed.FinalAnswer)
	}
	if parsed.InputTokens != 10 || parsed.OutputTokens != 20 || parsed.ReasoningTokens != 7 {
		t.Fatalf("usage = %d %d %d", parsed.InputTokens, parsed.OutputTokens, parsed.ReasoningTokens)
	}
	if questions.Grade(questions.Builtin()[0], "121").OK {
		t.Fatal("grader should not match 21 inside 121")
	}
}

func TestParseEventsDetectsToolUse(t *testing.T) {
	input := `{"type":"item.completed","item":{"type":"tool_call","name":"shell_command"}}`
	parsed, err := parseEvents(strings.NewReader(input), i18n.ZH)
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.ToolUsed {
		t.Fatal("expected tool use detection")
	}
}

func TestParseEventsCapturesDiagnostics(t *testing.T) {
	input := strings.Join([]string{
		`{"type":"thread.started","thread_id":"thread_1"}`,
		`{"type":"turn.completed","usage":{"input_tokens":100,"cached_input_tokens":25,"output_tokens":40,"reasoning_output_tokens":7}}`,
	}, "\n")

	parsed, err := parseEvents(strings.NewReader(input), i18n.ZH)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.ThreadID != "thread_1" || parsed.CachedInputTokens != 25 || parsed.EventCount != 2 {
		t.Fatalf("diagnostics = %#v", parsed)
	}
	if !reflect.DeepEqual(parsed.EventTypes, []string{"thread.started", "turn.completed"}) {
		t.Fatalf("event types = %#v", parsed.EventTypes)
	}
}

func TestCodexArgsMatchUpstreamInvocation(t *testing.T) {
	want := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
		"--ephemeral",
		"-s", "read-only",
		"--disable", "memories",
		"-c", "model_reasoning_effort=xhigh",
		"-m", "gpt-5.5",
	}
	if got := codexArgs(" gpt-5.5 ", "xhigh"); !reflect.DeepEqual(got, want) {
		t.Fatalf("codexArgs with model = %#v", got)
	}

	withoutModel := codexArgs("", "medium")
	for i, arg := range withoutModel {
		if arg == "-m" {
			t.Fatalf("unexpected -m at index %d in %#v", i, withoutModel)
		}
		if strings.Contains(arg, "ignore-user-config") || strings.Contains(arg, "ignore-rules") {
			t.Fatalf("unexpected config-disabling arg %q in %#v", arg, withoutModel)
		}
	}

	withDefaultModel := codexArgs("default", "medium")
	for i, arg := range withDefaultModel {
		if arg == "-m" {
			t.Fatalf("unexpected -m for default model at index %d in %#v", i, withDefaultModel)
		}
	}
}

func TestDisplayModelNameUsesCodexConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`model = "gpt-5.4"`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := displayModelName("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "gpt-5.4" {
		t.Fatalf("display model = %q", got)
	}
}

func TestPreview(t *testing.T) {
	got := Preview("  hello\n\n世界  again  ", 8)
	if got != "hello 世界..." {
		t.Fatalf("Preview = %q", got)
	}
	if got := Preview("abc", 0); got != "" {
		t.Fatalf("Preview max 0 = %q", got)
	}
}

func TestRunOptionValidation(t *testing.T) {
	if _, err := Run(context.Background(), Options{ReasoningEffort: "bad"}); err == nil {
		t.Fatal("expected invalid effort error")
	}
	if _, err := Run(context.Background(), Options{Model: "x", ReasoningEffort: "medium", Tests: MaxTests + 1}); err == nil {
		t.Fatal("expected max tests error")
	}
	if _, err := Run(context.Background(), Options{Model: "x", ReasoningEffort: "medium", Tests: -1}); err == nil {
		t.Fatal("expected negative tests error")
	}
}
