package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/1222hxy/LD-gpt-check/internal/i18n"
	"github.com/1222hxy/LD-gpt-check/internal/questions"
	"github.com/1222hxy/LD-gpt-check/internal/runner"
)

func TestDisplayWidthAndTruncate(t *testing.T) {
	if got := DisplayWidth("a糖b"); got != 4 {
		t.Fatalf("DisplayWidth = %d", got)
	}
	got := Truncate("最少需要取出21个", 8)
	if DisplayWidth(got) > 8 {
		t.Fatalf("Truncate width = %d, value %q", DisplayWidth(got), got)
	}
	if got := Truncate("abcdef", 2); got != ".." {
		t.Fatalf("Truncate tiny width = %q", got)
	}
	if got := Truncate("abcdef", 0); got != "" {
		t.Fatalf("Truncate zero width = %q", got)
	}
}

func TestDisplayWidthIgnoresANSIColor(t *testing.T) {
	colored := Colorize("PASS", colorGreen, true)
	if got := StripANSI(colored); got != "PASS" {
		t.Fatalf("StripANSI = %q", got)
	}
	if got := DisplayWidth(colored); got != 4 {
		t.Fatalf("DisplayWidth colored = %d", got)
	}
	if got := DisplayWidth(PadRight(colored, 6)); got != 6 {
		t.Fatalf("PadRight colored width = %d", got)
	}
}

func TestPrintSummaryPanelIncludesMetrics(t *testing.T) {
	var out bytes.Buffer
	PrintSummaryPanel(&out, runner.Summary{
		Tests:              5,
		Correct:            3,
		Accuracy:           60,
		AvgInputTokens:     100,
		AvgReasoningTokens: 40,
		AvgTimeSeconds:     12.5,
		AvgTPS:             9.8,
	}, i18n.ZH, false)
	text := out.String()
	for _, want := range []string{"运行概览", "正确率", "60.0%", "3/5", "12.5s", "9.8", "100", "40"} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary panel missing %q:\n%s", want, text)
		}
	}
}

func TestPrintProgressErrorDoesNotRepeatQuestionPrompt(t *testing.T) {
	var out bytes.Buffer
	progress := PrintProgress(&out, i18n.ZH, "gpt-5.4", "medium", false)
	progress(runner.ProgressEvent{
		Type:    runner.ProgressCaseError,
		Current: 1,
		Total:   1,
		Question: questions.Question{
			ID:     "q1",
			Title:  "原题标题",
			Prompt: "这里是完整原题内容",
		},
		Error: assertError("boom"),
	})
	text := out.String()
	if !strings.Contains(text, "运行失败") {
		t.Fatalf("progress output missing failure:\n%s", text)
	}
	if strings.Contains(text, "这里是完整原题内容") || strings.Contains(text, "失败题目") {
		t.Fatalf("error output should not repeat question prompt:\n%s", text)
	}
}

func TestPrintProgressStartDoesNotPrintQuestionPrompt(t *testing.T) {
	var out bytes.Buffer
	progress := PrintProgress(&out, i18n.ZH, "gpt-5.4", "medium", false)
	progress(runner.ProgressEvent{
		Type:      runner.ProgressCaseStart,
		Current:   1,
		Total:     1,
		TestIndex: 1,
		Question: questions.Question{
			ID:     "q1",
			Title:  "原题标题",
			Prompt: "这里是完整原题内容",
		},
	})
	text := out.String()
	if !strings.Contains(text, "正在运行") {
		t.Fatalf("progress output missing start:\n%s", text)
	}
	if strings.Contains(text, "测试题目：原题标题") || strings.Contains(text, "这里是完整原题内容") {
		t.Fatalf("case start should not print question prompt:\n%s", text)
	}
}

func TestPrintQuestionPromptsPrintsSelectedQuestion(t *testing.T) {
	var out bytes.Buffer
	PrintQuestionPrompts(&out, i18n.ZH, []questions.Question{{
		ID:     "q1",
		Title:  "原题标题",
		Prompt: "这里是完整原题内容",
	}}, false)
	text := out.String()
	for _, want := range []string{"测试题目：原题标题", "这里是完整原题内容"} {
		if !strings.Contains(text, want) {
			t.Fatalf("question output missing %q:\n%s", want, text)
		}
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }
