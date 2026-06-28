package runner

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/1222hxy/LD-gpt-check/internal/i18n"
	"github.com/1222hxy/LD-gpt-check/internal/questions"
	"github.com/1222hxy/LD-gpt-check/internal/system"
)

const (
	DefaultTimeout = 30 * time.Minute
	DefaultTests   = 5
	MaxTests       = 100
)

type Options struct {
	Model           string
	ReasoningEffort string
	Tests           int
	Timeout         time.Duration
	Lang            i18n.Lang
	QuestionSuite   string
	Questions       []questions.Question
	Progress        func(ProgressEvent)
}

type ProgressEvent struct {
	Type       ProgressEventType
	Current    int
	Total      int
	Question   questions.Question
	TestIndex  int
	CaseResult CaseResult
	Error      error
}

type ProgressEventType string

const (
	ProgressStarted   ProgressEventType = "started"
	ProgressCaseStart ProgressEventType = "case_start"
	ProgressCaseDone  ProgressEventType = "case_done"
	ProgressCaseError ProgressEventType = "case_error"
)

type CaseResult struct {
	Index                  int      `json:"index"`
	QuestionID             string   `json:"question_id"`
	QuestionVersion        string   `json:"question_version"`
	QuestionTitle          string   `json:"question_title"`
	OK                     bool     `json:"ok"`
	Status                 string   `json:"status"`
	ExpectedAnswer         string   `json:"expected_answer"`
	ExtractedAnswer        string   `json:"extracted_answer"`
	FailureReason          string   `json:"failure_reason,omitempty"`
	AnswerPreview          string   `json:"answer_preview"`
	AnswerPreviewTruncated bool     `json:"answer_preview_truncated"`
	AnswerHash             string   `json:"answer_hash,omitempty"`
	FullAnswer             string   `json:"-"`
	InputTokens            int      `json:"input_tokens"`
	CachedInputTokens      int      `json:"cached_input_tokens"`
	OutputTokens           int      `json:"output_tokens"`
	ReasoningTokens        int      `json:"reasoning_tokens"`
	TotalTokens            int      `json:"total_tokens"`
	TimeSeconds            float64  `json:"time_seconds"`
	TPS                    float64  `json:"tps"`
	CodexThreadID          string   `json:"codex_thread_id,omitempty"`
	EventCount             int      `json:"event_count"`
	EventTypes             []string `json:"event_types,omitempty"`
	ToolEventDetected      bool     `json:"tool_event_detected"`
	AnswerChars            int      `json:"answer_chars"`
	Error                  string   `json:"error,omitempty"`
	ErrorCode              string   `json:"error_code,omitempty"`
	StartedAt              string   `json:"started_at,omitempty"`
	FinishedAt             string   `json:"finished_at,omitempty"`
	TimeoutSeconds         float64  `json:"timeout_seconds,omitempty"`
}

type Summary struct {
	Model                 string            `json:"model"`
	ReasoningEffort       string            `json:"reasoning_effort"`
	Tests                 int               `json:"tests"`
	Correct               int               `json:"correct"`
	Accuracy              float64           `json:"accuracy"`
	AvgInputTokens        float64           `json:"avg_input_tokens"`
	AvgOutputTokens       float64           `json:"avg_output_tokens"`
	AvgReasoningTokens    float64           `json:"avg_reason_tokens"`
	AvgTimeSeconds        float64           `json:"avg_time_seconds"`
	AvgTPS                float64           `json:"avg_tps"`
	StartedAt             string            `json:"started_at,omitempty"`
	FinishedAt            string            `json:"finished_at,omitempty"`
	DurationSeconds       float64           `json:"duration_seconds,omitempty"`
	QuestionSuite         string            `json:"question_suite,omitempty"`
	ClientTimezone        string            `json:"client_timezone,omitempty"`
	UploadSchemaVersion   int               `json:"upload_schema_version"`
	CodexModelSource      string            `json:"codex_model_source"`
	CodexModelProvider    string            `json:"codex_model_provider,omitempty"`
	CodexProviderHost     string            `json:"codex_provider_host,omitempty"`
	CodexProviderBaseURL  string            `json:"codex_provider_base_url,omitempty"`
	CodexSandbox          string            `json:"codex_sandbox"`
	CodexEphemeral        bool              `json:"codex_ephemeral"`
	CodexSkipGitRepoCheck bool              `json:"codex_skip_git_repo_check"`
	CodexDisabledFeatures []string          `json:"codex_disabled_features,omitempty"`
	CodexInvocation       string            `json:"codex_invocation,omitempty"`
	Questions             []QuestionSummary `json:"questions"`
	Cases                 []CaseResult      `json:"cases"`
}

