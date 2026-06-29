package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/1222hxy/LD-gpt-check/internal/api"
	"github.com/1222hxy/LD-gpt-check/internal/auth"
	"github.com/1222hxy/LD-gpt-check/internal/codexauth"
	"github.com/1222hxy/LD-gpt-check/internal/config"
	"github.com/1222hxy/LD-gpt-check/internal/i18n"
	"github.com/1222hxy/LD-gpt-check/internal/questions"
	"github.com/1222hxy/LD-gpt-check/internal/report"
	"github.com/1222hxy/LD-gpt-check/internal/runner"
	"github.com/1222hxy/LD-gpt-check/internal/system"
	"github.com/1222hxy/LD-gpt-check/internal/updater"
	"github.com/1222hxy/LD-gpt-check/internal/wizard"
	"golang.org/x/term"
)

var version = "0.2.9"
var assetSuffix = ""
var gitCommit = ""
var gitCommitDate = ""
var gitModified = ""
var recentCommits = ""

var runWizard = wizard.Run
var runAutoUpdateCheck = autoUpdateCheck

func main() {
	lang := currentLang()
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		<-signals
		cancel()
		fmt.Fprintln(os.Stderr, i18n.New(lang).S("canceled"))
		os.Exit(130)
	}()
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
	if shouldAutoCheck(args) {
		if updated := runAutoUpdateCheck(ctx, lang); updated {
			return nil
		}
	}
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
	case "update":
		return updateCmd(ctx, args[1:], lang)
	case "doctor":
		return doctorCmd(ctx, args[1:], lang)
	case "version", "--version", "-v":
		printVersion(os.Stdout)
		return nil
	case "help", "-h", "--help":
		usage(l)
		return nil
	default:
		return fmt.Errorf("%s", l.S("unknown_command", args[0]))
	}
}

func printVersion(out io.Writer) {
	info := buildVersionInfo()
	fmt.Fprintln(out, info.Version)
	if info.Commit != "" {
		line := "commit: " + info.Commit
		if info.CommitDate != "" {
			line += " (" + info.CommitDate + ")"
		}
		if info.Modified {
			line += " dirty"
		}
		fmt.Fprintln(out, line)
	}
	if len(info.RecentCommits) > 0 {
		fmt.Fprintln(out, "recent commits:")
		for _, c := range info.RecentCommits {
			fmt.Fprintln(out, "  "+c)
		}
	}
}

type versionInfo struct {
	Version       string
	Commit        string
	CommitDate    string
	Modified      bool
	RecentCommits []string
}

func buildVersionInfo() versionInfo {
	info := versionInfo{
		Version:       version,
		Commit:        strings.TrimSpace(gitCommit),
		CommitDate:    strings.TrimSpace(gitCommitDate),
		Modified:      strings.EqualFold(strings.TrimSpace(gitModified), "true"),
		RecentCommits: splitRecentCommits(recentCommits),
	}
	if build, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range build.Settings {
			switch setting.Key {
			case "vcs.revision":
				if info.Commit == "" {
					info.Commit = shortCommit(setting.Value)
				}
			case "vcs.time":
				if info.CommitDate == "" {
					info.CommitDate = setting.Value
				}
			case "vcs.modified":
				if gitModified == "" {
					info.Modified = setting.Value == "true"
				}
			}
		}
	}
	info.Commit = shortCommit(info.Commit)
	return info
}

