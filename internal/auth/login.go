package auth

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/haowang02/ld-gpt-check/internal/api"
	"github.com/haowang02/ld-gpt-check/internal/config"
	"github.com/haowang02/ld-gpt-check/internal/i18n"
	"github.com/haowang02/ld-gpt-check/internal/system"
)

func Login(ctx context.Context, apiBaseURL string) (config.User, error) {
	return LoginWithOptions(ctx, LoginOptions{APIBaseURL: apiBaseURL})
}

type LoginOptions struct {
	APIBaseURL string
	Lang       i18n.Lang
	Stdout     io.Writer
}

func LoginWithOptions(ctx context.Context, opts LoginOptions) (config.User, error) {
	out := opts.Stdout
	if out == nil {
		out = os.Stdout
	}
	lang := i18n.Normalize(string(opts.Lang))
	l := i18n.New(lang)
	apiBaseURL := opts.APIBaseURL
	client := api.NewWithLang(apiBaseURL, "", lang)
	start, err := client.DeviceStart(ctx)
	if err != nil {
		return config.User{}, err
	}
	if start.DeviceCode == "" {
		return config.User{}, fmt.Errorf("%s", l.S("auth_missing_device_code"))
	}
	if start.UserCode == "" {
		return config.User{}, fmt.Errorf("%s", l.S("auth_missing_user_code"))
	}
	if start.VerificationURI == "" && start.VerificationURIComplete == "" {
		return config.User{}, fmt.Errorf("%s", l.S("auth_missing_verification_uri"))
	}

	fmt.Fprintln(out, l.S("auth_opening_browser"))
	if start.VerificationURIComplete != "" {
		_ = system.OpenBrowser(start.VerificationURIComplete)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, l.S("auth_manual_open"))
	if start.VerificationURI != "" {
		fmt.Fprintln(out, start.VerificationURI)
	} else {
		fmt.Fprintln(out, start.VerificationURIComplete)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, l.S("auth_user_code"))
	fmt.Fprintln(out, start.UserCode)
	fmt.Fprintln(out)

	interval := start.Interval
	if interval <= 0 {
		interval = 3
	}
	if interval > 30 {
		interval = 30
	}
	deadline := time.Now().Add(time.Duration(start.ExpiresIn) * time.Second)
	if start.ExpiresIn <= 0 {
		deadline = time.Now().Add(10 * time.Minute)
	}

	transientErrors := 0
	for {
		if time.Now().After(deadline) {
			return config.User{}, fmt.Errorf("%s", l.S("auth_expired"))
		}
		select {
		case <-ctx.Done():
			return config.User{}, ctx.Err()
		case <-time.After(time.Duration(interval) * time.Second):
		}
		poll, err := client.DevicePoll(ctx, start.DeviceCode)
		if err != nil {
			transientErrors++
			if transientErrors >= 3 {
				return config.User{}, err
			}
			continue
		}
		transientErrors = 0
		switch poll.Status {
		case "pending":
			continue
		case "slow_down":
			if interval < 30 {
				interval++
			}
			continue
		case "expired":
			return config.User{}, fmt.Errorf("%s", l.S("auth_expired"))
		case "authorized":
			if poll.AccessToken == "" {
				return config.User{}, fmt.Errorf("%s", l.S("auth_missing_access_token"))
			}
			if poll.User.ID == "" {
				return config.User{}, fmt.Errorf("%s", l.S("auth_missing_user_id"))
			}
			cfg := config.Config{
				APIBaseURL:  apiBaseURL,
				AccessToken: poll.AccessToken,
				Language:    string(lang),
				User:        poll.User,
			}
			if err := config.Save(cfg); err != nil {
				return config.User{}, err
			}
			return poll.User, nil
		default:
			return config.User{}, fmt.Errorf("%s", l.S("auth_unexpected_status", poll.Status))
		}
	}
}
