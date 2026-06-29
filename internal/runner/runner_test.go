package runner

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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
	got, err := displayModelName(Options{Lang: i18n.ZH}, BackendCodex)
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

func TestRunWithOpenAIChatAPIBackend(t *testing.T) {
	q := apiTestQuestion()
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/codex/v1/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-5.4" || body["reasoning_effort"] != "medium" {
			t.Fatalf("body = %#v", body)
		}
		messages, _ := body["messages"].([]any)
		if len(messages) != 2 {
			t.Fatalf("messages = %#v", body["messages"])
		}
		return jsonResponse(http.StatusOK, map[string]any{
			"id": "chatcmpl_1",
			"choices": []map[string]any{{
				"message": map[string]any{"content": "答案是 21。"},
			}},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"completion_tokens_details": map[string]any{
					"reasoning_tokens": 2,
				},
			},
		})
	})

	summary, err := Run(context.Background(), Options{
		Model:            "gpt-5.4",
		ReasoningEffort:  "medium",
		Tests:            1,
		Timeout:          5 * time.Second,
		Lang:             i18n.ZH,
		Backend:          BackendAPI,
		APIFormat:        APIFormatOpenAIChat,
		ModelAPIBaseURL:  "https://api.test/codex/v1/chat/completions?token=secret#fragment",
		ModelAPIKey:      "test-key",
		APIHTTPTransport: transport,
		Questions:        []questions.Question{q},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Cases[0].OK || summary.Cases[0].InputTokens != 10 || summary.Cases[0].OutputTokens != 5 || summary.Cases[0].ReasoningTokens != 2 {
		t.Fatalf("case = %#v", summary.Cases[0])
	}
	if summary.CodexProviderBaseURL != "https://api.test/codex/v1" || summary.CodexProviderHost != "api.test" || strings.Contains(summary.CodexInvocation, "test-key") {
		t.Fatalf("metadata = base=%q host=%q invocation=%s", summary.CodexProviderBaseURL, summary.CodexProviderHost, summary.CodexInvocation)
	}
}

func TestRunWithOpenAIResponsesAPIBackend(t *testing.T) {
	q := apiTestQuestion()
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer response-key" {
			t.Fatalf("authorization = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["input"] != q.Prompt || body["instructions"] == "" {
			t.Fatalf("body = %#v", body)
		}
		return jsonResponse(http.StatusOK, map[string]any{
			"id":          "resp_1",
			"output_text": "最终答案：21",
			"usage": map[string]any{
				"input_tokens":  11,
				"output_tokens": 6,
				"output_tokens_details": map[string]any{
					"reasoning_tokens": 3,
				},
			},
		})
	})

	summary, err := Run(context.Background(), Options{
		Model:            "gpt-5.4",
		ReasoningEffort:  "high",
		Tests:            1,
		Timeout:          5 * time.Second,
		Lang:             i18n.ZH,
		Backend:          BackendAPI,
		APIFormat:        APIFormatOpenAIResponses,
		ModelAPIBaseURL:  "https://api.test/v1",
		ModelAPIKey:      "response-key",
		APIHTTPTransport: transport,
		Questions:        []questions.Question{q},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Cases[0].OK || summary.Cases[0].InputTokens != 11 || summary.Cases[0].ReasoningTokens != 3 {
		t.Fatalf("case = %#v", summary.Cases[0])
	}
}

func TestRunWithAnthropicMessagesAPIBackend(t *testing.T) {
	q := apiTestQuestion()
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "anthropic-key" {
			t.Fatalf("x-api-key = %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Fatal("missing anthropic-version")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "claude-sonnet-4-5" || body["max_tokens"] == nil {
			t.Fatalf("body = %#v", body)
		}
		return jsonResponse(http.StatusOK, map[string]any{
			"id": "msg_1",
			"content": []map[string]any{{
				"type": "text",
				"text": "21",
			}},
			"usage": map[string]any{
				"input_tokens":      12,
				"output_tokens":     7,
				"cache_read_tokens": 0,
			},
		})
	})

	summary, err := Run(context.Background(), Options{
		Model:            "claude-sonnet-4-5",
		ReasoningEffort:  "medium",
		Tests:            1,
		Timeout:          5 * time.Second,
		Lang:             i18n.ZH,
		Backend:          BackendAPI,
		APIFormat:        APIFormatAnthropic,
		ModelAPIBaseURL:  "https://api.test/v1/messages",
		ModelAPIKey:      "anthropic-key",
		APIHTTPTransport: transport,
		Questions:        []questions.Question{q},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Cases[0].OK || summary.Cases[0].InputTokens != 12 || summary.Cases[0].OutputTokens != 7 {
		t.Fatalf("case = %#v", summary.Cases[0])
	}
}

func TestAPIBackendRequiresKey(t *testing.T) {
	_, err := Run(context.Background(), Options{
		Model:           "gpt-5.4",
		ReasoningEffort: "medium",
		Tests:           1,
		Backend:         BackendAPI,
		APIFormat:       APIFormatOpenAIChat,
		ModelAPIBaseURL: "https://api.example.com/v1",
		Questions:       []questions.Question{apiTestQuestion()},
	})
	if err == nil || !strings.Contains(err.Error(), "Key") {
		t.Fatalf("err = %v", err)
	}
}

func TestAPIBackendRedactsKeyFromErrorBody(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return textResponse(http.StatusUnauthorized, "bad key secret-key"), nil
	})

	_, err := Run(context.Background(), Options{
		Model:            "gpt-5.4",
		ReasoningEffort:  "medium",
		Tests:            1,
		Backend:          BackendAPI,
		APIFormat:        APIFormatOpenAIChat,
		ModelAPIBaseURL:  "https://api.test/v1",
		ModelAPIKey:      "secret-key",
		APIHTTPTransport: transport,
		Questions:        []questions.Question{apiTestQuestion()},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "secret-key") || !strings.Contains(err.Error(), "[redacted]") {
		t.Fatalf("err = %v", err)
	}
}

func TestAPIBackendRetriesBrokenStream(t *testing.T) {
	q := apiTestQuestion()
	attempts := 0
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return nil, io.ErrUnexpectedEOF
		}
		return jsonResponse(http.StatusOK, map[string]any{
			"id": "chatcmpl_retry",
			"choices": []map[string]any{{
				"message": map[string]any{"content": "21"},
			}},
		})
	})

	summary, err := Run(context.Background(), Options{
		Model:            "gpt-5.4",
		ReasoningEffort:  "medium",
		Tests:            1,
		Timeout:          5 * time.Second,
		Backend:          BackendAPI,
		APIFormat:        APIFormatOpenAIChat,
		ModelAPIBaseURL:  "https://api.test/v1",
		ModelAPIKey:      "secret-key",
		APIHTTPTransport: transport,
		Questions:        []questions.Question{q},
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || !summary.Cases[0].OK {
		t.Fatalf("attempts=%d case=%#v", attempts, summary.Cases[0])
	}
}

func TestAPIBackendAuthFailureHint(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return textResponse(http.StatusUnauthorized, "invalid token"), nil
	})

	_, err := Run(context.Background(), Options{
		Model:            "gpt-5.4",
		ReasoningEffort:  "medium",
		Tests:            1,
		Backend:          BackendAPI,
		APIFormat:        APIFormatOpenAIChat,
		ModelAPIBaseURL:  "https://api.test/v1",
		ModelAPIKey:      "secret-key",
		APIHTTPTransport: transport,
		Questions:        []questions.Question{apiTestQuestion()},
	})
	if err == nil {
		t.Fatal("expected auth error")
	}
	if !strings.Contains(err.Error(), "认证失败") || !strings.Contains(err.Error(), "Key") {
		t.Fatalf("err = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func jsonResponse(status int, body any) (*http.Response, error) {
	var b strings.Builder
	if err := json.NewEncoder(&b).Encode(body); err != nil {
		return nil, err
	}
	return textResponse(status, b.String()), nil
}

func textResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func apiTestQuestion() questions.Question {
	return questions.Question{
		ID:      "api_21",
		Version: "1",
		Title:   "API test",
		Prompt:  "What is 20 + 1?",
		Grader: questions.Grader{
			Type:             "number",
			Expected:         "21",
			IndependentMatch: true,
		},
	}
}
