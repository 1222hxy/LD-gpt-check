package report

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/1222hxy/LD-gpt-check/internal/i18n"
	"github.com/1222hxy/LD-gpt-check/internal/runner"
)

func PrintTable(s runner.Summary) {
	PrintTableWithLang(s, i18n.ZH)
}

func PrintTableWithLang(s runner.Summary, lang i18n.Lang) {
	PrintTableWithWriter(os.Stdout, s, lang, ColorEnabled(os.Stdout))
}

func PrintTableWithLangColor(s runner.Summary, lang i18n.Lang, color bool) {
	PrintTableWithWriter(os.Stdout, s, lang, color)
}

func PrintTableWithWriter(w io.Writer, s runner.Summary, lang i18n.Lang, color bool) {
	headers := []string{"Run", "Codex", "In Tok", "Out Tok", "Reason Tok", "Time(s)", "TPS", "OK"}
	widths := []int{3, 28, 6, 7, 10, 7, 4, 2}
	printRow(w, colorizeHeaders(headers, color), widths)
	printRow(w, []string{"---", strings.Repeat("-", 28), "------", "-------", "----------", "-------", "----", "--"}, widths)
	for _, c := range s.Cases {
		ok := "✗"
		if c.OK {
			ok = "✓"
		}
		printRow(w, []string{
			strconv.Itoa(c.Index),
			Truncate(c.AnswerPreview, widths[1]),
			strconv.Itoa(c.InputTokens),
			strconv.Itoa(c.OutputTokens),
			strconv.Itoa(c.ReasoningTokens),
			fmt.Sprintf("%.1f", c.TimeSeconds),
			fmt.Sprintf("%.1f", c.TPS),
			Colorize(ok, statusColor(c.OK), color),
		}, widths)
	}
	PrintSummaryPanel(w, s, lang, color)
}

func PrintProgress(w io.Writer, lang i18n.Lang, model, effort string, color bool) func(runner.ProgressEvent) {
	l := i18n.New(lang)
	if strings.TrimSpace(model) == "" {
		model = l.S("model_local_config")
	}
	return func(ev runner.ProgressEvent) {
		switch ev.Type {
		case runner.ProgressStarted:
			fmt.Fprintln(w, Colorize(l.S("run_status_start", model, effort, ev.Total), colorCyan, color))
		case runner.ProgressCaseStart:
			fmt.Fprintln(w, Colorize(l.S("run_status_case_start", ev.Current, ev.Total, ev.Question.ID, ev.TestIndex), colorBlue, color))
		case runner.ProgressCaseDone:
			label := Colorize("FAIL", colorRed, color)
			if ev.CaseResult.OK {
				label = Colorize("PASS", colorGreen, color)
			}
			fmt.Fprintln(w, l.S("run_status_case_done", ev.Current, ev.Total, label, ev.CaseResult.TimeSeconds, ev.CaseResult.TPS))
		case runner.ProgressCaseError:
			fmt.Fprintln(w, Colorize(l.S("run_status_case_error", ev.Current, ev.Total, ev.Error), colorRed, color))
		}
	}
}

func PrintBanner(w io.Writer, title, subtitle string, color bool) {
	line := strings.Repeat("=", max(44, DisplayWidth(title)+10))
	fmt.Fprintln(w, Colorize(line, colorCyan, color))
	fmt.Fprintln(w, Colorize(title, colorBold+";"+colorCyan, color))
	if strings.TrimSpace(subtitle) != "" {
		fmt.Fprintln(w, Colorize(subtitle, colorGray, color))
	}
	fmt.Fprintln(w, Colorize(line, colorCyan, color))
}

func PrintSection(w io.Writer, index int, title string, color bool) {
	label := fmt.Sprintf("[%d] %s", index, title)
	fmt.Fprintln(w)
	fmt.Fprintln(w, Colorize(label, colorBold+";"+colorBlue, color))
	fmt.Fprintln(w, Colorize(strings.Repeat("-", max(24, DisplayWidth(label))), colorBlue, color))
}

func PrintInfo(w io.Writer, label, value string, color bool) {
	fmt.Fprintf(w, "%s: %s\n", Colorize(label, colorCyan, color), Colorize(value, colorBold, color))
}

