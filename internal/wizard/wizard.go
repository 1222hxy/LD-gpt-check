package wizard

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/haowang02/ld-gpt-check/internal/api"
	"github.com/haowang02/ld-gpt-check/internal/auth"
	"github.com/haowang02/ld-gpt-check/internal/config"
	"github.com/haowang02/ld-gpt-check/internal/i18n"
	"github.com/haowang02/ld-gpt-check/internal/report"
	"github.com/haowang02/ld-gpt-check/internal/runner"
	"github.com/haowang02/ld-gpt-check/internal/system"
)

type Options struct {
	Version string
	Lang    i18n.Lang
	Stdin   io.Reader
	Stdout  io.Writer
}

func Run(ctx context.Context, opts Options) error {
	in := opts.Stdin
	if in == nil {
		in = os.Stdin
	}
	out := opts.Stdout
	if out == nil {
		out = os.Stdout
	}

	reader := bufio.NewReader(in)
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	lang := i18n.Detect(firstNonEmpty(string(opts.Lang), cfg.Language))
	l := i18n.New(lang)
	configPath, err := config.Path()
	if err != nil {
		return err
	}

	fmt.Fprintln(out, l.S("wizard_title"))
	fmt.Fprintln(out, "----------------")
	fmt.Fprintf(out, l.S("config_path")+"\n\n", configPath)

	oldAPIBase := cfg.APIBaseURL
	cfg.Language = string(lang)
	apiBase := config.DefaultAPIBaseURL()
	cfg.APIBaseURL = apiBase
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Fprintf(out, l.S("wizard_api_using")+"\n", apiBase)

	if cfg.AccessToken != "" {
		name := cfg.User.Username
		if name == "" {
			name = cfg.User.ID
		}
		fmt.Fprintf(out, l.S("wizard_existing_credential")+"\n", fallback(name, l.S("wizard_unknown_user")))
		reloginDefault := strings.TrimRight(oldAPIBase, "/") != strings.TrimRight(apiBase, "/")
		if reloginDefault {
			fmt.Fprintln(out, l.S("wizard_api_changed"))
		}
		if yes, err := promptBool(reader, out, l, l.S("wizard_relogin"), reloginDefault); err != nil {
			return err
		} else if yes {
			user, err := auth.LoginWithOptions(ctx, auth.LoginOptions{APIBaseURL: apiBase, Lang: lang, Stdout: out})
			if err != nil {
				return err
			}
			fmt.Fprintf(out, l.S("login_success")+"\n", user.Username, user.ID)
			fmt.Fprintf(out, l.S("credential_saved")+"\n", configPath)
			cfg, _ = config.Load()
		}
	} else {
		if yes, err := promptBool(reader, out, l, l.S("wizard_login_now"), true); err != nil {
			return err
		} else if yes {
			user, err := auth.LoginWithOptions(ctx, auth.LoginOptions{APIBaseURL: apiBase, Lang: lang, Stdout: out})
			if err != nil {
				return err
			}
			fmt.Fprintf(out, l.S("login_success")+"\n", user.Username, user.ID)
			fmt.Fprintf(out, l.S("credential_saved")+"\n", configPath)
			cfg, _ = config.Load()
		}
	}

	if ok, err := promptBool(reader, out, l, l.S("wizard_run_now"), true); err != nil {
		return err
	} else if !ok {
		fmt.Fprintln(out, l.S("wizard_done_next"))
		return nil
	}

	model, err := promptString(reader, out, l, l.S("wizard_model"), "gpt-5.5")
	if err != nil {
		return err
	}
	effort, err := promptEffort(reader, out, l, l.S("wizard_effort"), "medium")
	if err != nil {
		return err
	}
	tests, err := promptInt(reader, out, l, l.S("wizard_tests"), 1, 1, runner.MaxTests)
	if err != nil {
		return err
	}
	timeout, err := promptDuration(reader, out, l, l.S("wizard_timeout"), runner.DefaultTimeout)
	if err != nil {
		return err
	}

	uploadDefault := cfg.AccessToken != ""
	upload, err := promptBool(reader, out, l, l.S("wizard_upload"), uploadDefault)
	if err != nil {
		return err
	}
	if upload && cfg.AccessToken == "" {
		return fmt.Errorf("%s", l.S("wizard_upload_needs_login"))
	}

	summary, err := runner.Run(ctx, runner.Options{
		Model:           model,
		ReasoningEffort: effort,
		Tests:           tests,
		Timeout:         timeout,
		Lang:            lang,
	})
	if err != nil {
		return err
	}
	report.PrintTableWithLang(summary, lang)

	if upload {
		payload := api.PayloadFromSummary(opts.Version, summary, runtime.GOOS, runtime.GOARCH, system.CodexVersion())
		resp, err := api.NewWithLang(cfg.APIBaseURL, cfg.AccessToken, lang).UploadRun(ctx, payload)
		if err != nil {
			return err
		}
		if id, _ := resp["id"].(string); id != "" {
			fmt.Fprintf(out, l.S("uploaded_run")+"\n", id)
		} else {
			fmt.Fprintln(out, l.S("uploaded_run_no_id"))
		}
	}

	fmt.Fprintln(out, l.S("wizard_done"))
	return nil
}

