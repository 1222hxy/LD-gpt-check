package report

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/haowang02/ld-gpt-check/internal/i18n"
	"github.com/haowang02/ld-gpt-check/internal/runner"
)

func PrintTable(s runner.Summary) {
	PrintTableWithLang(s, i18n.ZH)
}

func PrintTableWithLang(s runner.Summary, lang i18n.Lang) {
	l := i18n.New(lang)
	headers := []string{"Run", "Codex", "In Tok", "Out Tok", "Reason Tok", "Time(s)", "TPS", "OK"}
	widths := []int{3, 28, 6, 7, 10, 7, 4, 2}
	printRow(headers, widths)
	printRow([]string{"---", strings.Repeat("-", 28), "------", "-------", "----------", "-------", "----", "--"}, widths)
	for _, c := range s.Cases {
		ok := "✗"
		if c.OK {
			ok = "✓"
		}
		printRow([]string{
			strconv.Itoa(c.Index),
			Truncate(c.AnswerPreview, widths[1]),
			strconv.Itoa(c.InputTokens),
			strconv.Itoa(c.OutputTokens),
			strconv.Itoa(c.ReasoningTokens),
			fmt.Sprintf("%.1f", c.TimeSeconds),
			fmt.Sprintf("%.1f", c.TPS),
			ok,
		}, widths)
	}
	fmt.Printf(l.S("report_summary"), s.Correct, s.Tests, s.Accuracy, s.AvgTimeSeconds, s.AvgTPS)
}

func printRow(cols []string, widths []int) {
	for i, col := range cols {
		if i > 0 {
			fmt.Print("  ")
		}
		if i == 0 || i == 1 || i == len(cols)-1 {
			fmt.Print(PadRight(col, widths[i]))
		} else {
			fmt.Print(PadLeft(col, widths[i]))
		}
	}
	fmt.Println()
}

func PadRight(s string, width int) string {
	w := DisplayWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func PadLeft(s string, width int) string {
	w := DisplayWidth(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

func Truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if DisplayWidth(s) <= width {
		return s
	}
	ellipsis := "..."
	if width <= DisplayWidth(ellipsis) {
		return strings.Repeat(".", width)
	}
	var b strings.Builder
	used := 0
	limit := width - DisplayWidth(ellipsis)
	for _, r := range s {
		rw := runeWidth(r)
		if used+rw > limit {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	b.WriteString(ellipsis)
	return b.String()
}

func DisplayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

func runeWidth(r rune) int {
	if r == '\t' {
		return 4
	}
	if r >= 0x1100 &&
		(r <= 0x115f ||
			r == 0x2329 || r == 0x232a ||
			(r >= 0x2e80 && r <= 0xa4cf) ||
			(r >= 0xac00 && r <= 0xd7a3) ||
			(r >= 0xf900 && r <= 0xfaff) ||
			(r >= 0xfe10 && r <= 0xfe19) ||
			(r >= 0xfe30 && r <= 0xfe6f) ||
			(r >= 0xff00 && r <= 0xff60) ||
			(r >= 0xffe0 && r <= 0xffe6)) {
		return 2
	}
	return 1
}