type ParsedEvents struct {
	FinalAnswer       string
	InputTokens       int
	CachedInputTokens int
	OutputTokens      int
	ReasoningTokens   int
	ThreadID          string
	EventCount        int
	EventTypes        []string
	ToolUsed          bool
}

type QuestionSummary struct {
	QuestionID      string  `json:"question_id"`
	QuestionVersion string  `json:"question_version"`
	QuestionTitle   string  `json:"question_title"`
	GraderType      string  `json:"grader_type"`
	ExpectedAnswer  string  `json:"expected_answer"`
	PromptHash      string  `json:"prompt_hash"`
	Tests           int     `json:"tests"`
	Correct         int     `json:"correct"`
	Accuracy        float64 `json:"accuracy"`
	AvgInputTokens  float64 `json:"avg_input_tokens"`
	AvgOutputTokens float64 `json:"avg_output_tokens"`
	AvgReasonTokens float64 `json:"avg_reason_tokens"`
	AvgTimeSeconds  float64 `json:"avg_time_seconds"`
	AvgTPS          float64 `json:"avg_tps"`
}

func ValidReasoningEffort(v string) bool {
	switch v {
	case "low", "medium", "high", "xhigh":
		return true
	default:
		return false
	}
}

func Run(ctx context.Context, opts Options) (Summary, error) {
	l := i18n.New(opts.Lang)
	if opts.ReasoningEffort == "" {
		opts.ReasoningEffort = "medium"
	}
	if !ValidReasoningEffort(opts.ReasoningEffort) {
		return Summary{}, fmt.Errorf("%s", l.S("runner_bad_effort", opts.ReasoningEffort))
	}
	if opts.Tests < 0 {
		return Summary{}, fmt.Errorf("%s", l.S("runner_tests_positive"))
	}
	if opts.Tests == 0 {
		opts.Tests = DefaultTests
	}
	if opts.Tests > MaxTests {
		return Summary{}, fmt.Errorf("%s", l.S("runner_tests_max", MaxTests))
	}
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultTimeout
	}
	if len(opts.Questions) == 0 {
		opts.Questions = questions.Builtin()
	}
	displayModel, err := displayModelName(opts.Model)
	if err != nil {
		return Summary{}, err
	}
	codex, err := system.CodexPath()
	if err != nil {
		return Summary{}, errors.New(l.S("runner_codex_missing"))
	}

	total := opts.Tests * len(opts.Questions)
	emitProgress(opts.Progress, ProgressEvent{Type: ProgressStarted, Total: total})
	results := make([]CaseResult, 0, opts.Tests*len(opts.Questions))
	current := 0
	for _, q := range opts.Questions {
		for i := 1; i <= opts.Tests; i++ {
			current++
			emitProgress(opts.Progress, ProgressEvent{Type: ProgressCaseStart, Current: current, Total: total, Question: q, TestIndex: i})
			res, err := runOne(ctx, codex, opts, q, i)
			if err != nil {
				emitProgress(opts.Progress, ProgressEvent{Type: ProgressCaseError, Current: current, Total: total, Question: q, TestIndex: i, Error: err})
				return Summary{}, err
			}
			emitProgress(opts.Progress, ProgressEvent{Type: ProgressCaseDone, Current: current, Total: total, Question: q, TestIndex: i, CaseResult: res})
			results = append(results, res)
		}
	}
	return summarize(opts, displayModel, results), nil
}

func emitProgress(fn func(ProgressEvent), ev ProgressEvent) {
	if fn != nil {
		fn(ev)
	}
}

