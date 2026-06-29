package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/1222hxy/LD-gpt-check/internal/api"
	"github.com/1222hxy/LD-gpt-check/internal/auth"
	"github.com/1222hxy/LD-gpt-check/internal/config"
	"github.com/1222hxy/LD-gpt-check/internal/i18n"
	"github.com/1222hxy/LD-gpt-check/internal/questions"
	"github.com/1222hxy/LD-gpt-check/internal/report"
	"github.com/1222hxy/LD-gpt-check/internal/runner"
	"github.com/1222hxy/LD-gpt-check/internal/system"
	"github.com/1222hxy/LD-gpt-check/internal/wizard"
)

var version = "0.2.1"

var runWizard = wizard.Run

func main() {
	lang := currentLang()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], lang); err != nil {
		l := i18n.New(lang)
		if err == context.Canceled {
			err = fmt.Errorf("%s", l.S("canceled"))
		}
		fmt.Fprintln(os.Stderr, l.S("error_prefix", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, lang i18n.Lang) error {
	l := i18n.New(lang)
	if len(args) == 0 {
		return runWizard(ctx, wizard.Options{Version: version, Lang: lang})
	}
	switch args[0] {
	case "setup", "wizard":
		return wizardCmd(ctx, args[1:], lang)
	case "run":
		return runCmd(ctx, args[1:], lang)
	case "login":
		return loginCmd(ctx, args[1:], lang)
	case "whoami":
		return whoamiCmd(ctx, args[1:], lang)
	case "config":
		return configCmd(ctx, args[1:], lang)
	case "logout":
		return logoutCmd(ctx, args[1:], lang)
	case "version", "--version", "-v":
		fmt.Println(version)
		return nil
	case "help", "-h", "--help":
		usage(l)
		return nil
	default:
		return fmt.Errorf("%s", l.S("unknown_command", args[0]))
	}
}

func wizardCmd(ctx context.Context, args []string, lang i18n.Lang) error {
	l := i18n.New(lang)
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	langFlag := fs.String("lang", string(lang), l.S("flag_lang"))
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%s", l.S("unexpected_args", fs.Args()))
	}
	return runWizard(ctx, wizard.Options{Version: version, Lang: i18n.Normalize(*langFlag)})
}

func runCmd(ctx context.Context, args []string, lang i18n.Lang) error {
	l := i18n.New(lang)
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	model := fs.String("m", "", l.S("flag_model"))
	modelLong := fs.String("model", "", l.S("flag_model"))
	effort := fs.String("r", "medium", l.S("flag_effort"))
	effortLong := fs.String("reasoning-effort", "", l.S("flag_effort"))
	backend := fs.String("backend", string(runner.BackendAuto), l.S("flag_backend"))
	apiFormat := fs.String("api-format", "", l.S("flag_api_format"))
	modelAPIBase := fs.String("model-api-base-url", os.Getenv("LD_GPT_CHECK_MODEL_API_BASE_URL"), l.S("flag_model_api_base"))
	modelAPIKey := fs.String("model-api-key", "", l.S("flag_model_api_key"))
	tests := fs.Int("n", runner.DefaultTests, l.S("flag_tests"))
	testsLong := fs.Int("tests", 0, l.S("flag_tests"))
	timeout := fs.Duration("timeout", runner.DefaultTimeout, l.S("flag_timeout"))
	suite := fs.String("suite", questions.DefaultSuite, "question suite ids, comma-separated")
	questionFile := fs.String("question-file", "", "local question bank JSON file")
	questionURL := fs.String("question-url", "", "remote HTTPS question bank JSON URL")
	questionCache := fs.String("question-cache", questions.DefaultCacheDir(), "question bank cache directory")
	noRemoteQuestions := fs.Bool("no-remote-questions", false, "do not fetch the default remote question bank")
	listSuites := fs.Bool("list-suites", false, "list available question suites")
	upload := fs.Bool("upload", false, l.S("flag_upload"))
	anonymous := fs.Bool("anonymous", false, l.S("flag_anonymous"))
	jsonOut := fs.Bool("json", false, l.S("flag_json"))
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%s", l.S("unexpected_args", fs.Args()))
	}
	if *modelLong != "" {
		*model = *modelLong
	}
	if *effortLong != "" {
		*effort = *effortLong
	}
	if *testsLong > 0 {
		*tests = *testsLong
	}
	if *tests <= 0 {
		return fmt.Errorf("%s", l.S("flag_count_positive", "tests"))
	}
	if *timeout <= 0 {
		return fmt.Errorf("%s", l.S("timeout_positive"))
	}
	remoteURL := strings.TrimSpace(*questionURL)
	fallbackRemote := false
	if remoteURL == "" && !*noRemoteQuestions {
		apiBase := config.DefaultAPIBaseURL()
		if cfg, err := config.Load(); err == nil && strings.TrimSpace(cfg.APIBaseURL) != "" {
			apiBase = cfg.APIBaseURL
		}
		remoteURL = strings.TrimRight(apiBase, "/") + "/api/v1/questions"
		fallbackRemote = true
	}
	allQuestions, err := questions.Load(ctx, questions.LoadOptions{
		File:                  *questionFile,
		URL:                   remoteURL,
		CacheDir:              *questionCache,
		FallbackOnRemoteError: fallbackRemote,
	})
	if err != nil {
		return err
	}
	if *listSuites {
		for _, q := range allQuestions {
			fmt.Printf("%s\t%s\t%s\n", q.ID, q.Version, q.Title)
		}
		return nil
	}
	selected, err := questions.Select(allQuestions, *suite)
	if err != nil {
		return err
	}
	resolvedProgressModel := progressModel(*model)
	if *upload && resolvedProgressModel == "" {
		return fmt.Errorf("%s", l.S("api_upload_model_required"))
	}

	progressOut := os.Stdout
	if *jsonOut {
		progressOut = os.Stderr
	}
	progress := report.PrintProgress(progressOut, lang, resolvedProgressModel, *effort, report.ColorEnabled(progressOut))
	summary, err := runner.Run(ctx, runner.Options{
		Model:           *model,
		ReasoningEffort: *effort,
		Tests:           *tests,
		Timeout:         *timeout,
		Lang:            lang,
		Backend:         runner.Backend(*backend),
		APIFormat:       runner.APIFormat(*apiFormat),
		ModelAPIBaseURL: *modelAPIBase,
		ModelAPIKey:     *modelAPIKey,
		QuestionSuite:   *suite,
		Questions:       selected,
		Progress:        progress,
	})
	if err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(summary); err != nil {
			return err
		}
	} else {
		report.PrintTableWithLangColor(summary, lang, report.ColorEnabled(os.Stdout))
	}
	if *upload {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		lang = i18n.Detect(cfg.Language)
		l = i18n.New(lang)
		codexVersion := system.UploadCodexVersion(summary.CodexSandbox)
		payload := api.PayloadFromSummary(version, summary, runtime.GOOS, runtime.GOARCH, codexVersion)
		payload.Anonymous = *anonymous
		client := api.NewWithLang(cfg.APIBaseURL, cfg.AccessToken, lang)
		resp, err := client.UploadRun(ctx, payload)
		if err != nil {
			return err
		}
		out := os.Stdout
		if *jsonOut {
			out = os.Stderr
		}
		if id, _ := resp["id"].(string); id != "" {
			fmt.Fprintf(out, l.S("uploaded_run")+"\n", id)
		} else {
			fmt.Fprintln(out, l.S("uploaded_run_no_id"))
		}
	}
	return nil
}

