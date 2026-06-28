package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/haowang02/ld-gpt-check/internal/config"
	"github.com/haowang02/ld-gpt-check/internal/i18n"
	"github.com/haowang02/ld-gpt-check/internal/runner"
)

type Client struct {
	BaseURL string
	Token   string
	Lang    i18n.Lang
	HTTP    *http.Client
	Retry   RetryPolicy
}

const (
	userAgent       = "ld-gpt-check/0.1"
	maxResponseBody = 1 << 20
)

type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{MaxAttempts: 3, BaseDelay: 500 * time.Millisecond, MaxDelay: 5 * time.Second}
}

type APIError struct {
	Method     string
	Path       string
	Status     int
	Message    string
	RequestID  string
	Body       string
	RetryAfter time.Duration
}

type transportError struct {
	Method string
	Path   string
	Err    error
}

func (e *transportError) Error() string {
	if e == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *transportError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	msg := e.Message
	if msg == "" {
		msg = e.Body
	}
	if msg == "" {
		msg = http.StatusText(e.Status)
	}
	if e.RequestID != "" {
		return fmt.Sprintf("%s (HTTP %d, request_id=%s)", msg, e.Status, e.RequestID)
	}
	return fmt.Sprintf("%s (HTTP %d)", msg, e.Status)
}

func New(baseURL, token string) Client {
	return NewWithLang(baseURL, token, i18n.Detect(""))
}

func NewWithLang(baseURL, token string, lang i18n.Lang) Client {
	baseURL = strings.TrimSpace(baseURL)
	return Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		Lang:    i18n.Normalize(string(lang)),
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		Retry:   DefaultRetryPolicy(),
	}
}

type DeviceStartResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type DevicePollResponse struct {
	Status      string      `json:"status"`
	AccessToken string      `json:"access_token"`
	User        config.User `json:"user"`
}

type MeResponse struct {
	User config.User `json:"user"`
}

type UploadPayload struct {
	UploadID              string                 `json:"upload_id"`
	UploadSchemaVersion   int                    `json:"upload_schema_version"`
	ClientVersion         string                 `json:"client_version"`
	Model                 string                 `json:"model"`
	ReasoningEffort       string                 `json:"reasoning_effort"`
	QuestionCount         int                    `json:"question_count"`
	AttemptCount          int                    `json:"attempt_count"`
	Correct               int                    `json:"correct"`
	Accuracy              float64                `json:"accuracy"`
	AvgInputTokens        float64                `json:"avg_input_tokens"`
	AvgOutputTokens       float64                `json:"avg_output_tokens"`
	AvgReasonTokens       float64                `json:"avg_reason_tokens"`
	AvgTimeSeconds        float64                `json:"avg_time_seconds"`
	AvgTPS                float64                `json:"avg_tps"`
	Anonymous             bool                   `json:"anonymous,omitempty"`
	StartedAt             string                 `json:"started_at,omitempty"`
	FinishedAt            string                 `json:"finished_at,omitempty"`
	DurationSeconds       float64                `json:"duration_seconds,omitempty"`
	QuestionSuite         string                 `json:"question_suite,omitempty"`
	ClientTimezone        string                 `json:"client_timezone,omitempty"`
	OS                    string                 `json:"os"`
	Arch                  string                 `json:"arch"`
	CodexVersion          string                 `json:"codex_version"`
	CodexModelSource      string                 `json:"codex_model_source"`
	CodexModelProvider    string                 `json:"codex_model_provider,omitempty"`
	CodexProviderHost     string                 `json:"codex_provider_host,omitempty"`
	CodexSandbox          string                 `json:"codex_sandbox"`
	CodexEphemeral        bool                   `json:"codex_ephemeral"`
	CodexSkipGitRepoCheck bool                   `json:"codex_skip_git_repo_check"`
	CodexDisabledFeatures []string               `json:"codex_disabled_features,omitempty"`
	CodexInvocation       string                 `json:"codex_invocation,omitempty"`
	Questions             []UploadQuestionResult `json:"questions"`
	Attempts              []UploadAttempt        `json:"attempts"`
}

