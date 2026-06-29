package runner

import (
	"context"
	"encoding/base64"
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

func TestParseEventsStreamsDeltas(t *testing.T) {
	input := strings.Join([]string{
		`{"type":"item.delta","delta":"答案"}`,
		`{"type":"item.delta","delta":"是 21"}`,
		`{"type":"item.completed","item":{"role":"assistant","content":"答案是 21"}}`,
	}, "\n")
	var chunks []string
	parsed, err := parseEventsWithStream(strings.NewReader(input), i18n.ZH, func(text string) {
		chunks = append(chunks, text)
	})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.FinalAnswer != "答案是 21" {
		t.Fatalf("answer = %q", parsed.FinalAnswer)
	}
	if !reflect.DeepEqual(chunks, []string{"答案", "是 21"}) {
		t.Fatalf("chunks = %#v", chunks)
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

func TestCodexStartupArgsAreParsedAndRecorded(t *testing.T) {
	parsed, err := ParseCodexStartupArgs(`-c model_provider="custom provider" --profile test`)
	if err != nil {
		t.Fatal(err)
	}
	wantParsed := []string{"-c", "model_provider=custom provider", "--profile", "test"}
	if !reflect.DeepEqual(parsed, wantParsed) {
		t.Fatalf("parsed = %#v", parsed)
	}

	args, err := codexArgsWithCustom("gpt-5.5", "high", `-c model_provider="custom provider" --profile test`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range wantParsed {
		found := false
		for _, arg := range args {
			if arg == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing custom arg %q in %#v", want, args)
		}
	}

	invocation := sanitizedInvocation("gpt-5.5", "high", `--profile test`)
	if !strings.Contains(invocation, `"custom_startup_args":"--profile test"`) || !strings.Contains(invocation, `"--profile"`) {
		t.Fatalf("invocation did not record custom args: %s", invocation)
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
	if _, err := Run(context.Background(), Options{ReasoningEffort: "\t"}); err == nil {
		t.Fatal("expected empty effort error")
	}
	if _, err := Run(context.Background(), Options{Model: "x", ReasoningEffort: "medium", Tests: MaxTests + 1}); err == nil {
		t.Fatal("expected max tests error")
	}
	if _, err := Run(context.Background(), Options{Model: "x", ReasoningEffort: "medium", Tests: -1}); err == nil {
		t.Fatal("expected negative tests error")
	}
}

func TestValidReasoningEffortAllowsCustomValues(t *testing.T) {
	for _, effort := range []string{"low", "xhigh", "minimal", "ultra-high", "vendor_custom_1"} {
		if !ValidReasoningEffort(effort) {
			t.Fatalf("ValidReasoningEffort(%q) = false", effort)
		}
	}
	for _, effort := range []string{"", " \t ", "bad\nvalue"} {
		if ValidReasoningEffort(effort) {
			t.Fatalf("ValidReasoningEffort(%q) = true", effort)
		}
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

func TestRunWithOpenAIChatAPIStreamProgress(t *testing.T) {
	q := apiTestQuestion()
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["stream"] != true || body["stream_options"] == nil {
			t.Fatalf("stream body = %#v", body)
		}
		return sseResponse(strings.Join([]string{
			`data: {"id":"chatcmpl_stream","choices":[{"delta":{"content":"答案"}}]}`,
			``,
			`data: {"choices":[{"delta":{"content":"是 21"}}]}`,
			``,
			`data: {"usage":{"prompt_tokens":10,"completion_tokens":5,"completion_tokens_details":{"reasoning_tokens":2}}}`,
			``,
			`data: [DONE]`,
			``,
		}, "\n")), nil
	})

	var chunks []string
	summary, err := Run(context.Background(), Options{
		Model:            "gpt-5.4",
		ReasoningEffort:  "medium",
		Tests:            1,
		Timeout:          5 * time.Second,
		Lang:             i18n.ZH,
		Backend:          BackendAPI,
		APIFormat:        APIFormatOpenAIChat,
		ModelAPIBaseURL:  "https://api.test/v1",
		ModelAPIKey:      "test-key",
		APIHTTPTransport: transport,
		Questions:        []questions.Question{q},
		Progress: func(ev ProgressEvent) {
			if ev.Type == ProgressCaseStream {
				chunks = append(chunks, ev.StreamText)
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Cases[0].OK || summary.Cases[0].InputTokens != 10 || summary.Cases[0].OutputTokens != 5 || summary.Cases[0].ReasoningTokens != 2 {
		t.Fatalf("case = %#v", summary.Cases[0])
	}
	if !reflect.DeepEqual(chunks, []string{"答案", "是 21"}) {
		t.Fatalf("chunks = %#v", chunks)
	}
}

func TestParseAPIStreamResponsesAndAnthropicDeltas(t *testing.T) {
	t.Run("responses", func(t *testing.T) {
		input := strings.Join([]string{
			`data: {"type":"response.output_text.delta","delta":"最终"}`,
			``,
			`data: {"type":"response.output_text.delta","delta":"答案：21"}`,
			``,
			`data: {"type":"response.completed","response":{"id":"resp_1","output_text":"最终答案：21","usage":{"input_tokens":11,"output_tokens":6,"output_tokens_details":{"reasoning_tokens":3}}}}`,
			``,
		}, "\n")
		var chunks []string
		parsed, err := parseAPIStream(strings.NewReader(input), APIFormatOpenAIResponses, func(ev ProgressEvent) {
			if ev.Type == ProgressCaseStream {
				chunks = append(chunks, ev.StreamText)
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		if parsed.FinalAnswer != "最终答案：21" || parsed.InputTokens != 11 || parsed.ReasoningTokens != 3 {
			t.Fatalf("parsed = %#v", parsed)
		}
		if !reflect.DeepEqual(chunks, []string{"最终", "答案：21"}) {
			t.Fatalf("chunks = %#v", chunks)
		}
	})

	t.Run("anthropic", func(t *testing.T) {
		input := strings.Join([]string{
			`data: {"type":"message_start","message":{"id":"msg_1","usage":{"input_tokens":12}}}`,
			``,
			`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"21"}}`,
			``,
			`data: {"type":"message_delta","usage":{"output_tokens":7}}`,
			``,
		}, "\n")
		var chunks []string
		parsed, err := parseAPIStream(strings.NewReader(input), APIFormatAnthropic, func(ev ProgressEvent) {
			if ev.Type == ProgressCaseStream {
				chunks = append(chunks, ev.StreamText)
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		if parsed.FinalAnswer != "21" || parsed.OutputTokens != 7 {
			t.Fatalf("parsed = %#v", parsed)
		}
		if !reflect.DeepEqual(chunks, []string{"21"}) {
			t.Fatalf("chunks = %#v", chunks)
		}
	})
}

func TestParseAPIStreamResponseFailed(t *testing.T) {
	input := strings.Join([]string{
		`data: {"type":"response.failed","response":{"error":{"message":"backend rejected request"}}}`,
		``,
	}, "\n")
	_, err := parseAPIStream(strings.NewReader(input), APIFormatOpenAIResponses, nil)
	if err == nil || !strings.Contains(err.Error(), "backend rejected request") {
		t.Fatalf("err = %v", err)
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

func TestRunWithAuthJSONBackend(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "auth.json")
	accessToken := runnerTestJWT(map[string]any{
		"email": "person@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "acct_secret",
			"chatgpt_plan_type":  "plus",
		},
	})
	if err := os.WriteFile(authPath, []byte(`{"auth_mode":"chatgpt","tokens":{"access_token":"`+accessToken+`"}}`), 0600); err != nil {
		t.Fatal(err)
	}

	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://chatgpt.com/backend-api/codex/responses" {
			t.Fatalf("url = %s", r.URL.String())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+accessToken {
			t.Fatalf("authorization = %q", got)
		}
		if got := r.Header.Get("Originator"); got != "codex_cli_rs" {
			t.Fatalf("originator = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("accept = %q", got)
		}
		if got := r.Header.Get("Origin"); got != "https://chatgpt.com" {
			t.Fatalf("origin = %q", got)
		}
		if got := r.Header.Get("Referer"); got != "https://chatgpt.com/" {
			t.Fatalf("referer = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-5.5" || body["store"] != false || body["stream"] != true {
			t.Fatalf("body = %#v", body)
		}
		input, _ := body["input"].([]any)
		if len(input) != 1 {
			t.Fatalf("input = %#v", body["input"])
		}
		message, _ := input[0].(map[string]any)
		content, _ := message["content"].([]any)
		if message["type"] != "message" || message["role"] != "user" || len(content) != 1 {
			t.Fatalf("message input = %#v", message)
		}
		textPart, _ := content[0].(map[string]any)
		if textPart["type"] != "input_text" || textPart["text"] != apiTestQuestion().Prompt {
			t.Fatalf("text part = %#v", textPart)
		}
		reasoning, _ := body["reasoning"].(map[string]any)
		if reasoning["effort"] != "xhigh" {
			t.Fatalf("reasoning = %#v", reasoning)
		}
		if reasoning["summary"] != "auto" {
			t.Fatalf("reasoning summary = %#v", reasoning)
		}
		text, _ := body["text"].(map[string]any)
		if text["verbosity"] != "medium" {
			t.Fatalf("text config = %#v", text)
		}
		return sseResponse(strings.Join([]string{
			`data: {"type":"response.output_text.delta","delta":"最终"}`,
			``,
			`data: {"type":"response.output_text.delta","delta":"答案：21"}`,
			``,
			`data: {"type":"response.completed","response":{"id":"resp_auth_1","output_text":"最终答案：21","usage":{"input_tokens":15,"output_tokens":6,"output_tokens_details":{"reasoning_tokens":4}}}}`,
			``,
			`data: [DONE]`,
			``,
		}, "\n")), nil
	})

	summary, err := Run(context.Background(), Options{
		Model:            "gpt-5.5",
		ReasoningEffort:  "xhigh",
		Tests:            1,
		Timeout:          5 * time.Second,
		Lang:             i18n.ZH,
		Backend:          BackendAuthJSON,
		AuthPath:         authPath,
		APIHTTPTransport: transport,
		Questions:        []questions.Question{apiTestQuestion()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Cases[0].OK || summary.Cases[0].InputTokens != 15 || summary.Cases[0].ReasoningTokens != 4 {
		t.Fatalf("case = %#v", summary.Cases[0])
	}
	if summary.CodexModelProvider != "auth-json" || summary.CodexProviderBaseURL != "https://chatgpt.com/backend-api/codex" || summary.CodexSandbox != "auth-json" {
		t.Fatalf("metadata = %#v", summary)
	}
	if strings.Contains(summary.CodexInvocation, accessToken) || !strings.Contains(summary.CodexInvocation, `"token_status":"valid"`) || !strings.Contains(summary.CodexInvocation, "p***@example.com") {
		t.Fatalf("unsafe invocation = %s", summary.CodexInvocation)
	}
}

func TestBuildCodexRequestIncludesRequiredRootFields(t *testing.T) {
	req := BuildCodexRequest("gpt-5.5", "", "hello")
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatal(err)
	}
	if body["stream"] != true || body["store"] != false {
		t.Fatalf("required booleans missing or wrong: %s", data)
	}
	reasoning, _ := body["reasoning"].(map[string]any)
	if reasoning["effort"] != "medium" || reasoning["summary"] != "auto" {
		t.Fatalf("reasoning = %#v", reasoning)
	}
	text, _ := body["text"].(map[string]any)
	if text["verbosity"] != "medium" {
		t.Fatalf("text = %#v", text)
	}
}

func TestAuthJSONBackendRequiresUsableAccessToken(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := Run(context.Background(), Options{
		Model:           "gpt-5.5",
		ReasoningEffort: "medium",
		Tests:           1,
		Backend:         BackendAuthJSON,
		AuthPath:        authPath,
		Questions:       []questions.Question{apiTestQuestion()},
	})
	if err == nil || !strings.Contains(err.Error(), "access_token") {
		t.Fatalf("err = %v", err)
	}
}

func TestAuthJSONBackendRefreshesMissingAccessToken(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"refresh_token":"refresh-secret"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	refreshedToken := runnerTestJWT(map[string]any{
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	var refreshed bool
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.String() {
		case "https://auth.openai.com/oauth/token":
			refreshed = true
			if got := r.Header.Get("Content-Type"); !strings.Contains(got, "application/x-www-form-urlencoded") {
				t.Fatalf("refresh content-type = %q", got)
			}
			return jsonResponse(http.StatusOK, map[string]any{
				"access_token": refreshedToken,
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case "https://chatgpt.com/backend-api/codex/responses":
			if got := r.Header.Get("Authorization"); got != "Bearer "+refreshedToken {
				t.Fatalf("authorization = %q", got)
			}
			return sseResponse(strings.Join([]string{
				`data: {"type":"response.output_text.delta","delta":"21"}`,
				``,
				`data: [DONE]`,
				``,
			}, "\n")), nil
		default:
			t.Fatalf("unexpected URL = %s", r.URL.String())
			return nil, nil
		}
	})

	summary, err := Run(context.Background(), Options{
		Model:            "gpt-5.5",
		ReasoningEffort:  "medium",
		Tests:            1,
		Backend:          BackendAuthJSON,
		AuthPath:         authPath,
		APIHTTPTransport: transport,
		Questions:        []questions.Question{apiTestQuestion()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !refreshed || !summary.Cases[0].OK {
		t.Fatalf("refreshed=%v case=%#v", refreshed, summary.Cases[0])
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

func TestAPIBackendDefaultsBaseURLByFormat(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://api.openai.com/v1/chat/completions" {
			t.Fatalf("url = %s", r.URL.String())
		}
		return jsonResponse(http.StatusOK, map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{"content": "21"},
			}},
		})
	})
	summary, err := Run(context.Background(), Options{
		Model:            "gpt-5.4",
		ReasoningEffort:  "medium",
		Tests:            1,
		Backend:          BackendAPI,
		APIFormat:        APIFormatOpenAIChat,
		ModelAPIKey:      "secret-key",
		APIHTTPTransport: transport,
		Questions:        []questions.Question{apiTestQuestion()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Cases[0].OK {
		t.Fatalf("case = %#v", summary.Cases[0])
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
	oldDelay := apiRetryBaseDelay
	apiRetryBaseDelay = time.Nanosecond
	defer func() { apiRetryBaseDelay = oldDelay }()

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

func TestAPIBackendRetriesServerErrorThenExplainsExhaustion(t *testing.T) {
	oldDelay := apiRetryBaseDelay
	apiRetryBaseDelay = time.Nanosecond
	defer func() { apiRetryBaseDelay = oldDelay }()

	attempts := 0
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		return textResponse(http.StatusBadGateway, "temporary upstream error"), nil
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
		t.Fatal("expected server error")
	}
	if attempts != apiMaxAttempts {
		t.Fatalf("attempts = %d", attempts)
	}
	if !strings.Contains(err.Error(), "自动重试仍失败") {
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

func sseResponse(body string) *http.Response {
	resp := textResponse(http.StatusOK, body)
	resp.Header.Set("Content-Type", "text/event-stream")
	return resp
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

func runnerTestJWT(claims map[string]any) string {
	header, _ := json.Marshal(map[string]any{"alg": "none"})
	payload, _ := json.Marshal(claims)
	return base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}
