package codexauth

import (
	"strings"
	"time"
)

type CodexAuth struct {
	AuthPath     string
	AuthMode     string
	LastRefresh  string
	AccessToken  string
	RefreshToken string
	IDToken      string
	AccountID    string

	Email         string
	ChatGPTUserID string
	PlanType      string
	OrgID         string

	AccessExpiresAt *time.Time
	IDExpiresAt     *time.Time

	HasAPIKey              bool
	HasPersonalAccessToken bool
	HasAgentIdentity       bool
}

type CodexAuthInfo struct {
	AuthPath        string
	AuthMode        string
	HasAccessToken  bool
	HasRefreshToken bool
	HasIDToken      bool
	Email           string
	PlanType        string
	AccountID       string
	UserID          string
	OrgID           string
	AccessExpiresAt string
	AccessStatus    string
}

func Parse(raw map[string]any, path string) *CodexAuth {
	auth := &CodexAuth{
		AuthPath: path,
		AuthMode: PickString(raw,
			"auth_mode",
			"authMode",
		),
		LastRefresh: PickString(raw, "last_refresh", "lastRefresh"),
		AccessToken: PickString(raw,
			"tokens.access_token",
			"tokens.accessToken",
			"access_token",
			"accessToken",
			"token",
		),
		RefreshToken: PickString(raw,
			"tokens.refresh_token",
			"tokens.refreshToken",
			"refresh_token",
			"refreshToken",
		),
		IDToken: PickString(raw,
			"tokens.id_token",
			"tokens.idToken",
			"id_token",
			"idToken",
		),
		AccountID: PickString(raw,
			"tokens.account_id",
			"tokens.accountId",
			"account_id",
			"accountId",
		),
		HasAPIKey:              PickString(raw, "OPENAI_API_KEY", "openai_api_key") != "",
		HasPersonalAccessToken: PickString(raw, "personal_access_token", "personalAccessToken") != "",
		HasAgentIdentity:       HasPath(raw, "agent_identity") || HasPath(raw, "agentIdentity"),
	}
	if agent := GetMap(raw, "agent_identity"); agent != nil {
		enrichFromAgentIdentity(auth, agent)
	}
	if agent := GetMap(raw, "agentIdentity"); agent != nil {
		enrichFromAgentIdentity(auth, agent)
	}
	EnrichAuthFromJWT(auth, auth.AccessToken, true)
	EnrichAuthFromJWT(auth, auth.IDToken, false)
	return auth
}

func (a *CodexAuth) Info() CodexAuthInfo {
	info := CodexAuthInfo{
		AuthPath:        a.AuthPath,
		AuthMode:        a.AuthMode,
		HasAccessToken:  a.AccessToken != "",
		HasRefreshToken: a.RefreshToken != "",
		HasIDToken:      a.IDToken != "",
		Email:           MaskEmail(a.Email),
		PlanType:        a.PlanType,
		AccountID:       MaskToken(a.AccountID),
		UserID:          MaskToken(a.ChatGPTUserID),
		OrgID:           MaskToken(a.OrgID),
		AccessStatus:    AccessTokenStatus(a),
	}
	if a.AccessExpiresAt != nil {
		info.AccessExpiresAt = a.AccessExpiresAt.UTC().Format(time.RFC3339)
	}
	return info
}

func enrichFromAgentIdentity(auth *CodexAuth, agent map[string]any) {
	if auth.AccountID == "" {
		auth.AccountID = GetString(agent, "account_id")
	}
	if auth.ChatGPTUserID == "" {
		auth.ChatGPTUserID = GetString(agent, "chatgpt_user_id")
	}
	if auth.Email == "" {
		auth.Email = GetString(agent, "email")
	}
	if auth.PlanType == "" {
		auth.PlanType = GetString(agent, "plan_type")
	}
}

func PickString(raw map[string]any, paths ...string) string {
	for _, path := range paths {
		if v := GetNestedString(raw, path); strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func HasPath(raw map[string]any, path string) bool {
	_, ok := getNested(raw, path)
	return ok
}

func GetNestedString(raw map[string]any, path string) string {
	v, ok := getNested(raw, path)
	if !ok {
		return ""
	}
	return stringFromValue(v)
}

func GetMap(raw map[string]any, key string) map[string]any {
	m, _ := raw[key].(map[string]any)
	return m
}

func GetString(raw map[string]any, key string) string {
	return strings.TrimSpace(stringFromValue(raw[key]))
}

func GetInt64(raw map[string]any, key string) int64 {
	switch v := raw[key].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case jsonNumber:
		i, _ := v.Int64()
		return i
	case string:
		var n int64
		for _, r := range strings.TrimSpace(v) {
			if r < '0' || r > '9' {
				return 0
			}
			n = n*10 + int64(r-'0')
		}
		return n
	default:
		return 0
	}
}

type jsonNumber interface {
	Int64() (int64, error)
}

func getNested(raw map[string]any, path string) (any, bool) {
	if path == "" {
		return nil, false
	}
	parts := strings.Split(path, ".")
	var cur any = raw
	for _, part := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := m[part]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

func stringFromValue(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return ""
	}
}

func FirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