type UploadQuestionResult struct {
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

type UploadAttempt struct {
	QuestionID             string   `json:"question_id"`
	QuestionVersion        string   `json:"question_version"`
	CaseIndex              int      `json:"case_index"`
	Status                 string   `json:"status"`
	IsCorrect              bool     `json:"is_correct"`
	ExpectedAnswer         string   `json:"expected_answer"`
	ExtractedAnswer        string   `json:"extracted_answer"`
	FailureReason          string   `json:"failure_reason,omitempty"`
	AnswerPreview          string   `json:"answer_preview"`
	AnswerPreviewTruncated bool     `json:"answer_preview_truncated"`
	AnswerHash             string   `json:"answer_hash,omitempty"`
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
	ErrorCode              string   `json:"error_code,omitempty"`
	StartedAt              string   `json:"started_at,omitempty"`
	FinishedAt             string   `json:"finished_at,omitempty"`
	TimeoutSeconds         float64  `json:"timeout_seconds,omitempty"`
}

func PayloadFromSummary(version string, s runner.Summary, osName, arch, codexVersion string) UploadPayload {
	attempts := make([]UploadAttempt, 0, len(s.Cases))
	for _, c := range s.Cases {
		attempts = append(attempts, UploadAttempt{
			QuestionID:             c.QuestionID,
			QuestionVersion:        c.QuestionVersion,
			CaseIndex:              c.Index,
			Status:                 firstNonEmpty(c.Status, "completed"),
			IsCorrect:              c.OK,
			ExpectedAnswer:         c.ExpectedAnswer,
			ExtractedAnswer:        c.ExtractedAnswer,
			FailureReason:          c.FailureReason,
			AnswerPreview:          runner.Preview(c.AnswerPreview, 300),
			AnswerPreviewTruncated: c.AnswerPreviewTruncated,
			AnswerHash:             c.AnswerHash,
			InputTokens:            c.InputTokens,
			CachedInputTokens:      c.CachedInputTokens,
			OutputTokens:           c.OutputTokens,
			ReasoningTokens:        c.ReasoningTokens,
			TotalTokens:            c.TotalTokens,
			TimeSeconds:            c.TimeSeconds,
			TPS:                    c.TPS,
			CodexThreadID:          c.CodexThreadID,
			EventCount:             c.EventCount,
			EventTypes:             append([]string(nil), c.EventTypes...),
			ToolEventDetected:      c.ToolEventDetected,
			AnswerChars:            c.AnswerChars,
			ErrorCode:              c.ErrorCode,
			StartedAt:              c.StartedAt,
			FinishedAt:             c.FinishedAt,
			TimeoutSeconds:         c.TimeoutSeconds,
		})
	}
	questionResults := make([]UploadQuestionResult, 0, len(s.Questions))
	for _, q := range s.Questions {
		questionResults = append(questionResults, UploadQuestionResult(q))
	}
	return UploadPayload{
		UploadID:              newUploadID(),
		UploadSchemaVersion:   firstPositive(s.UploadSchemaVersion, 3),
		ClientVersion:         version,
		Model:                 strings.TrimSpace(s.Model),
		ReasoningEffort:       s.ReasoningEffort,
		QuestionCount:         len(s.Questions),
		AttemptCount:          len(s.Cases),
		Correct:               s.Correct,
		Accuracy:              s.Accuracy,
		AvgInputTokens:        s.AvgInputTokens,
		AvgOutputTokens:       s.AvgOutputTokens,
		AvgReasonTokens:       s.AvgReasoningTokens,
		AvgTimeSeconds:        s.AvgTimeSeconds,
		AvgTPS:                s.AvgTPS,
		StartedAt:             s.StartedAt,
		FinishedAt:            s.FinishedAt,
		DurationSeconds:       s.DurationSeconds,
		QuestionSuite:         s.QuestionSuite,
		ClientTimezone:        s.ClientTimezone,
		OS:                    osName,
		Arch:                  arch,
		CodexVersion:          codexVersion,
		CodexModelSource:      firstNonEmpty(s.CodexModelSource, "unknown"),
		CodexModelProvider:    s.CodexModelProvider,
		CodexProviderHost:     s.CodexProviderHost,
		CodexSandbox:          firstNonEmpty(s.CodexSandbox, "read-only"),
		CodexEphemeral:        s.CodexEphemeral,
		CodexSkipGitRepoCheck: s.CodexSkipGitRepoCheck,
		CodexDisabledFeatures: append([]string(nil), s.CodexDisabledFeatures...),
		CodexInvocation:       s.CodexInvocation,
		Questions:             questionResults,
		Attempts:              attempts,
	}
}

func (c Client) DeviceStart(ctx context.Context) (DeviceStartResponse, error) {
	var out DeviceStartResponse
	err := c.do(ctx, requestOptions{method: http.MethodPost, path: "/api/device/start", out: &out})
	return out, err
}

func (c Client) DevicePoll(ctx context.Context, deviceCode string) (DevicePollResponse, error) {
	if strings.TrimSpace(deviceCode) == "" {
		return DevicePollResponse{}, fmt.Errorf("%s", c.localizer().S("api_empty_device_code"))
	}
	var out DevicePollResponse
	err := c.do(ctx, requestOptions{
		method:    http.MethodPost,
		path:      "/api/device/poll",
		body:      map[string]string{"device_code": deviceCode},
		out:       &out,
		retrySafe: true,
	})
	return out, err
}

func (c Client) Me(ctx context.Context) (MeResponse, error) {
	var out MeResponse
	err := c.do(ctx, requestOptions{method: http.MethodGet, path: "/api/me", auth: true, out: &out, retrySafe: true})
	return out, err
}

func (c Client) UploadRun(ctx context.Context, payload UploadPayload) (map[string]any, error) {
	if err := c.validateUploadPayload(payload); err != nil {
		return nil, err
	}
	var out map[string]any
	err := c.do(ctx, requestOptions{method: http.MethodPost, path: "/api/v1/submissions", body: payload, auth: true, out: &out, retrySafe: true})
	return out, err
}

func (c Client) DeleteSubmission(ctx context.Context, id string) (map[string]any, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("submission id is required")
	}
	var out map[string]any
	err := c.do(ctx, requestOptions{method: http.MethodDelete, path: "/api/v1/submissions/" + url.PathEscape(id), auth: true, out: &out, retrySafe: true})
	return out, err
}

