package runner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/1222hxy/LD-gpt-check/internal/i18n"
	"github.com/1222hxy/LD-gpt-check/internal/questions"
	"github.com/1222hxy/LD-gpt-check/internal/system"
	"github.com/go-resty/resty/v2"
)

const (
	apiSystemPrompt      = "You are a helpful assistant. Answer the user's question directly."
	anthropicAPIVersion  = "2023-06-01"
	defaultAPIMaxTokens  = 4096
	envModelAPIKey       = "LD_GPT_CHECK_MODEL_API_KEY"
	envModelAPIBaseURL   = "LD_GPT_CHECK_MODEL_API_BASE_URL"
	envModelAPIFormat    = "LD_GPT_CHECK_API_FORMAT"
	apiResponseBodyLimit = 4096
	apiMaxAttempts       = 3
)

var apiRetryBaseDelay = 500 * time.Millisecond
var errAPIStreamUnsupported = errors.New("api streaming unsupported")

type apiBackend struct {
	format          APIFormat
	model           string
	apiKey          string
	providerBaseURL string
	providerHost    string
	endpointURL     string
	client          *resty.Client
}

func newAPIBackend(opts Options) (*apiBackend, error) {
	l := i18n.New(opts.Lang)
	format, ok := NormalizeAPIFormat(APIFormat(firstNonEmpty(string(opts.APIFormat), os.Getenv(envModelAPIFormat))))
	if !ok {
		return nil, fmt.Errorf("%s", l.S("runner_api_format_invalid", opts.APIFormat))
	}
	model := strings.TrimSpace(opts.Model)
	if !system.ConcreteCodexModel(model) {
		return nil, fmt.Errorf("%s", l.S("runner_model_required"))
	}
	apiKey := strings.TrimSpace(firstNonEmpty(opts.ModelAPIKey, os.Getenv(envModelAPIKey)))
	if apiKey == "" {
		return nil, fmt.Errorf("%s", l.S("runner_api_key_required", envModelAPIKey))
	}
	rawBase := firstNonEmpty(opts.ModelAPIBaseURL, os.Getenv(envModelAPIBaseURL), defaultModelAPIBaseURL(format))
	endpointURL, providerBaseURL, providerHost, err := apiEndpointURL(rawBase, format)
	if err != nil {
		return nil, fmt.Errorf("%s", l.S("runner_api_base_invalid", rawBase))
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	client := resty.New().
		SetTimeout(timeout).
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetHeader("User-Agent", "ld-gpt-check/0.1")
	if opts.APIHTTPTransport != nil {
		client.SetTransport(opts.APIHTTPTransport)
	}
	return &apiBackend{
		format:          format,
		model:           model,
		apiKey:          apiKey,
		providerBaseURL: providerBaseURL,
		providerHost:    providerHost,
		endpointURL:     endpointURL,
		client:          client,
	}, nil
}

func defaultModelAPIBaseURL(format APIFormat) string {
	if format == APIFormatAnthropic {
		return "https://api.anthropic.com/v1"
	}
	return "https://api.openai.com/v1"
}

func (b *apiBackend) runOne(ctx context.Context, opts Options, q questions.Question, index int) (CaseResult, error) {
	runCtx := ctx
	cancel := func() {}
	if opts.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
	}
	defer cancel()

	start := time.Now()
	parsed, err := b.request(runCtx, opts, q.Prompt)
	if err != nil {
		if runCtx.Err() == context.Canceled {
			return CaseResult{}, context.Canceled
		}
		if runCtx.Err() == context.DeadlineExceeded {
			return CaseResult{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_api_timeout", opts.Timeout))
		}
		return CaseResult{}, err
	}
	if parsed.FinalAnswer == "" {
		return CaseResult{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_api_empty_answer"))
	}
	return caseResultFromParsed(opts, q, index, parsed, start, time.Now()), nil
}

func (b *apiBackend) request(ctx context.Context, opts Options, prompt string) (ParsedEvents, error) {
	if opts.Progress != nil {
		parsed, err := b.requestStream(ctx, opts, prompt)
		if err == nil {
			return parsed, nil
		}
		if !errors.Is(err, errAPIStreamUnsupported) {
			return ParsedEvents{}, err
		}
	}
	return b.requestNonStream(ctx, opts, prompt)
}

func (b *apiBackend) requestNonStream(ctx context.Context, opts Options, prompt string) (ParsedEvents, error) {
	var resp *resty.Response
	var err error
	for attempt := 1; attempt <= apiMaxAttempts; attempt++ {
		resp, err = b.doRequest(ctx, opts, prompt)
		if !shouldRetryAPI(resp, err) || attempt == apiMaxAttempts {
			break
		}
		if sleepErr := sleepWithContext(ctx, time.Duration(attempt)*apiRetryBaseDelay); sleepErr != nil {
			return ParsedEvents{}, sleepErr
		}
	}
	if err != nil {
		return ParsedEvents{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_api_failed", classifyTransportError(err, opts.Lang)))
	}
	if resp == nil {
		return ParsedEvents{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_api_failed", i18n.New(opts.Lang).S("runner_api_empty_response")))
	}
	if resp.IsError() {
		status := resp.StatusCode()
		preview := bodyPreview(resp.Body(), b.apiKey)
		if status == httpStatusUnauthorized || status == httpStatusForbidden {
			return ParsedEvents{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_api_auth_failed", status, preview))
		}
		if shouldRetryStatus(status) {
			return ParsedEvents{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_api_status_retry_exhausted", status, preview))
		}
		return ParsedEvents{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_api_status_failed", status, preview))
	}
	var obj map[string]any
	dec := json.NewDecoder(strings.NewReader(string(resp.Body())))
	dec.UseNumber()
	if err := dec.Decode(&obj); err != nil {
		return ParsedEvents{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_api_decode_failed", err))
	}
	return parseAPIResponse(b.format, obj), nil
}

func (b *apiBackend) requestStream(ctx context.Context, opts Options, prompt string) (ParsedEvents, error) {
	var resp *http.Response
	var err error
	for attempt := 1; attempt <= apiMaxAttempts; attempt++ {
		resp, err = b.doStreamRequest(ctx, opts, prompt)
		if retryableTransportError(err) && attempt < apiMaxAttempts {
			if sleepErr := sleepWithContext(ctx, time.Duration(attempt)*apiRetryBaseDelay); sleepErr != nil {
				return ParsedEvents{}, sleepErr
			}
			continue
		}
		if err == nil && resp != nil && shouldRetryStatus(resp.StatusCode) && attempt < apiMaxAttempts {
			_ = resp.Body.Close()
			if sleepErr := sleepWithContext(ctx, time.Duration(attempt)*apiRetryBaseDelay); sleepErr != nil {
				return ParsedEvents{}, sleepErr
			}
			continue
		}
		break
	}
	if err != nil {
		return ParsedEvents{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_api_failed", classifyTransportError(err, opts.Lang)))
	}
	if resp == nil {
		return ParsedEvents{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_api_failed", i18n.New(opts.Lang).S("runner_api_empty_response")))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		preview := streamBodyPreview(resp.Body, b.apiKey)
		if resp.StatusCode == httpStatusUnauthorized || resp.StatusCode == httpStatusForbidden {
			return ParsedEvents{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_api_auth_failed", resp.StatusCode, preview))
		}
		if streamFallbackStatus(resp.StatusCode) {
			return ParsedEvents{}, errAPIStreamUnsupported
		}
		if shouldRetryStatus(resp.StatusCode) {
			return ParsedEvents{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_api_status_retry_exhausted", resp.StatusCode, preview))
		}
		return ParsedEvents{}, fmt.Errorf("%s", i18n.New(opts.Lang).S("runner_api_status_failed", resp.StatusCode, preview))
	}
	return parseAPIStream(resp.Body, b.format, opts.Progress)
}

func (b *apiBackend) doRequest(ctx context.Context, opts Options, prompt string) (*resty.Response, error) {
	req := b.client.R().
		SetContext(ctx).
		SetBody(b.requestBody(prompt, opts.ReasoningEffort))
	if b.format == APIFormatAnthropic {
		req.SetHeader("x-api-key", b.apiKey)
		req.SetHeader("anthropic-version", anthropicAPIVersion)
	} else {
		req.SetHeader("Authorization", "Bearer "+b.apiKey)
	}
	return req.Post(b.endpointURL)
}

func (b *apiBackend) doStreamRequest(ctx context.Context, opts Options, prompt string) (*http.Response, error) {
	body := b.streamRequestBody(prompt, opts.ReasoningEffort)
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpointURL, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ld-gpt-check/0.1")
	if b.format == APIFormatAnthropic {
		req.Header.Set("x-api-key", b.apiKey)
		req.Header.Set("anthropic-version", anthropicAPIVersion)
	} else {
		req.Header.Set("Authorization", "Bearer "+b.apiKey)
	}
	return b.client.GetClient().Do(req)
}

func (b *apiBackend) streamRequestBody(prompt, effort string) map[string]any {
	body := b.requestBody(prompt, effort)
	body["stream"] = true
	if b.format == APIFormatOpenAIChat {
		body["stream_options"] = map[string]any{"include_usage": true}
	}
	return body
}

func streamFallbackStatus(status int) bool {
	return status == 400 || status == 404 || status == 405 || status == 415 || status == 422
}

func streamBodyPreview(r io.Reader, apiKey string) string {
	b, _ := io.ReadAll(io.LimitReader(r, apiResponseBodyLimit))
	return bodyPreview(b, apiKey)
}

const (
	httpStatusUnauthorized = 401
	httpStatusForbidden    = 403
	httpStatusTooMany      = 429
)

func shouldRetryAPI(resp *resty.Response, err error) bool {
	if err != nil {
		return retryableTransportError(err)
	}
	if resp == nil {
		return false
	}
	status := resp.StatusCode()
	return shouldRetryStatus(status)
}

func shouldRetryStatus(status int) bool {
	return status == httpStatusTooMany || status == 408 || status >= 500
}

func retryableTransportError(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF || strings.Contains(strings.ToLower(err.Error()), "unexpected eof") || strings.Contains(strings.ToLower(err.Error()), "connection reset") {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary())
}

func classifyTransportError(err error, lang i18n.Lang) string {
	if retryableTransportError(err) {
		return i18n.New(lang).S("runner_api_retry_exhausted", err)
	}
	return err.Error()
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (b *apiBackend) requestBody(prompt, effort string) map[string]any {
	switch b.format {
	case APIFormatOpenAIResponses:
		body := map[string]any{
			"model":        b.model,
			"instructions": apiSystemPrompt,
			"input":        prompt,
		}
		if effort = strings.TrimSpace(effort); effort != "" {
			body["reasoning"] = map[string]any{"effort": effort}
		}
		return body
	case APIFormatAnthropic:
		return map[string]any{
			"model":      b.model,
			"system":     apiSystemPrompt,
			"messages":   []map[string]string{{"role": "user", "content": prompt}},
			"max_tokens": defaultAPIMaxTokens,
		}
	default:
		body := map[string]any{
			"model": b.model,
			"messages": []map[string]string{
				{"role": "system", "content": apiSystemPrompt},
				{"role": "user", "content": prompt},
			},
		}
		if effort = strings.TrimSpace(effort); effort != "" {
			body["reasoning_effort"] = effort
		}
		return body
	}
}

func parseAPIResponse(format APIFormat, obj map[string]any) ParsedEvents {
	parsed := ParsedEvents{
		ThreadID:   stringField(obj, "id"),
		EventCount: 1,
		EventTypes: []string{"api." + string(format)},
	}
	switch format {
	case APIFormatOpenAIResponses:
		parsed.FinalAnswer = firstNonEmpty(
			textFromValue(obj["output_text"]),
			textFromValue(obj["output"]),
			textFromValue(obj["content"]),
			textFromValue(obj["text"]),
			textFromValue(nestedValue(obj, "message", "content")),
		)
	case APIFormatAnthropic:
		parsed.FinalAnswer = textFromValue(obj["content"])
	default:
		parsed.FinalAnswer = openAIChatText(obj)
	}
	parsed.InputTokens, parsed.CachedInputTokens, parsed.OutputTokens, parsed.ReasoningTokens = extractAPIUsage(obj)
	return parsed
}

func nestedValue(m map[string]any, objectKey, valueKey string) any {
	if obj, ok := m[objectKey].(map[string]any); ok {
		return obj[valueKey]
	}
	return nil
}

func parseAPIStream(r io.Reader, format APIFormat, progress func(ProgressEvent)) (ParsedEvents, error) {
	var parsed ParsedEvents
	eventTypes := map[string]struct{}{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	var dataLines []string
	flush := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		payload := strings.TrimSpace(strings.Join(dataLines, "\n"))
		dataLines = nil
		if payload == "" || payload == "[DONE]" {
			return nil
		}
		var obj map[string]any
		dec := json.NewDecoder(strings.NewReader(payload))
		dec.UseNumber()
		if err := dec.Decode(&obj); err != nil {
			return err
		}
		if err := apiStreamFailureError(obj); err != nil {
			return err
		}
		applyAPIStreamEvent(format, obj, &parsed, eventTypes, progress)
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			if err := flush(); err != nil {
				return ParsedEvents{}, err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return ParsedEvents{}, err
	}
	if err := flush(); err != nil {
		return ParsedEvents{}, err
	}
	parsed.EventTypes = sortedKeys(eventTypes)
	if parsed.EventCount == 0 && parsed.FinalAnswer == "" {
		return ParsedEvents{}, io.ErrUnexpectedEOF
	}
	return parsed, nil
}

func apiStreamFailureError(obj map[string]any) error {
	if stringField(obj, "type") != "response.failed" {
		return nil
	}
	response, _ := obj["response"].(map[string]any)
	if response == nil {
		return fmt.Errorf("response failed")
	}
	errObj, _ := response["error"].(map[string]any)
	if errObj == nil {
		return fmt.Errorf("response failed")
	}
	msg := firstNonEmpty(stringField(errObj, "message"), stringField(errObj, "code"), "response failed")
	return fmt.Errorf("%s", msg)
}

func applyAPIStreamEvent(format APIFormat, obj map[string]any, parsed *ParsedEvents, eventTypes map[string]struct{}, progress func(ProgressEvent)) {
	parsed.EventCount++
	name := firstNonEmpty(eventName(obj), stringField(obj, "type"))
	if name == "" {
		name = "api." + string(format) + ".stream"
	}
	eventTypes[name] = struct{}{}
	if parsed.ThreadID == "" {
		parsed.ThreadID = firstNonEmpty(stringField(obj, "id"), nestedStringField(obj, "response", "id"), nestedStringField(obj, "message", "id"))
	}
	if delta := apiStreamDelta(format, obj); delta != "" {
		parsed.FinalAnswer += delta
		if progress != nil {
			progress(ProgressEvent{Type: ProgressCaseStream, StreamText: delta})
		}
	}
	if completed := apiStreamCompleted(format, obj); completed.FinalAnswer != "" || completed.InputTokens > 0 || completed.OutputTokens > 0 || completed.ReasoningTokens > 0 {
		mergeParsedAPIStream(parsed, completed)
	}
	in, cached, out, reason := extractAPIUsage(obj)
	if in > 0 || cached > 0 || out > 0 || reason > 0 {
		parsed.InputTokens, parsed.CachedInputTokens, parsed.OutputTokens, parsed.ReasoningTokens = in, cached, out, reason
	}
}

func apiStreamDelta(format APIFormat, obj map[string]any) string {
	switch format {
	case APIFormatOpenAIResponses:
		if strings.Contains(stringField(obj, "type"), "output_text.delta") {
			return textFromValue(obj["delta"])
		}
		return textFromValue(obj["delta"])
	case APIFormatAnthropic:
		if delta, _ := obj["delta"].(map[string]any); delta != nil {
			return textFromValue(delta["text"])
		}
		return ""
	default:
		var b strings.Builder
		choices, _ := obj["choices"].([]any)
		for _, choice := range choices {
			m, _ := choice.(map[string]any)
			if delta, _ := m["delta"].(map[string]any); delta != nil {
				b.WriteString(textFromValue(delta["content"]))
			}
			b.WriteString(textFromValue(m["text"]))
		}
		return b.String()
	}
}

func apiStreamCompleted(format APIFormat, obj map[string]any) ParsedEvents {
	if response, _ := obj["response"].(map[string]any); response != nil {
		return parseAPIResponse(format, response)
	}
	if message, _ := obj["message"].(map[string]any); message != nil {
		return parseAPIResponse(format, message)
	}
	return ParsedEvents{}
}

func mergeParsedAPIStream(dst *ParsedEvents, src ParsedEvents) {
	if src.ThreadID != "" {
		dst.ThreadID = src.ThreadID
	}
	if src.FinalAnswer != "" {
		dst.FinalAnswer = src.FinalAnswer
	}
	if src.InputTokens > 0 || src.CachedInputTokens > 0 || src.OutputTokens > 0 || src.ReasoningTokens > 0 {
		dst.InputTokens = src.InputTokens
		dst.CachedInputTokens = src.CachedInputTokens
		dst.OutputTokens = src.OutputTokens
		dst.ReasoningTokens = src.ReasoningTokens
	}
}

func openAIChatText(obj map[string]any) string {
	choices, _ := obj["choices"].([]any)
	for _, choice := range choices {
		m, _ := choice.(map[string]any)
		if msg, _ := m["message"].(map[string]any); msg != nil {
			if text := textFromValue(msg["content"]); text != "" {
				return text
			}
		}
		if text := textFromValue(m["text"]); text != "" {
			return text
		}
	}
	return ""
}

func extractAPIUsage(obj map[string]any) (int, int, int, int) {
	usage, _ := obj["usage"].(map[string]any)
	if usage == nil {
		usage = obj
	}
	input := firstPositiveInt(
		intField(usage, "input_tokens"),
		intField(usage, "prompt_tokens"),
	)
	output := firstPositiveInt(
		intField(usage, "output_tokens"),
		intField(usage, "completion_tokens"),
	)
	cached := firstPositiveInt(
		intField(usage, "cached_input_tokens"),
		nestedIntField(usage, "input_tokens_details", "cached_tokens"),
		nestedIntField(usage, "prompt_tokens_details", "cached_tokens"),
	)
	if cached == 0 {
		cached = intField(usage, "cache_read_input_tokens") + intField(usage, "cache_creation_input_tokens")
	}
	reason := firstPositiveInt(
		intField(usage, "reasoning_output_tokens"),
		intField(usage, "reasoning_tokens"),
		nestedIntField(usage, "output_tokens_details", "reasoning_tokens"),
		nestedIntField(usage, "completion_tokens_details", "reasoning_tokens"),
	)
	return input, cached, output, reason
}

func apiEndpointURL(raw string, format APIFormat) (endpointURL, providerBaseURL, providerHost string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", fmt.Errorf("empty base url")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", "", "", fmt.Errorf("invalid base url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", "", fmt.Errorf("unsupported scheme")
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	basePath := stripAPIEndpointPath(u.Path)
	suffix := apiEndpointSuffix(format)
	endpoint := *u
	endpoint.Path = joinURLPath(basePath, suffix)
	provider := *u
	provider.Path = basePath
	return endpoint.String(), provider.String(), provider.Host, nil
}

func apiEndpointSuffix(format APIFormat) string {
	switch format {
	case APIFormatOpenAIResponses:
		return "/responses"
	case APIFormatAnthropic:
		return "/messages"
	default:
		return "/chat/completions"
	}
}

func stripAPIEndpointPath(path string) string {
	path = strings.TrimRight(path, "/")
	for _, suffix := range []string{"/chat/completions", "/responses", "/messages"} {
		if strings.HasSuffix(path, suffix) {
			return strings.TrimRight(strings.TrimSuffix(path, suffix), "/")
		}
	}
	return path
}

func joinURLPath(basePath, suffix string) string {
	basePath = strings.TrimRight(basePath, "/")
	suffix = "/" + strings.TrimLeft(suffix, "/")
	if basePath == "" {
		return suffix
	}
	return basePath + suffix
}

func applyAPIMetadata(s *Summary, opts Options) {
	format, _ := NormalizeAPIFormat(APIFormat(firstNonEmpty(string(opts.APIFormat), os.Getenv(envModelAPIFormat))))
	_, baseURL, host, _ := apiEndpointURL(firstNonEmpty(opts.ModelAPIBaseURL, os.Getenv(envModelAPIBaseURL)), format)
	s.CodexModelSource = "explicit"
	s.CodexModelProvider = string(format)
	s.CodexProviderHost = host
	s.CodexProviderBaseURL = baseURL
	s.CodexSandbox = "api"
	s.CodexEphemeral = false
	s.CodexSkipGitRepoCheck = false
	s.CodexDisabledFeatures = nil
	s.CodexInvocation = sanitizedAPIInvocation(opts, format, baseURL)
}

func sanitizedAPIInvocation(opts Options, format APIFormat, baseURL string) string {
	safe := struct {
		Backend         string `json:"backend"`
		APIFormat       string `json:"api_format"`
		Model           string `json:"model"`
		BaseURL         string `json:"base_url"`
		PromptFromSuite bool   `json:"prompt_from_suite"`
		KeyFromEnv      bool   `json:"key_from_env"`
	}{
		Backend:         string(BackendAPI),
		APIFormat:       string(format),
		Model:           strings.TrimSpace(opts.Model),
		BaseURL:         baseURL,
		PromptFromSuite: true,
		KeyFromEnv:      strings.TrimSpace(opts.ModelAPIKey) == "" && strings.TrimSpace(os.Getenv(envModelAPIKey)) != "",
	}
	b, err := json.Marshal(safe)
	if err != nil {
		return ""
	}
	return string(b)
}

func bodyPreview(b []byte, secret string) string {
	s := strings.Join(strings.Fields(string(b)), " ")
	if secret = strings.TrimSpace(secret); secret != "" {
		s = strings.ReplaceAll(s, secret, "[redacted]")
	}
	if len(s) > apiResponseBodyLimit {
		return s[:apiResponseBodyLimit] + "..."
	}
	return s
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func firstPositiveInt(values ...int) int {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}