func PrintSuccess(w io.Writer, msg string, color bool) {
	fmt.Fprintln(w, Colorize(msg, colorGreen, color))
}

func PrintWarning(w io.Writer, msg string, color bool) {
	fmt.Fprintln(w, Colorize(msg, colorYellow, color))
}

func Muted(s string, color bool) string {
	return Colorize(s, colorGray, color)
}

func PrintSummaryPanel(w io.Writer, s runner.Summary, lang i18n.Lang, color bool) {
	l := i18n.New(lang)
	title := l.S("report_panel_title")
	lineWidth := max(58, DisplayWidth(title)+8)
	fmt.Fprintln(w)
	fmt.Fprintln(w, Colorize("┌─ "+title+" "+strings.Repeat("─", max(1, lineWidth-DisplayWidth(title)-4))+"┐", colorCyan, color))
	fmt.Fprintf(w, "│ %s  %s  %s  %s │\n",
		PadRight(Colorize(l.S("report_metric_accuracy"), colorGray, color), 8),
		PadRight(Colorize(fmt.Sprintf("%.1f%%", s.Accuracy), colorSummary(s), color), 10),
		PadRight(Colorize(l.S("report_metric_correct"), colorGray, color), 8),
		PadRight(Colorize(fmt.Sprintf("%d/%d", s.Correct, s.Tests), colorSummary(s), color), lineWidth-34),
	)
	fmt.Fprintf(w, "│ %s  %s  %s  %s │\n",
		PadRight(Colorize(l.S("report_metric_time"), colorGray, color), 8),
		PadRight(Colorize(fmt.Sprintf("%.1fs", s.AvgTimeSeconds), colorYellow, color), 10),
		PadRight(Colorize(l.S("report_metric_tps"), colorGray, color), 8),
		PadRight(Colorize(fmt.Sprintf("%.1f", s.AvgTPS), colorYellow, color), lineWidth-34),
	)
	fmt.Fprintf(w, "│ %s  %s  %s  %s │\n",
		PadRight(Colorize(l.S("report_metric_input"), colorGray, color), 8),
		PadRight(Colorize(fmt.Sprintf("%.0f", s.AvgInputTokens), colorBlue, color), 10),
		PadRight(Colorize(l.S("report_metric_reason"), colorGray, color), 8),
		PadRight(Colorize(fmt.Sprintf("%.0f", s.AvgReasoningTokens), colorMagenta, color), lineWidth-34),
	)
	fmt.Fprintln(w, Colorize("└"+strings.Repeat("─", lineWidth)+"┘", colorCyan, color))
	fmt.Fprintf(w, Colorize(l.S("report_summary"), colorSummary(s), color), s.Correct, s.Tests, s.Accuracy, s.AvgTimeSeconds, s.AvgTPS)
}

func printRow(w io.Writer, cols []string, widths []int) {
	for i, col := range cols {
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		if i == 0 || i == 1 || i == len(cols)-1 {
			fmt.Fprint(w, PadRight(col, widths[i]))
		} else {
			fmt.Fprint(w, PadLeft(col, widths[i]))
		}
	}
	fmt.Fprintln(w)
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
	for _, r := range StripANSI(s) {
		w += runeWidth(r)
	}
	return w
}

func StripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inEscape {
			if c >= '@' && c <= '~' {
				inEscape = false
			}
			continue
		}
		if c == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func ColorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}

func Colorize(s, code string, enabled bool) string {
	if !enabled || code == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func colorizeHeaders(headers []string, enabled bool) []string {
	out := make([]string, len(headers))
	for i, h := range headers {
		out[i] = Colorize(h, colorBold, enabled)
	}
	return out
}

func statusColor(ok bool) string {
	if ok {
		return colorGreen
	}
	return colorRed
}

func colorSummary(s runner.Summary) string {
	if s.Tests > 0 && s.Correct == s.Tests {
		return colorGreen
	}
	if s.Correct == 0 {
		return colorRed
	}
	return colorYellow
}

const (
	colorBold    = "1"
	colorRed     = "31"
	colorGreen   = "32"
	colorYellow  = "33"
	colorBlue    = "34"
	colorMagenta = "35"
	colorCyan    = "36"
	colorGray    = "90"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