func runOne(ctx context.Context, codex string, opts Options, q questions.Question, index int) (CaseResult, error) {
	runCtx := ctx
	cancel := func() {}
	if opts.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
	}
	defer cancel()
	args := codexArgs(opts.Model, opts.ReasoningEffort)
	cmd := exec.CommandContext(runCtx, codex, args...)
	cmd.Stdin = strings.NewReader(q.Prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return CaseResult{}, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return CaseResult{}, err
	}

	parsed, parseErr := parseEvents(stdout, opts.Lang)
	if parseErr != nil {
		cancel()
		_ = cmd.Wait()
		return CaseResult{}, parseErr
	}
	waitErr := cmd.Wait()
	if parsed.ToolUsed {
		return CaseResult{}, errors.New(i18n.New(opts.Lang).S("runner_tool_used"))
	}
	if waitErr != nil {
		msg := strings.TrimSpace(stderr.String())
		if runCtx.Err() == context.Canceled {
			return CaseResult{}, context.Canceled
		}
		if runCtx.Err() == context.DeadlineExceeded {
			return CaseResult{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_codex_timeout", opts.Timeout))
		}
		if msg == "" {
			msg = waitErr.Error()
		}
		return CaseResult{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_codex_failed", msg))
	}

	finished := time.Now()
	elapsed := finished.Sub(start).Seconds()
	tps := 0.0
	if elapsed > 0 {
		tps = float64(parsed.OutputTokens) / elapsed
	}
	if parsed.FinalAnswer == "" {
		parsed.FinalAnswer = "(no final assistant message found)"
	}
	previewMax := 300
	grade := questions.Grade(q, parsed.FinalAnswer)
	return CaseResult{
		Index:                  index,
		QuestionID:             q.ID,
		QuestionVersion:        q.Version,
		QuestionTitle:          q.Title,
		OK:                     grade.OK,
		Status:                 "completed",
		ExpectedAnswer:         grade.ExpectedAnswer,
		ExtractedAnswer:        grade.ExtractedAnswer,
		FailureReason:          grade.FailureReason,
		AnswerPreview:          Preview(parsed.FinalAnswer, previewMax),
		AnswerPreviewTruncated: PreviewTruncated(parsed.FinalAnswer, previewMax),
		AnswerHash:             SHA256Hex(parsed.FinalAnswer),
		FullAnswer:             parsed.FinalAnswer,
		InputTokens:            parsed.InputTokens,
		CachedInputTokens:      parsed.CachedInputTokens,
		OutputTokens:           parsed.OutputTokens,
		ReasoningTokens:        parsed.ReasoningTokens,
		TotalTokens:            parsed.InputTokens + parsed.OutputTokens,
		TimeSeconds:            elapsed,
		TPS:                    tps,
		CodexThreadID:          parsed.ThreadID,
		EventCount:             parsed.EventCount,
		EventTypes:             parsed.EventTypes,
		ToolEventDetected:      parsed.ToolUsed,
		AnswerChars:            utf8.RuneCountInString(parsed.FinalAnswer),
		StartedAt:              start.UTC().Format(time.RFC3339Nano),
		FinishedAt:             finished.UTC().Format(time.RFC3339Nano),
		TimeoutSeconds:         opts.Timeout.Seconds(),
	}, nil
}

func codexArgs(model, effort string) []string {
	args := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
		"--ephemeral",
		"-s", "read-only",
		"--disable", "memories",
		"-c", "model_reasoning_effort=" + effort,
	}
	if system.ConcreteCodexModel(model) {
		args = append(args, "-m", strings.TrimSpace(model))
	}
	return args
}

func parseEvents(r io.Reader, lang i18n.Lang) (ParsedEvents, error) {
	var out ParsedEvents
	eventTypes := make(map[string]struct{})
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		ev, err := parseEventLine(line)
		if err != nil {
			continue
		}
		out.EventCount++
		if name := eventName(ev); name != "" {
			eventTypes[name] = struct{}{}
		}
		if out.ThreadID == "" {
			out.ThreadID = stringField(ev, "thread_id")
		}
		if eventUsesTool(ev) {
			out.ToolUsed = true
		}
		if isEvent(ev, "item.completed") {
			if msg := extractAgentMessage(ev); msg != "" {
				out.FinalAnswer = msg
			}
		}
		if isEvent(ev, "turn.completed") {
			in, cached, output, reason := extractUsage(ev)
			if in > 0 || cached > 0 || output > 0 || reason > 0 {
				out.InputTokens, out.CachedInputTokens, out.OutputTokens, out.ReasoningTokens = in, cached, output, reason
			}
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return ParsedEvents{}, errors.New(i18n.New(lang).S("runner_event_too_large"))
		}
		return ParsedEvents{}, err
	}
	out.EventTypes = sortedKeys(eventTypes)
	return out, nil
}