func (c Client) DeleteAllSubmissions(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	err := c.do(ctx, requestOptions{method: http.MethodDelete, path: "/api/v1/submissions", auth: true, out: &out})
	return out, err
}

func (c Client) Logout(ctx context.Context) error {
	var out map[string]any
	return c.do(ctx, requestOptions{method: http.MethodPost, path: "/api/logout", auth: true, out: &out, retrySafe: true})
}

type requestOptions struct {
	method    string
	path      string
	body      any
	auth      bool
	out       any
	retrySafe bool
}

func (c Client) do(ctx context.Context, opts requestOptions) error {
	if err := c.validate(); err != nil {
		return err
	}
	var bodyBytes []byte
	if opts.body != nil {
		b, err := json.Marshal(opts.body)
		if err != nil {
			return err
		}
		bodyBytes = b
	}

	attempts := 1
	if opts.retrySafe {
		attempts = c.retryPolicy().MaxAttempts
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		resp, respBody, err := c.once(ctx, opts, bodyBytes)
		if err == nil {
			return c.handleResponse(opts, resp, respBody)
		}
		lastErr = err
		if attempt == attempts || !c.shouldRetry(err) {
			return err
		}
		if err := sleepContext(ctx, c.retryDelay(attempt, err)); err != nil {
			return err
		}
	}
	return lastErr
}

func (c Client) once(ctx context.Context, opts requestOptions, bodyBytes []byte) (*http.Response, []byte, error) {
	var r io.Reader
	if bodyBytes != nil {
		r = bytes.NewReader(bodyBytes)
	}
	reqURL, err := c.requestURL(opts.path)
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, opts.method, reqURL, r)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if opts.auth {
		if c.Token == "" {
			return nil, nil, fmt.Errorf("%s", c.localizer().S("api_not_logged_in"))
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, nil, &transportError{
			Method: opts.method,
			Path:   opts.path,
			Err:    fmt.Errorf("%s", c.localizer().S("api_request_failed", opts.method, opts.path, err)),
		}
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp, respBody, c.responseError(opts, resp, respBody)
	}
	return resp, respBody, nil
}