func splitRecentCommits(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	raw = strings.ReplaceAll(raw, "|", "\n")
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

func shortCommit(commit string) string {
	commit = strings.TrimSpace(commit)
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}

func shouldAutoCheck(args []string) bool {
	if updater.NoUpdateDisabled() {
		return false
	}
	if len(args) == 0 {
		return true
	}
	switch args[0] {
	case "version", "--version", "-v", "help", "-h", "--help", "config", "update":
		return false
	default:
		return true
	}
}

func autoUpdateCheck(ctx context.Context, lang i18n.Lang) bool {
	l := i18n.New(lang)
	client := updater.Client{CurrentVersion: version, AssetSuffix: assetSuffix}
	if !client.ShouldCheck() {
		return false
	}
	result, err := client.Check(ctx)
	client.MarkChecked()
	if err != nil {
		fmt.Fprintln(os.Stderr, l.S("update_check_failed", err))
		return false
	}
	if !result.UpdateAvailable {
		return false
	}
	if !isTerminal(os.Stdin) {
		fmt.Fprintln(os.Stderr, l.S("update_available_noninteractive", result.LatestVersion, version, result.GitHubAssetURL))
		return false
	}
	if !promptUpdate(bufio.NewReader(os.Stdin), os.Stderr, l, result) {
		return false
	}
	return installUpdate(ctx, client, result, l, os.Stderr)
}

func updateCmd(ctx context.Context, args []string, lang i18n.Lang) error {
	l := i18n.New(lang)
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	yes := fs.Bool("yes", false, l.S("flag_update_yes"))
	checkOnly := fs.Bool("check-only", false, l.S("flag_update_check_only"))
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%s", l.S("unexpected_args", fs.Args()))
	}
	client := updater.Client{CurrentVersion: version, AssetSuffix: assetSuffix}
	result, err := client.Check(ctx)
	client.MarkChecked()
	if err != nil {
		return err
	}
	if !result.UpdateAvailable {
		fmt.Println(l.S("update_none", version))
		return nil
	}
	if *checkOnly {
		fmt.Println(l.S("update_available", result.LatestVersion, version, result.GitHubAssetURL))
		return nil
	}
	if !*yes {
		if !isTerminal(os.Stdin) {
			fmt.Println(l.S("update_available_noninteractive", result.LatestVersion, version, result.GitHubAssetURL))
			return nil
		}
		if !promptUpdate(bufio.NewReader(os.Stdin), os.Stdout, l, result) {
			fmt.Println(l.S("update_declined"))
			return nil
		}
	}
	if !installUpdate(ctx, client, result, l, os.Stdout) {
		return fmt.Errorf("%s", l.S("update_install_failed"))
	}
	return nil
}

func promptUpdate(r *bufio.Reader, out io.Writer, l i18n.Localizer, result updater.CheckResult) bool {
	fmt.Fprintf(out, "%s [%s]: ", l.S("update_available_prompt", result.LatestVersion, result.CurrentVersion), l.BoolSuffix(false))
	answer, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return false
	}
	yes, parsed := i18n.ParseBoolInput(answer, false)
	return parsed && yes
}

func installUpdate(ctx context.Context, client updater.Client, result updater.CheckResult, l i18n.Localizer, out io.Writer) bool {
	fmt.Fprintln(out, l.S("update_downloading", result.LatestVersion))
	status, err := client.Install(ctx, result)
	if err != nil {
		fmt.Fprintln(out, l.S("update_install_error", err))
		return false
	}
	if status == "pending" {
		fmt.Fprintln(out, l.S("update_windows_pending", result.LatestVersion))
	} else {
		fmt.Fprintln(out, l.S("update_success_restart", result.LatestVersion))
	}
	return true
}