func progressModel(model string) string {
	if system.ConcreteCodexModel(model) {
		return model
	}
	configured, err := system.CodexConfiguredModel()
	if err == nil && system.ConcreteCodexModel(configured) {
		return configured
	}
	return ""
}

func loginCmd(ctx context.Context, args []string, lang i18n.Lang) error {
	l := i18n.New(lang)
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	apiBase := fs.String("api-base-url", config.DefaultAPIBaseURL(), l.S("flag_api_base"))
	langFlag := fs.String("lang", string(lang), l.S("flag_lang"))
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%s", l.S("unexpected_args", fs.Args()))
	}
	lang = i18n.Normalize(*langFlag)
	l = i18n.New(lang)
	user, err := auth.LoginWithOptions(ctx, auth.LoginOptions{APIBaseURL: *apiBase, Lang: lang, Stdout: os.Stdout})
	if err != nil {
		return err
	}
	fmt.Printf(l.S("login_success")+"\n", user.Username, user.ID)
	if path, err := config.Path(); err == nil {
		fmt.Printf(l.S("credential_saved")+"\n", path)
	}
	return nil
}

func whoamiCmd(ctx context.Context, args []string, lang i18n.Lang) error {
	l := i18n.New(lang)
	fs := flag.NewFlagSet("whoami", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%s", l.S("unexpected_args", fs.Args()))
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	me, err := api.NewWithLang(cfg.APIBaseURL, cfg.AccessToken, lang).Me(ctx)
	if err != nil {
		return err
	}
	name := me.User.Username
	if name == "" {
		name = me.User.ID
	}
	fmt.Printf("%s (%s)\n", name, me.User.ID)
	return nil
}

func configCmd(ctx context.Context, args []string, lang i18n.Lang) error {
	_ = ctx
	l := i18n.New(lang)
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%s", l.S("unexpected_args", fs.Args()))
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	path, err := config.Path()
	if err != nil {
		return err
	}
	fmt.Printf(l.S("config_path")+"\n", path)
	fmt.Printf(l.S("config_api_base")+"\n", cfg.APIBaseURL)
	fmt.Printf(l.S("config_language")+"\n", cfg.Language)
	if cfg.AccessToken == "" {
		fmt.Println(l.S("config_not_logged_in"))
		return nil
	}
	name := cfg.User.Username
	if name == "" {
		name = cfg.User.ID
	}
	fmt.Print(l.S("config_logged_in"))
	if name != "" {
		fmt.Printf("（%s）", name)
	}
	fmt.Println()
	return nil
}

func logoutCmd(ctx context.Context, args []string, lang i18n.Lang) error {
	l := i18n.New(lang)
	fs := flag.NewFlagSet("logout", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%s", l.S("unexpected_args", fs.Args()))
	}
	cfg, _ := config.Load()
	if cfg.AccessToken != "" {
		lang = i18n.Detect(cfg.Language)
		_ = api.NewWithLang(cfg.APIBaseURL, cfg.AccessToken, lang).Logout(ctx)
	}
	if err := config.DeleteToken(); err != nil {
		return err
	}
	fmt.Println(l.S("logout_success"))
	if path, err := config.Path(); err == nil {
		fmt.Printf(l.S("credential_removed")+"\n", path)
	}
	return nil
}

func usage(l i18n.Localizer) {
	fmt.Println(l.S("usage"))
}

func currentLang() i18n.Lang {
	cfg, err := config.Load()
	if err != nil {
		return i18n.Detect("")
	}
	return i18n.Detect(cfg.Language)
}