func summarize(opts Options, displayModel string, cases []CaseResult) Summary {
	var correct, in, out, reason int
	var secs, tps float64
	for _, c := range cases {
		if c.OK {
			correct++
		}
		in += c.InputTokens
		out += c.OutputTokens
		reason += c.ReasoningTokens
		secs += c.TimeSeconds
		tps += c.TPS
	}
	n := float64(len(cases))
	s := Summary{
		Model:                 displayModel,
		ReasoningEffort:       opts.ReasoningEffort,
		Tests:                 len(cases),
		Correct:               correct,
		QuestionSuite:         strings.TrimSpace(opts.QuestionSuite),
		ClientTimezone:        time.Now().Format("-07:00"),
		UploadSchemaVersion:   4,
		CodexSandbox:          "read-only",
		CodexEphemeral:        true,
		CodexSkipGitRepoCheck: true,
		CodexDisabledFeatures: []string{"memories"},
		CodexInvocation:       sanitizedInvocation(opts.Model, opts.ReasoningEffort),
		Questions:             summarizeQuestions(opts.Questions, cases),
		Cases:                 cases,
	}
	applyCodexConfigMetadata(&s, opts.Model)
	if n > 0 {
		s.Accuracy = float64(correct) * 100 / n
		s.AvgInputTokens = float64(in) / n
		s.AvgOutputTokens = float64(out) / n
		s.AvgReasoningTokens = float64(reason) / n
		s.AvgTimeSeconds = secs / n
		s.AvgTPS = tps / n
		s.StartedAt = cases[0].StartedAt
		s.FinishedAt = cases[len(cases)-1].FinishedAt
		if start, err := time.Parse(time.RFC3339Nano, s.StartedAt); err == nil {
			if finish, err := time.Parse(time.RFC3339Nano, s.FinishedAt); err == nil && finish.After(start) {
				s.DurationSeconds = finish.Sub(start).Seconds()
			}
		}
	}
	return s
}

func SHA256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func applyCodexConfigMetadata(s *Summary, requestedModel string) {
	info, err := system.CodexConfigInfo()
	if err == nil {
		s.CodexModelProvider = info.ModelProvider
		s.CodexProviderHost = info.ProviderHost
		s.CodexProviderBaseURL = info.ProviderBaseURL
	}
	switch {
	case system.ConcreteCodexModel(requestedModel):
		s.CodexModelSource = "explicit"
	case system.ConcreteCodexModel(s.Model):
		s.CodexModelSource = "codex_config"
	default:
		s.CodexModelSource = "unknown"
	}
}

func sanitizedInvocation(model, effort string) string {
	args := codexArgs(model, effort)
	safe := struct {
		Command              string   `json:"command"`
		Args                 []string `json:"args"`
		PromptFromStdin      bool     `json:"prompt_from_stdin"`
		Sandbox              string   `json:"sandbox"`
		Ephemeral            bool     `json:"ephemeral"`
		SkipGitRepoCheck     bool     `json:"skip_git_repo_check"`
		DisabledFeatures     []string `json:"disabled_features"`
		ModelReasoningEffort string   `json:"model_reasoning_effort"`
	}{
		Command:              "codex",
		Args:                 args,
		PromptFromStdin:      true,
		Sandbox:              "read-only",
		Ephemeral:            true,
		SkipGitRepoCheck:     true,
		DisabledFeatures:     []string{"memories"},
		ModelReasoningEffort: effort,
	}
	b, err := json.Marshal(safe)
	if err != nil {
		return ""
	}
	return string(b)
}

func displayModelName(requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if system.ConcreteCodexModel(requested) {
		return requested, nil
	}
	configured, err := system.CodexConfiguredModel()
	if err != nil {
		return "", err
	}
	if system.ConcreteCodexModel(configured) {
		return configured, nil
	}
	return "", nil
}

func summarizeQuestions(qs []questions.Question, cases []CaseResult) []QuestionSummary {
	out := make([]QuestionSummary, 0, len(qs))
	for _, q := range qs {
		var selected []CaseResult
		for _, c := range cases {
			if c.QuestionID == q.ID && c.QuestionVersion == q.Version {
				selected = append(selected, c)
			}
		}
		var correct, in, outTok, reason int
		var secs, tps float64
		for _, c := range selected {
			if c.OK {
				correct++
			}
			in += c.InputTokens
			outTok += c.OutputTokens
			reason += c.ReasoningTokens
			secs += c.TimeSeconds
			tps += c.TPS
		}
		n := float64(len(selected))
		item := QuestionSummary{
			QuestionID:      q.ID,
			QuestionVersion: q.Version,
			QuestionTitle:   q.Title,
			GraderType:      q.Grader.Type,
			ExpectedAnswer:  questions.Grade(q, "").ExpectedAnswer,
			PromptHash:      questions.PromptHash(q.Prompt),
			Tests:           len(selected),
			Correct:         correct,
		}
		if n > 0 {
			item.Accuracy = float64(correct) * 100 / n
			item.AvgInputTokens = float64(in) / n
			item.AvgOutputTokens = float64(outTok) / n
			item.AvgReasonTokens = float64(reason) / n
			item.AvgTimeSeconds = secs / n
			item.AvgTPS = tps / n
		}
		out = append(out, item)
	}
	return out
}

func isEvent(ev map[string]any, want string) bool {
	for _, k := range []string{"type", "event", "name"} {
		if s, _ := ev[k].(string); s == want || strings.HasSuffix(s, "."+want) {
			return true
		}
	}
	return false
}

