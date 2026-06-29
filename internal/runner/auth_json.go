package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/1222hxy/LD-gpt-check/internal/codexauth"
	"github.com/1222hxy/LD-gpt-check/internal/i18n"
	"github.com/1222hxy/LD-gpt-check/internal/questions"
	"github.com/1222hxy/LD-gpt-check/internal/system"
	"github.com/go-resty/resty/v2"
)

type authJSONBackend struct {
	auth   *codexauth.CodexAuth
	client *resty.Client
}

func newAuthJSONBackend(opts Options) (*authJSONBackend, error) {
	l := i18n.New(opts.Lang)
	model := strings.TrimSpace(opts.Model)
	if !system.ConcreteCodexModel(model) {
		return nil, fmt.Errorf("%s", l.S("runner_model_required"))
	}
	auth, err := codexauth.Load(opts.AuthPath)
	if err != nil {
		return nil, fmt.Errorf("%s", l.S("runner_auth_json_load_failed", codexauth.ResolveAuthPath(opts.AuthPath), err))
	}
	switch codexauth.AccessTokenStatus(auth) {
	case "missing":
		if strings.TrimSpace(auth.RefreshToken) == "" {
			return nil, fmt.Errorf("%s", l.S("runner_auth_json_access_missing", auth.AuthPath))
		}
		if err := refreshAuthJSON(context.Background(), opts, auth); err != nil {
			return nil, fmt.Errorf("%s", l.S("runner_auth_json_access_missing_refresh_failed", auth.AuthPath, err))
		}
	case "expired":
		if err := refreshAuthJSON(context.Background(), opts, auth); err != nil {
			return nil, fmt.Errorf("%s", l.S("runner_auth_json_access_expired", auth.AuthPath, err))
		}
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	client := resty.New().
		SetTimeout(timeout).
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetHeader("Originator", "codex_cli_rs").
		SetHeader("User-Agent", "codex_cli_rs/0.125.0")
	if opts.APIHTTPTransport != nil {
		client.SetTransport(opts.APIHTTPTransport)
	}
	return &authJSONBackend{auth: auth, client: client}, nil
}

func refreshAuthJSON(ctx context.Context, opts Options, auth *codexauth.CodexAuth) error {
	if strings.TrimSpace(auth.RefreshToken) == "" {
		return fmt.Errorf("refresh_token missing")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	if opts.APIHTTPTransport != nil {
		client.Transport = opts.APIHTTPTransport
	}
	resp, err := codexauth.RefreshTokens(ctx, client, auth.RefreshToken)
	if err != nil {
		return err
	}
	codexauth.ApplyTokenResponse(auth, resp)
	if status := codexauth.AccessTokenStatus(auth); status == "missing" || status == "expired" {
		return fmt.Errorf("refreshed access_token is %s", status)
	}
	return nil
}

func (b *authJSONBackend) runOne(ctx context.Context, opts Options, q questions.Question, index int) (CaseResult, error) {
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

func (b *authJSONBackend) request(ctx context.Context, opts Options, prompt string) (ParsedEvents, error) {
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
	l := i18n.New(opts.Lang)
	if err != nil {
		return ParsedEvents{}, fmt.Errorf("%s", l.S("runner_api_failed", classifyTransportError(err, opts.Lang)))
	}
	if resp == nil {
		return ParsedEvents{}, fmt.Errorf("%s", l.S("runner_api_failed", l.S("runner_api_empty_response")))
	}
	if resp.IsError() {
		status := resp.StatusCode()
		preview := bodyPreview(resp.Body(), b.auth.AccessToken)
		if status == httpStatusUnauthorized || status == httpStatusForbidden {
			return ParsedEvents{}, fmt.Errorf("%s", l.S("runner_auth_json_auth_failed", status, preview))
		}
		if shouldRetryStatus(status) {
			return ParsedEvents{}, fmt.Errorf("%s", l.S("runner_api_status_retry_exhausted", status, preview))
		}
		return ParsedEvents{}, fmt.Errorf("%s", l.S("runner_api_status_failed", status, preview))
	}
	var obj map[string]any
	dec := json.NewDecoder(strings.NewReader(string(resp.Body())))
	dec.UseNumber()
	if err := dec.Decode(&obj); err != nil {
		return ParsedEvents{}, fmt.Errorf("%s", l.S("runner_api_decode_failed", err))
	}
	parsed := parseAPIResponse(APIFormatOpenAIResponses, obj)
	parsed.EventTypes = []string{"auth_json.codex.responses"}
	return parsed, nil
}

func (b *authJSONBackend) doRequest(ctx context.Context, opts Options, prompt string) (*resty.Response, error) {
	return b.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+b.auth.AccessToken).
		SetBody(authJSONRequestBody(prompt, opts)).
		Post(codexauth.CodexResponsesEndpoint)
}

func authJSONRequestBody(prompt string, opts Options) map[string]any {
	body := map[string]any{
		"model": strings.TrimSpace(opts.Model),
		"input": prompt,
	}
	if effort := strings.TrimSpace(opts.ReasoningEffort); effort != "" {
		body["reasoning"] = map[string]any{"effort": effort}
	}
	return body
}