func (c Client) handleResponse(opts requestOptions, resp *http.Response, respBody []byte) error {
	if opts.out == nil || len(strings.TrimSpace(string(respBody))) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, opts.out); err != nil {
		return fmt.Errorf("%s", c.localizer().S("api_decode_failed", opts.path, err))
	}
	return nil
}

func (c Client) responseError(opts requestOptions, resp *http.Response, respBody []byte) error {
	bodyText := strings.TrimSpace(string(respBody))
	var e struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	msg := ""
	if json.Unmarshal(respBody, &e) == nil {
		msg = firstNonEmpty(e.Error, e.Message)
	}
	if msg == "" {
		msg = bodyText
	}
	if msg == "" {
		msg = c.localizer().S("api_status_failed", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return &APIError{
		Method:     opts.method,
		Path:       opts.path,
		Status:     resp.StatusCode,
		Message:    msg,
		RequestID:  responseRequestID(resp),
		Body:       bodyText,
		RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
	}
}

func (c Client) validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("%s", c.localizer().S("api_base_empty"))
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%s", c.localizer().S("api_base_invalid", c.BaseURL))
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s", c.localizer().S("api_base_bad_scheme", u.Scheme))
	}
	if c.HTTP == nil {
		return fmt.Errorf("%s", c.localizer().S("api_http_nil"))
	}
	return nil
}

func (c Client) requestURL(path string) (string, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", err
	}
	rel, err := url.Parse(strings.TrimPrefix(path, "/"))
	if err != nil {
		return "", err
	}
	if base.Path == "" || strings.HasSuffix(base.Path, "/") {
		base.Path += rel.Path
	} else {
		base.Path += "/" + rel.Path
	}
	base.RawQuery = rel.RawQuery
	return base.String(), nil
}

func (c Client) localizer() i18n.Localizer {
	return i18n.New(c.Lang)
}

func (c Client) retryPolicy() RetryPolicy {
	p := c.Retry
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = 1
	}
	if p.BaseDelay <= 0 {
		p.BaseDelay = 500 * time.Millisecond
	}
	if p.MaxDelay <= 0 {
		p.MaxDelay = 5 * time.Second
	}
	return p
}

func (c Client) shouldRetry(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Status == http.StatusTooManyRequests || apiErr.Status == http.StatusBadGateway ||
			apiErr.Status == http.StatusServiceUnavailable || apiErr.Status == http.StatusGatewayTimeout
	}
	var netErr *transportError
	return errors.As(err, &netErr)
}

func (c Client) retryDelay(attempt int, err error) time.Duration {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.RetryAfter > 0 && apiErr.RetryAfter <= c.retryPolicy().MaxDelay {
			return apiErr.RetryAfter
		}
		if apiErr.Status == http.StatusTooManyRequests {
			return c.retryPolicy().BaseDelay * time.Duration(attempt)
		}
	}
	d := float64(c.retryPolicy().BaseDelay) * math.Pow(2, float64(attempt-1))
	delay := time.Duration(d)
	if delay > c.retryPolicy().MaxDelay {
		delay = c.retryPolicy().MaxDelay
	}
	return delay
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func responseRequestID(resp *http.Response) string {
	for _, key := range []string{"cf-ray", "x-request-id", "x-correlation-id"} {
		if v := resp.Header.Get(key); v != "" {
			return v
		}
	}
	return ""
}

func (c Client) validateUploadPayload(p UploadPayload) error {
	l := c.localizer()
	if strings.TrimSpace(p.Model) == "" {
		return fmt.Errorf("%s", l.S("api_upload_model_required"))
	}
	if strings.TrimSpace(p.UploadID) == "" {
		return fmt.Errorf("%s", l.S("api_upload_id_required"))
	}
	if p.AttemptCount <= 0 {
		return fmt.Errorf("%s", l.S("api_upload_tests_invalid"))
	}
	if len(p.Attempts) != p.AttemptCount {
		return fmt.Errorf("%s", l.S("api_upload_cases_mismatch"))
	}
	if p.QuestionCount <= 0 || len(p.Questions) != p.QuestionCount {
		return fmt.Errorf("%s", l.S("api_upload_questions_mismatch"))
	}
	return nil
}

func newUploadID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "upl_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return "upl_" + hex.EncodeToString(b[:])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func firstPositive(values ...int) int {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}

func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(v); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}