func eventName(ev map[string]any) string {
	for _, k := range []string{"type", "event", "name"} {
		if s := stringField(ev, k); s != "" {
			return s
		}
	}
	return ""
}

func stringField(obj map[string]any, key string) string {
	if s, _ := obj[key].(string); s != "" {
		return s
	}
	return ""
}

func parseEventLine(line string) (map[string]any, error) {
	dec := json.NewDecoder(strings.NewReader(line))
	dec.UseNumber()
	var ev map[string]any
	if err := dec.Decode(&ev); err != nil {
		return nil, err
	}
	return ev, nil
}

func eventUsesTool(ev map[string]any) bool {
	var walk func(any) bool
	walk = func(v any) bool {
		switch t := v.(type) {
		case map[string]any:
			for k, v := range t {
				if k == "type" || k == "event" || k == "name" {
					if s, ok := v.(string); ok && toolMarker(s) {
						return true
					}
				}
				if walk(v) {
					return true
				}
			}
		case []any:
			for _, v := range t {
				if walk(v) {
					return true
				}
			}
		}
		return false
	}
	return walk(ev)
}

func toolMarker(s string) bool {
	s = strings.ToLower(s)
	for _, marker := range []string{"tool_call", "function_call", "exec_command", "shell_command", "command_execution"} {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

func extractUsage(ev map[string]any) (int, int, int, int) {
	for _, obj := range candidateObjects(ev, "usage") {
		in := intField(obj, "input_tokens")
		cached := intField(obj, "cached_input_tokens")
		out := intField(obj, "output_tokens")
		reason := intField(obj, "reasoning_output_tokens")
		if reason == 0 {
			reason = intField(obj, "reasoning_tokens")
		}
		if reason == 0 {
			reason = nestedIntField(obj, "output_tokens_details", "reasoning_tokens")
		}
		if reason == 0 {
			reason = nestedIntField(obj, "completion_tokens_details", "reasoning_tokens")
		}
		if in > 0 || cached > 0 || out > 0 || reason > 0 {
			return in, cached, out, reason
		}
	}
	return 0, 0, 0, 0
}

func extractAgentMessage(ev map[string]any) string {
	if item, ok := ev["item"].(map[string]any); ok {
		if msg := messageFromItem(item); msg != "" {
			return msg
		}
	}
	for _, obj := range candidateObjects(ev, "item") {
		if msg := messageFromItem(obj); msg != "" {
			return msg
		}
	}
	return ""
}

func messageFromItem(item map[string]any) string {
	if role, _ := item["role"].(string); role != "" && role != "assistant" {
		return ""
	}
	if typ, _ := item["type"].(string); typ == "reasoning" || typ == "tool_call" || typ == "function_call" {
		return ""
	}
	if s, _ := item["text"].(string); s != "" {
		return s
	}
	if s, _ := item["message"].(string); s != "" {
		return s
	}
	if s, _ := item["content"].(string); s != "" {
		return s
	}
	if msg := textFromValue(item["content"]); msg != "" {
		return msg
	}
	return ""
}

func textFromValue(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		var parts []string
		for _, v := range t {
			if s := textFromValue(v); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	case map[string]any:
		if typ, _ := t["type"].(string); typ == "reasoning" || typ == "tool_call" || typ == "function_call" {
			return ""
		}
		for _, key := range []string{"text", "content", "message", "output_text"} {
			if s := textFromValue(t[key]); s != "" {
				return s
			}
		}
	}
	return ""
}

func candidateObjects(v any, key string) []map[string]any {
	var out []map[string]any
	var walk func(any)
	walk = func(x any) {
		switch t := x.(type) {
		case map[string]any:
			if obj, ok := t[key].(map[string]any); ok {
				out = append(out, obj)
			}
			for _, v := range t {
				walk(v)
			}
		case []any:
			for _, v := range t {
				walk(v)
			}
		}
	}
	walk(v)
	return out
}

func intField(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	case string:
		i, _ := strconv.Atoi(v)
		return i
	default:
		return 0
	}
}

func nestedIntField(m map[string]any, objectKey, intKey string) int {
	if obj, ok := m[objectKey].(map[string]any); ok {
		return intField(obj, intKey)
	}
	return 0
}

func Preview(s string, maxRunes int) string {
	s = strings.Join(strings.Fields(s), " ")
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "..."
}

func PreviewTruncated(s string, maxRunes int) bool {
	s = strings.Join(strings.Fields(s), " ")
	return maxRunes > 0 && utf8.RuneCountInString(s) > maxRunes
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
