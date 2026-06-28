package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/haowang02/ld-gpt-check/internal/i18n"
	"github.com/haowang02/ld-gpt-check/internal/runner"
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
