package codexauth

import (
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveAuthPathPriority(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	if got := ResolveAuthPath("custom.json"); got != "custom.json" {
		t.Fatalf("explicit path = %q", got)
	}
	if got := ResolveAuthPath(""); got != filepath.Join(home, "auth.json") {
		t.Fatalf("CODEX_HOME path = %q", got)
	}
}

func TestParseCodexAuthAndJWTClaims(t *testing.T) {
	exp := time.Now().Add(time.Hour).Unix()
	token := testJWT(map[string]any{
		"sub":   "user_sub",
		"email": "person@example.com",
		"exp":   exp,
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "acct_123456789",
			"chatgpt_user_id":    "user_abcdef",
			"chatgpt_plan_type":  "plus",
			"organizations":      []any{map[string]any{"id": "org_1"}, map[string]any{"id": "org_default", "is_default": true}},
		},
	})
	auth := Parse(map[string]any{
		"auth_mode": "chatgpt",
		"tokens": map[string]any{
			"access_token": token,
			"refreshToken": "refresh_secret",
			"id_token":     testJWT(map[string]any{"exp": exp + 60}),
			"account_id":   "",
		},
		"personal_access_token": "pat_secret",
		"agent_identity": map[string]any{
			"email": "agent@example.com",
		},
	}, "/tmp/auth.json")

	if auth.AccessToken != token || auth.RefreshToken != "refresh_secret" || !auth.HasPersonalAccessToken || !auth.HasAgentIdentity {
		t.Fatalf("tokens/features = %#v", auth)
	}
	if auth.Email != "agent@example.com" || auth.AccountID != "acct_123456789" || auth.ChatGPTUserID != "user_abcdef" || auth.PlanType != "plus" || auth.OrgID != "org_default" {
		t.Fatalf("enriched auth = %#v", auth)
	}
	if AccessTokenStatus(auth) != "valid" {
		t.Fatalf("status = %s", AccessTokenStatus(auth))
	}
	info := auth.Info()
	if info.Email != "a***@example.com" || strings.Contains(info.AccountID, "123456789") {
		t.Fatalf("info not masked: %#v", info)
	}
}

func TestAccessTokenStatus(t *testing.T) {
	if got := AccessTokenStatus(&CodexAuth{}); got != "missing" {
		t.Fatalf("missing = %s", got)
	}
	if got := AccessTokenStatus(&CodexAuth{AccessToken: "not-jwt"}); got != "unknown" {
		t.Fatalf("unknown = %s", got)
	}
	expired := time.Now().Add(-5 * time.Minute)
	if got := AccessTokenStatus(&CodexAuth{AccessToken: "token", AccessExpiresAt: &expired}); got != "expired" {
		t.Fatalf("expired = %s", got)
	}
}

func testJWT(claims map[string]any) string {
	header, _ := json.Marshal(map[string]any{"alg": "none"})
	payload, _ := json.Marshal(claims)
	return base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}
