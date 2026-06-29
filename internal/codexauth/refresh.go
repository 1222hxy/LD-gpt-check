package codexauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	OpenAICodexClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
	OpenAITokenURL      = "https://auth.openai.com/oauth/token"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

func BuildRefreshRequest(refreshToken string) url.Values {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", OpenAICodexClientID)
	form.Set("refresh_token", strings.TrimSpace(refreshToken))
	form.Set("scope", "openid profile email")
	return form
}

func RefreshTokens(ctx context.Context, client *http.Client, refreshToken string) (*TokenResponse, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("refresh_token is required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	body := strings.NewReader(BuildRefreshRequest(refreshToken).Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, OpenAITokenURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("token refresh failed: HTTP %d", resp.StatusCode)
	}
	var out TokenResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out.AccessToken == "" {
		return nil, fmt.Errorf("token refresh response missing access_token")
	}
	return &out, nil
}

func ApplyTokenResponse(auth *CodexAuth, resp *TokenResponse) {
	if auth == nil || resp == nil {
		return
	}
	if resp.AccessToken != "" {
		auth.AccessToken = resp.AccessToken
	}
	if resp.RefreshToken != "" {
		auth.RefreshToken = resp.RefreshToken
	}
	if resp.IDToken != "" {
		auth.IDToken = resp.IDToken
	}
	auth.AccessExpiresAt = nil
	auth.IDExpiresAt = nil
	if resp.ExpiresIn > 0 && auth.AccessToken != "" {
		t := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second).UTC()
		auth.AccessExpiresAt = &t
	}
	EnrichAuthFromJWT(auth, auth.AccessToken, true)
	EnrichAuthFromJWT(auth, auth.IDToken, false)
}