func promptString(r *bufio.Reader, out io.Writer, l i18n.Localizer, label, def string) (string, error) {
	for {
		fmt.Fprintf(out, "%s [%s]: ", label, def)
		s, err := readLine(r)
		if err != nil {
			return "", err
		}
		if s == "" {
			s = def
		}
		s = strings.TrimSpace(s)
		if s != "" {
			return s, nil
		}
		fmt.Fprintln(out, l.S("prompt_non_empty"))
	}
}

func promptBool(r *bufio.Reader, out io.Writer, l i18n.Localizer, label string, def bool) (bool, error) {
	suffix := l.BoolSuffix(def)
	for {
		fmt.Fprintf(out, "%s [%s]: ", label, suffix)
		s, err := readLine(r)
		if err != nil {
			return false, err
		}
		if ok, parsed := i18n.ParseBoolInput(s, def); parsed {
			return ok, nil
		}
		fmt.Fprintln(out, l.S("prompt_bool"))
	}
}

func promptEffort(r *bufio.Reader, out io.Writer, l i18n.Localizer, label, def string) (string, error) {
	for {
		v, err := promptString(r, out, l, label+" (low/medium/high/xhigh)", def)
		if err != nil {
			return "", err
		}
		if runner.ValidReasoningEffort(v) {
			return v, nil
		}
		fmt.Fprintln(out, l.S("prompt_effort"))
	}
}

func promptInt(r *bufio.Reader, out io.Writer, l i18n.Localizer, label string, def, min, max int) (int, error) {
	for {
		v, err := promptString(r, out, l, label, strconv.Itoa(def))
		if err != nil {
			return 0, err
		}
		n, err := strconv.Atoi(v)
		if err == nil && n >= min && n <= max {
			return n, nil
		}
		fmt.Fprintf(out, l.S("prompt_int_range")+"\n", min, max)
	}
}

func promptDuration(r *bufio.Reader, out io.Writer, l i18n.Localizer, label string, def time.Duration) (time.Duration, error) {
	for {
		v, err := promptString(r, out, l, label, def.String())
		if err != nil {
			return 0, err
		}
		d, err := time.ParseDuration(v)
		if err == nil && d > 0 {
			return d, nil
		}
		fmt.Fprintln(out, l.S("prompt_duration"))
	}
}

func readLine(r *bufio.Reader) (string, error) {
	s, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	if err == io.EOF && s == "" {
		return "", io.ErrUnexpectedEOF
	}
	return strings.TrimSpace(s), nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func fallback(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
