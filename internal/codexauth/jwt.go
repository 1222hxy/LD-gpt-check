package codexauth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func DecodeJWTPayload(token string) (map[string]any, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid jwt format")
	}
	payload := parts[1]
	data, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		padded := payload
		if rem := len(padded) % 4; rem > 0 {
			padded += strings.Repeat("=", 4-rem)
		}
		data, err = base64.URLEncoding.DecodeString(padded)
		if err != nil {
			return nil, err
		}
	}
	var claims map[string]any
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	if err := dec.Decode(&claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func EnrichAuthFromJWT(auth *CodexAuth, token string, isAccessToken bool) {
	if auth == nil || strings.TrimSpace(token) == "" {
		return
	}
	claims, err := DecodeJWTPayload(token)
	if err != nil {
		return
	}
	if email := GetString(claims, "email"); email != "" && auth.Email == "" {
		auth.Email = email
	}
	if exp := GetInt64(claims, "exp"); exp > 0 {
		t := time.Unix(exp, 0).UTC()
		if isAccessToken {
			auth.AccessExpiresAt = &t
		} else {
			auth.IDExpiresAt = &t
		}
	}
	openaiAuth := GetMap(claims, "https://api.openai.com/auth")
	if openaiAuth == nil {
		if auth.ChatGPTUserID == "" {
			auth.ChatGPTUserID = GetString(claims, "sub")
		}
		return
	}
	if auth.AccountID == "" {
		auth.AccountID = GetString(openaiAuth, "chatgpt_account_id")
	}
	if auth.ChatGPTUserID == "" {
		auth.ChatGPTUserID = FirstNonEmpty(
			GetString(openaiAuth, "chatgpt_user_id"),
			GetString(openaiAuth, "user_id"),
			GetString(claims, "sub"),
		)
	}
	if auth.PlanType == "" {
		auth.PlanType = GetString(openaiAuth, "chatgpt_plan_type")
	}
	if auth.OrgID == "" {
		auth.OrgID = GetString(openaiAuth, "poid")
	}
	if auth.OrgID == "" {
		auth.OrgID = PickDefaultOrganizationID(openaiAuth)
	}
}

func PickDefaultOrganizationID(openaiAuth map[string]any) string {
	orgs, ok := openaiAuth["organizations"].([]any)
	if !ok || len(orgs) == 0 {
		return ""
	}
	for _, item := range orgs {
		org, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if isDefault, _ := org["is_default"].(bool); isDefault {
			return GetString(org, "id")
		}
	}
	if first, ok := orgs[0].(map[string]any); ok {
		return GetString(first, "id")
	}
	return ""
}

func AccessTokenStatus(auth *CodexAuth) string {
	if auth == nil || auth.AccessToken == "" {
		return "missing"
	}
	if auth.AccessExpiresAt == nil {
		return "unknown"
	}
	now := time.Now()
	if now.After(auth.AccessExpiresAt.Add(120 * time.Second)) {
		return "expired"
	}
	if time.Until(*auth.AccessExpiresAt) < 3*time.Minute {
		return "expiring"
	}
	return "valid"
}

func MaskEmail(email string) string {
	email = strings.TrimSpace(email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[1] == "" {
		return ""
	}
	name := parts[0]
	if len([]rune(name)) <= 1 {
		return "*@" + parts[1]
	}
	runes := []rune(name)
	return string(runes[:1]) + "***@" + parts[1]
}

func MaskToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	r := []rune(value)
	if len(r) <= 8 {
		return "****"
	}
	return string(r[:4]) + "..." + string(r[len(r)-4:])
}
