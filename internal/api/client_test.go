package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/1222hxy/LD-gpt-check/internal/questions"
	"github.com/1222hxy/LD-gpt-check/internal/runner"
)

func TestDoSendsAuthAndDecodesError(t *testing.T) {
	client := New("https://example.com", "token")
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("User-Agent") == "" {
			t.Fatal("missing User-Agent")
		}
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"error":"nope"}`)),
		}, nil
	})}

	err := client.do(context.Background(), requestOptions{method: http.MethodGet, path: "/api/me", auth: true})
	if err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("error = %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Status != http.StatusUnauthorized {
		t.Fatalf("status = %d", apiErr.Status)
	}
}

func TestClientRejectsInvalidBaseURL(t *testing.T) {
	err := New("://bad", "").do(context.Background(), requestOptions{method: http.MethodGet, path: "/health"})
	if err == nil {
		t.Fatal("expected invalid URL error")
	}
}

func TestPayloadFromSummaryStripsFullAnswer(t *testing.T) {
	s := runner.Summary{
		Model:           "gpt-5.5",
		ReasoningEffort: "xhigh",
		Tests:           1,
		Questions: []runner.QuestionSummary{{
			QuestionID:      questions.DefaultSuite,
			QuestionVersion: "1",
			QuestionTitle:   "糖果形状口味保证题",
			GraderType:      "number",
			ExpectedAnswer:  "21",
			PromptHash:      "abc",
			Tests:           1,
		}},
		Cases: []runner.CaseResult{{
			Index:             1,
			QuestionID:        questions.DefaultSuite,
			QuestionVersion:   "1",
			AnswerPreview:     "short",
			AnswerHash:        runner.SHA256Hex("full private answer"),
			FullAnswer:        "full private answer",
			InputTokens:       101,
			OutputTokens:      202,
			ReasoningTokens:   303,
			TimeSeconds:       4.5,
			TPS:               44.8,
			CachedInputTokens: 10,
			TotalTokens:       303,
			CodexThreadID:     "thread_1",
			EventCount:        3,
			EventTypes:        []string{"item.completed", "turn.completed"},
			AnswerChars:       12,
			StartedAt:         "2026-06-28T10:00:00Z",
			FinishedAt:        "2026-06-28T10:00:05Z",
			TimeoutSeconds:    1800,
		}},
		StartedAt:            "2026-06-28T10:00:00Z",
		FinishedAt:           "2026-06-28T10:00:05Z",
		DurationSeconds:      5,
		QuestionSuite:        questions.DefaultSuite,
		ClientTimezone:       "+08:00",
		CodexProviderBaseURL: "https://api.openai.com/v1",
	}
	p := PayloadFromSummary("0.1.0", s, "linux", "amd64", "codex 1")
	p.Anonymous = true
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) == "" || strings.Contains(string(b), "full private answer") {
		t.Fatalf("payload leaked full answer: %s", string(b))
	}
	if len(p.Attempts) != 1 {
		t.Fatalf("attempts = %d", len(p.Attempts))
	}
	a := p.Attempts[0]
	if a.InputTokens != 101 || a.OutputTokens != 202 || a.ReasoningTokens != 303 || a.TimeSeconds != 4.5 || a.TPS != 44.8 {
		t.Fatalf("table metrics not preserved: %#v", a)
	}
	if a.CachedInputTokens != 10 || a.TotalTokens != 303 || a.CodexThreadID != "thread_1" || a.EventCount != 3 || a.AnswerChars != 12 {
		t.Fatalf("diagnostics not preserved: %#v", a)
	}
	if p.UploadSchemaVersion != 4 || p.CodexProviderBaseURL != "https://api.openai.com/v1" || p.QuestionSuite != questions.DefaultSuite || p.ClientTimezone != "+08:00" || p.DurationSeconds != 5 {
		t.Fatalf("v4 summary fields not preserved: %#v", p)
	}
	if !p.Anonymous || !strings.Contains(string(b), `"anonymous":true`) {
		t.Fatalf("anonymous flag not preserved: %s", string(b))
	}
	if a.AnswerHash == "" || a.AnswerHash == a.AnswerPreview || a.StartedAt == "" || a.TimeoutSeconds != 1800 {
		t.Fatalf("v4 attempt fields not preserved: %#v", a)
	}
}

func TestRetrySafeRequestRetries503(t *testing.T) {
	var calls atomic.Int32
	client := New("https://example.com", "token")
	client.Retry = RetryPolicy{MaxAttempts: 2, BaseDelay: time.Nanosecond, MaxDelay: time.Nanosecond}
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			return response(http.StatusServiceUnavailable, `{"error":"try again"}`), nil
		}
		return response(http.StatusOK, `{"user":{"id":"u1","username":"alice"}}`), nil
	})}

	me, err := client.Me(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if me.User.ID != "u1" {
		t.Fatalf("user = %#v", me.User)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestRetrySafeRequestRetriesTransportError(t *testing.T) {
	var calls atomic.Int32
	client := New("https://example.com", "token")
	client.Retry = RetryPolicy{MaxAttempts: 2, BaseDelay: time.Nanosecond, MaxDelay: time.Nanosecond}
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			return nil, errors.New("temporary network error")
		}
		return response(http.StatusOK, `{"user":{"id":"u1"}}`), nil
	})}

	if _, err := client.Me(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestRequestURLPreservesBasePath(t *testing.T) {
	client := New("https://example.com/base", "")
	got, err := client.requestURL("/api/me")
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != "/base/api/me" {
		t.Fatalf("path = %q, url = %s", u.Path, got)
	}
}

func TestUploadRetriesTransientErrorWithSameUploadID(t *testing.T) {
	var calls atomic.Int32
	var uploadIDs []string
	client := New("https://example.com", "token")
	client.Retry = RetryPolicy{MaxAttempts: 3, BaseDelay: time.Nanosecond, MaxDelay: time.Nanosecond}
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body UploadPayload
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		uploadIDs = append(uploadIDs, body.UploadID)
		if calls.Add(1) == 1 {
			return response(http.StatusServiceUnavailable, `{"error":"busy"}`), nil
		}
		return response(http.StatusOK, `{"id":"sub_1","duplicate":false}`), nil
	})}

	payload := validUploadPayload()
	resp, err := client.UploadRun(context.Background(), payload)
	if err != nil {
		t.Fatal(err)
	}
	if resp["id"] != "sub_1" {
		t.Fatalf("response = %#v", resp)
	}
	if calls.Load() != 2 {
		t.Fatalf("upload should retry once, calls = %d", calls.Load())
	}
	if len(uploadIDs) != 2 || uploadIDs[0] != payload.UploadID || uploadIDs[1] != payload.UploadID {
		t.Fatalf("upload ids = %#v", uploadIDs)
	}
}

func TestDevicePollRejectsEmptyCode(t *testing.T) {
	if _, err := New("https://example.com", "").DevicePoll(context.Background(), " "); err == nil {
		t.Fatal("expected empty device code error")
	}
}

func TestUploadPayloadValidation(t *testing.T) {
	client := New("https://example.com", "token")
	if _, err := client.UploadRun(context.Background(), UploadPayload{UploadID: "upl_x", AttemptCount: 1}); err == nil {
		t.Fatal("expected missing model error")
	}
	if _, err := client.UploadRun(context.Background(), UploadPayload{UploadID: "upl_x", Model: "m"}); err == nil {
		t.Fatal("expected invalid tests error")
	}
	payload := validUploadPayload()
	payload.Attempts = append(payload.Attempts, UploadAttempt{QuestionID: questions.DefaultSuite, QuestionVersion: "1", CaseIndex: 2})
	if _, err := client.UploadRun(context.Background(), payload); err == nil {
		t.Fatal("expected attempts mismatch error")
	}
	payload = validUploadPayload()
	payload.Questions = nil
	if _, err := client.UploadRun(context.Background(), payload); err == nil {
		t.Fatal("expected questions mismatch error")
	}
	payload = validUploadPayload()
	payload.CodexProviderBaseURL = ""
	if _, err := client.UploadRun(context.Background(), payload); err == nil {
		t.Fatal("expected missing provider base url error")
	}
}

func TestParseRetryAfter(t *testing.T) {
	if got := parseRetryAfter("2"); got != 2*time.Second {
		t.Fatalf("retry-after seconds = %s", got)
	}
	if got := parseRetryAfter(""); got != 0 {
		t.Fatalf("empty retry-after = %s", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func validUploadPayload() UploadPayload {
	return UploadPayload{
		UploadID:             "upl_test",
		ClientVersion:        "0.1.0",
		Model:                "m",
		ReasoningEffort:      "medium",
		CodexProviderBaseURL: "https://api.openai.com/v1",
		QuestionCount:        1,
		AttemptCount:         1,
		Correct:              1,
		Questions: []UploadQuestionResult{{
			QuestionID:      questions.DefaultSuite,
			QuestionVersion: "1",
			QuestionTitle:   "糖果形状口味保证题",
			GraderType:      "number",
			ExpectedAnswer:  "21",
			PromptHash:      "abc",
			Tests:           1,
		}},
		Attempts: []UploadAttempt{{
			QuestionID:      questions.DefaultSuite,
			QuestionVersion: "1",
			CaseIndex:       1,
			Status:          "completed",
			IsCorrect:       true,
		}},
	}
}