func isTerminal(f *os.File) bool {
	return f != nil && term.IsTerminal(int(f.Fd()))
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
	authJSON := fs.Bool("auth-json", false, l.S("flag_auth_json"))
	authPath := fs.String("auth-path", "", l.S("flag_auth_path"))
	apiFormat := fs.String("api-format", "", l.S("flag_api_format"))
	modelAPIBase := fs.String("model-api-base-url", os.Getenv("LD_GPT_CHECK_MODEL_API_BASE_URL"), l.S("flag_model_api_base"))
	modelAPIKey := fs.String("model-api-key", "", l.S("flag_model_api_key"))
	codexStartupArgs := fs.String("codex-args", "", l.S("flag_codex_args"))
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
	if *authJSON {
		*backend = string(runner.BackendAuthJSON)
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
	if *upload && strings.TrimSpace(*codexStartupArgs) != "" {
		fmt.Fprintln(progressOut, l.S("codex_args_upload_notice"))
	}
	normalizedBackend, _ := runner.NormalizeBackend(runner.Backend(*backend))
	if normalizedBackend == runner.BackendCodex || normalizedBackend == runner.BackendAuto {
		if resolution := system.DetectCCSwitchCodexResolution(); resolution.ProviderBaseURL != "" && system.CCSwitchAutoResolveEnabled() {
			fmt.Fprintf(progressOut, l.S("cc_switch_auto_using")+"\n", resolution.ProviderBaseURL)
		}
	}
	report.PrintQuestionPrompts(progressOut, lang, selected, report.ColorEnabled(progressOut))
	progress := report.PrintProgress(progressOut, lang, resolvedProgressModel, *effort, report.ColorEnabled(progressOut))
	summary, err := runner.Run(ctx, runner.Options{
		Model:            *model,
		ReasoningEffort:  *effort,
		Tests:            *tests,
		Timeout:          *timeout,
		Lang:             lang,
		Backend:          runner.Backend(*backend),
		APIFormat:        runner.APIFormat(*apiFormat),
		ModelAPIBaseURL:  *modelAPIBase,
		ModelAPIKey:      *modelAPIKey,
		CodexStartupArgs: *codexStartupArgs,
		AuthPath:         *authPath,
		QuestionSuite:    *suite,
		Questions:        selected,
		Progress:         progress,
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
		if provider, _ := resp["provider"].(map[string]any); provider != nil {
			if label, _ := provider["label"].(string); strings.TrimSpace(label) != "" {
				fmt.Fprintf(out, l.S("uploaded_provider")+"\n", label)
			}
		}
	}
	return nil
}

func doctorCmd(ctx context.Context, args []string, lang i18n.Lang) error {
	_ = ctx
	l := i18n.New(lang)
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	authJSON := fs.Bool("auth-json", false, l.S("flag_auth_json"))
	authPath := fs.String("auth-path", "", l.S("flag_auth_path"))
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%s", l.S("unexpected_args", fs.Args()))
	}
	if path, err := system.CodexPath(); err == nil {
		fmt.Println(l.S("doctor_codex_cli", path))
	} else {
		fmt.Println(l.S("doctor_codex_cli", l.S("doctor_missing")))
	}
	if *authJSON {
		printAuthJSONDoctor(l, *authPath)
	}
	return nil
}

func printAuthJSONDoctor(l i18n.Localizer, path string) {
	resolved := codexauth.ResolveAuthPath(path)
	fmt.Println(l.S("doctor_auth_json_title"))
	fmt.Println(l.S("doctor_auth_json_path", resolved))
	auth, err := codexauth.Load(resolved)
	if err != nil {
		fmt.Println(l.S("doctor_auth_json_file_missing", err))
		return
	}
	info := auth.Info()
	fmt.Println(l.S("doctor_auth_json_file_found"))
	fmt.Println(l.S("doctor_auth_json_auth_mode", valueOrDash(info.AuthMode)))
	fmt.Println(l.S("doctor_auth_json_token", presentLabel(l, info.HasAccessToken)))
	fmt.Println(l.S("doctor_auth_json_refresh", presentLabel(l, info.HasRefreshToken)))
	fmt.Println(l.S("doctor_auth_json_id", presentLabel(l, info.HasIDToken)))
	fmt.Println(l.S("doctor_auth_json_status", valueOrDash(info.AccessStatus)))
	if info.AccessExpiresAt != "" {
		fmt.Println(l.S("doctor_auth_json_exp", info.AccessExpiresAt))
	}
	if info.Email != "" {
		fmt.Println(l.S("doctor_auth_json_email", info.Email))
	}
	if info.PlanType != "" {
		fmt.Println(l.S("doctor_auth_json_plan", info.PlanType))
	}
}

func presentLabel(l i18n.Localizer, ok bool) string {
	if ok {
		return l.S("doctor_present")
	}
	return l.S("doctor_missing")
}

func valueOrDash(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return strings.TrimSpace(v)
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
