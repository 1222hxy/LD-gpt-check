package codexauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	ChatGPTBaseURL                = "https://chatgpt.com"
	AccountsCheckPath             = "/backend-api/accounts/check/v4-2023-04-27"
	SubscriptionsPath             = "/backend-api/subscriptions"
	CodexResponsesEndpoint        = "https://chatgpt.com/backend-api/codex/responses"
	CodexResponsesProviderBaseURL = "https://chatgpt.com/backend-api/codex"
)

type ChatGPTAccountInfo struct {
	PlanType              string
	Email                 string
	SubscriptionExpiresAt string
}

type SubscriptionInfo struct {
	PlanType    string `json:"plan_type"`
	ActiveUntil string `json:"active_until"`
	WillRenew   bool   `json:"will_renew"`
	ID          string `json:"id"`
}

func FetchAccountInfo(ctx context.Context, client *http.Client, accessToken, orgID string) (*ChatGPTAccountInfo, error) {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ChatGPTBaseURL+AccountsCheckPath, nil)
	if err != nil {
		return nil, err
	}
	setChatGPTHeaders(req, accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("accounts check failed: HTTP %d", resp.StatusCode)
	}
	var raw map[string]any
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	return ParseAccountInfo(raw, orgID), nil
}

func ParseAccountInfo(raw map[string]any, orgID string) *ChatGPTAccountInfo {
	accounts := GetMap(raw, "accounts")
	if accounts == nil {
		return nil
	}
	if orgID != "" {
		if acct := GetMap(accounts, orgID); acct != nil {
			if info := ExtractAccountInfo(acct); info.PlanType != "" {
				return info
			}
		}
	}
	var defaultInfo, paidInfo, anyInfo *ChatGPTAccountInfo
	for _, value := range accounts {
		acct, ok := value.(map[string]any)
		if !ok {
			continue
		}
		info := ExtractAccountInfo(acct)
		if info.PlanType == "" {
			continue
		}
		if anyInfo == nil {
			anyInfo = info
		}
		if accountObj := GetMap(acct, "account"); accountObj != nil {
			if isDefault, _ := accountObj["is_default"].(bool); isDefault {
				defaultInfo = info
			}
		}
		if info.PlanType != "free" && paidInfo == nil {
			paidInfo = info
		}
	}
	if defaultInfo != nil {
		return defaultInfo
	}
	if paidInfo != nil {
		return paidInfo
	}
	return anyInfo
}

func ExtractAccountInfo(acct map[string]any) *ChatGPTAccountInfo {
	info := &ChatGPTAccountInfo{}
	if account := GetMap(acct, "account"); account != nil {
		info.PlanType = GetString(account, "plan_type")
		info.Email = GetString(account, "email")
	}
	if entitlement := GetMap(acct, "entitlement"); entitlement != nil {
		if info.PlanType == "" {
			info.PlanType = GetString(entitlement, "subscription_plan")
		}
		info.SubscriptionExpiresAt = GetString(entitlement, "expires_at")
	}
	return info
}

func FetchSubscription(ctx context.Context, client *http.Client, accessToken, accountID string) (*SubscriptionInfo, error) {
	if strings.TrimSpace(accountID) == "" {
		return nil, fmt.Errorf("account_id is required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	u, err := url.Parse(ChatGPTBaseURL + SubscriptionsPath)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("account_id", accountID)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	setChatGPTHeaders(req, accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("subscription check failed: HTTP %d", resp.StatusCode)
	}
	var out SubscriptionInfo
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func setChatGPTHeaders(req *http.Request, accessToken string) {
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	req.Header.Set("Origin", "https://chatgpt.com")
	req.Header.Set("Referer", "https://chatgpt.com/")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "codex_cli_rs/0.125.0")
}
