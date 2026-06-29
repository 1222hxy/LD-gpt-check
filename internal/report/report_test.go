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

func TestPrintWizardRunRecordIncludesConfigAndResults(t *testing.T) {
	var out bytes.Buffer
	longArgs := "--profile fair-test " + strings.Repeat("x", 100)
	PrintWizardRunRecord(&out, WizardRunRecord{
		Backend:          runner.BackendAPI,
		APIFormat:        runner.APIFormatOpenAIChat,
		Model:            "deepseek-reasoner",
		ModelAPIBaseURL:  "https://api.deepseek.com/v1",
		CodexStartupArgs: longArgs,
		ReasoningEffort:  "medium",
		Tests:            1,
		Timeout:          5,
		Upload:           true,
		Anonymous:        false,
		UploadStatus:     "已上传 run_123",
		UploadStatusOK:   true,
		Question: questions.Question{
			ID:      "remote_1",
			Version: "1",
			Title:   "远程题",
		},
		QuestionSource: "remote",
		Summary: runner.Summary{
			Tests:               1,
			Correct:             1,
			Accuracy:            100,
			AvgInputTokens:      10,
			AvgOutputTokens:     5,
			AvgReasoningTokens:  2,
			AvgTimeSeconds:      3.4,
			AvgTPS:              1.5,
			UploadSchemaVersion: 4,
		},
	}, i18n.ZH, false)
	text := out.String()
	for _, want := range []string{"向导运行记录", "API 模式", "OpenAI Chat Completions", "deepseek-reasoner", "https://api.deepseek.com/v1", "远程题", "remote_1", "100.0%", "1/1", "输入 10 / 输出 5 / 推理 2", "已上传 run_123"} {
		if !strings.Contains(text, want) {
			t.Fatalf("wizard record missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, longArgs) || !strings.Contains(text, "...") {
		t.Fatalf("long codex args should be truncated:\n%s", text)
	}
	if strings.Contains(text, "secret-key") {
		t.Fatalf("record should not include API keys:\n%s", text)
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }
