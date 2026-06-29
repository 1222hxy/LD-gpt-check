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

	"github.com/1222hxy/LD-gpt-check/internal/api"
	"github.com/1222hxy/LD-gpt-check/internal/auth"
	"github.com/1222hxy/LD-gpt-check/internal/codexauth"
	"github.com/1222hxy/LD-gpt-check/internal/config"
	"github.com/1222hxy/LD-gpt-check/internal/i18n"
	"github.com/1222hxy/LD-gpt-check/internal/questions"
	"github.com/1222hxy/LD-gpt-check/internal/report"
	"github.com/1222hxy/LD-gpt-check/internal/runner"
	"github.com/1222hxy/LD-gpt-check/internal/system"
	"golang.org/x/term"
)

type Options struct {
	Version string
	Lang    i18n.Lang
	Stdin   io.Reader
	Stdout  io.Writer
}

var runBenchmark = runner.Run
var loadRemoteQuestions = questions.LoadRemoteNoCache

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
	color := report.ColorEnabled(out)
	configPath, err := config.Path()
	if err != nil {
		return err
	}

	report.PrintBanner(out, l.S("wizard_title"), l.S("wizard_subtitle"), color)
	report.PrintSection(out, 1, l.S("wizard_step_config"), color)
	report.PrintInfo(out, l.S("wizard_config_file"), configPath, color)

	oldAPIBase := cfg.APIBaseURL
	cfg.Language = string(lang)
	apiBase := config.DefaultAPIBaseURL()
	cfg.APIBaseURL = apiBase
	if err := config.Save(cfg); err != nil {
		return err
	}
	report.PrintInfo(out, l.S("wizard_api_label"), apiBase, color)

	report.PrintSection(out, 2, l.S("wizard_step_login"), color)
	if cfg.AccessToken != "" {
		name := cfg.User.Username
		if name == "" {
			name = cfg.User.ID
		}
		report.PrintSuccess(out, l.S("wizard_existing_credential", fallback(name, l.S("wizard_unknown_user"))), color)
		reloginDefault := strings.TrimRight(oldAPIBase, "/") != strings.TrimRight(apiBase, "/")
		if reloginDefault {
			report.PrintWarning(out, l.S("wizard_api_changed"), color)
		}
		if yes, err := promptBool(reader, out, l, l.S("wizard_relogin"), reloginDefault); err != nil {
			return err
		} else if yes {
			user, err := auth.LoginWithOptions(ctx, auth.LoginOptions{APIBaseURL: apiBase, Lang: lang, Stdout: out})
			if err != nil {
				return err
			}
			report.PrintSuccess(out, l.S("login_success", user.Username, user.ID), color)
			report.PrintInfo(out, l.S("wizard_credentials_file"), configPath, color)
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
			report.PrintSuccess(out, l.S("login_success", user.Username, user.ID), color)
			report.PrintInfo(out, l.S("wizard_credentials_file"), configPath, color)
			cfg, _ = config.Load()
		} else {
			report.PrintWarning(out, l.S("wizard_skip_login"), color)
		}
	}

	report.PrintSection(out, 3, l.S("wizard_step_run"), color)
	if ok, err := promptBool(reader, out, l, l.S("wizard_run_now"), true); err != nil {
		return err
	} else if !ok {
		report.PrintSuccess(out, l.S("wizard_done_next"), color)
		return nil
	}
	selectedQuestion, err := promptQuestion(ctx, reader, out, l, color, apiBase)
	if err != nil {
		return err
	}

	backend := runner.BackendCodex
	apiFormat := runner.APIFormat("")
	modelAPIBase := ""
	modelAPIKey := ""
	credentialSource := ""
	configSource := ""
	authPath := ""
	codexStartupArgs := ""
	codexPath, codexErr := system.CodexPath()
	if codexErr == nil {
		report.PrintSuccess(out, l.S("wizard_codex_found", codexPath), color)
	} else {
		report.PrintWarning(out, l.S("wizard_codex_missing"), color)
	}
	authJSONAvailable := false
	if auth, err := codexauth.Load(""); err == nil {
		status := codexauth.AccessTokenStatus(auth)
		if authJSONUsableInWizard(auth, status) {
			authJSONAvailable = true
			authPath = auth.AuthPath
			report.PrintSuccess(out, l.S("wizard_auth_json_found", auth.AuthPath, status), color)
		}
	}
	ccSwitchResolution := system.DetectCCSwitchCodexResolution()
	if ccSwitchResolution.ProviderBaseURL != "" {
		report.PrintSuccess(out, l.S("wizard_cc_switch_detected", ccSwitchResolution.ProviderBaseURL), color)
	}
	printCodexConfigLocation(out, l, color, ccSwitchResolution.ProviderBaseURL != "")
	if codexErr == nil {
		backend = runner.BackendCodex
		report.PrintSuccess(out, l.S("wizard_backend_codex_auto"), color)
	} else if ccSwitchResolution.ProviderBaseURL != "" {
		if yes, err := promptBool(reader, out, l, l.S("wizard_cc_switch_api_use"), true); err != nil {
			return err
		} else if yes {
			backend = runner.BackendAPI
			modelAPIBase = ccSwitchResolution.ProviderBaseURL
			modelAPIKey = ccSwitchResolution.ModelAPIKey
			configSource = l.S("wizard_config_source_cc_switch")
			if modelAPIKey != "" {
				credentialSource = l.S("wizard_credential_cc_switch_key")
			}
			if format, ok := runner.NormalizeAPIFormat(runner.APIFormat(ccSwitchResolution.APIFormat)); ok {
				apiFormat = format
			}
			report.PrintSuccess(out, l.S("wizard_cc_switch_using", ccSwitchResolution.ProviderBaseURL), color)
		} else {
			backend, err = promptBackend(reader, out, l, color, false, authJSONAvailable)
			if err != nil {
				return err
			}
		}
	} else {
		backend, err = promptBackend(reader, out, l, color, false, authJSONAvailable)
		if err != nil {
			return err
		}
	}
	if backend == runner.BackendCodex && codexErr != nil {
		report.PrintWarning(out, l.S("wizard_need_api_without_codex"), color)
		report.PrintSuccess(out, l.S("wizard_done_next"), color)
		return nil
	}
	if backend == runner.BackendAPI {
		report.PrintSuccess(out, l.S("wizard_ready_api_run"), color)
		if apiFormat == "" {
			apiFormat, err = promptAPIFormat(reader, out, l, color)
			if err != nil {
				return err
			}
		} else {
			report.PrintInfo(out, l.S("wizard_api_format"), wizardAPIFormatLabel(l, apiFormat), color)
		}
		modelAPIBase, err = promptString(reader, out, l, l.S("wizard_model_api_base"), firstNonEmpty(modelAPIBase, defaultAPIBaseURL(apiFormat)))
		if err != nil {
			return err
		}
		modelAPIKey = firstNonEmpty(modelAPIKey, os.Getenv("LD_GPT_CHECK_MODEL_API_KEY"))
		if modelAPIKey == "" {
			report.PrintWarning(out, l.S("wizard_api_key_warning"), color)
			modelAPIKey, err = promptMaskedString(reader, out, l, l.S("wizard_model_api_key"), in)
			if err != nil {
				return err
			}
			credentialSource = l.S("wizard_credential_manual_key")
		} else if credentialSource == "" {
			credentialSource = l.S("wizard_credential_env_key")
		} else if ccSwitchResolution.ModelAPIKey != "" && modelAPIKey == ccSwitchResolution.ModelAPIKey {
			report.PrintSuccess(out, l.S("wizard_cc_switch_key_using"), color)
		}
		printExtractedRunConfig(out, l, color, backend, configSource, modelAPIBase, apiFormat, credentialSource, authPath)
	} else if backend == runner.BackendAuthJSON {
		configSource = l.S("wizard_config_source_auth_json")
		credentialSource = l.S("wizard_credential_auth_json")
		report.PrintSuccess(out, l.S("wizard_ready_auth_json_run"), color)
		printExtractedRunConfig(out, l, color, backend, configSource, modelAPIBase, apiFormat, credentialSource, authPath)
	} else {
		applyCCSwitchResolutionPrompt(reader, out, l, color, ccSwitchResolution)
		report.PrintSuccess(out, l.S("wizard_ready_run"), color)
		printExtractedRunConfig(out, l, color, backend, l.S("wizard_config_source_codex_cli"), modelAPIBase, apiFormat, l.S("wizard_credential_codex_cli"), authPath)
		report.PrintWarning(out, l.S("codex_args_upload_notice"), color)
		codexStartupArgs, err = promptOptionalString(reader, out, l.S("wizard_codex_startup_args"), l.S("wizard_codex_startup_args_empty"))
		if err != nil {
			return err
		}
	}
	model, err := promptRunModel(reader, out, l, color, backend)
	if err != nil {
		return err
	}
	effort, err := promptEffort(reader, out, l, l.S("wizard_effort"), "medium")
	if err != nil {
		return err
	}
	tests, err := promptInt(reader, out, l, l.S("wizard_tests"), runner.DefaultTests, 1, runner.MaxTests)
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
	anonymous := false
	if upload {
		report.PrintWarning(out, l.S("wizard_anonymous_note"), color)
		anonymous, err = promptBool(reader, out, l, l.S("wizard_anonymous_upload"), false)
		if err != nil {
			return err
		}
	}

	report.PrintQuestionPrompts(out, lang, []questions.Question{selectedQuestion}, color)
	summary, err := runBenchmark(ctx, runner.Options{
		Model:            model,
		ReasoningEffort:  effort,
		Tests:            tests,
		Timeout:          timeout,
		Lang:             lang,
		Backend:          backend,
		APIFormat:        apiFormat,
		ModelAPIBaseURL:  modelAPIBase,
		ModelAPIKey:      modelAPIKey,
		AuthPath:         authPath,
		CodexStartupArgs: codexStartupArgs,
		QuestionSuite:    selectedQuestion.ID,
		Questions:        []questions.Question{selectedQuestion},
		Progress:         report.PrintProgress(out, lang, progressModel(model), effort, color),
	})
	if err != nil {
		return err
	}
	report.PrintTableWithWriter(out, summary, lang, color)

	report.PrintSection(out, 4, l.S("wizard_step_upload"), color)
	uploadStatus := l.S("wizard_upload_skipped")
	uploadStatusOK := false
	if upload {
		codexVersion := system.UploadCodexVersion(summary.CodexSandbox)
		payload := api.PayloadFromSummary(opts.Version, summary, runtime.GOOS, runtime.GOARCH, codexVersion)
		payload.Anonymous = anonymous
		resp, err := api.NewWithLang(cfg.APIBaseURL, cfg.AccessToken, lang).UploadRun(ctx, payload)
		if err != nil {
			return err
		}
		if id, _ := resp["id"].(string); id != "" {
			uploadStatus = l.S("uploaded_run", id)
			uploadStatusOK = true
			report.PrintSuccess(out, uploadStatus, color)
		} else {
			uploadStatus = l.S("uploaded_run_no_id")
			uploadStatusOK = true
			report.PrintSuccess(out, uploadStatus, color)
		}
	} else {
		report.PrintWarning(out, l.S("wizard_upload_skipped"), color)
	}

	report.PrintWizardRunRecord(out, report.WizardRunRecord{
		Backend:          backend,
		APIFormat:        apiFormat,
		Model:            model,
		ModelAPIBaseURL:  modelAPIBase,
		CodexStartupArgs: codexStartupArgs,
		ReasoningEffort:  effort,
		Tests:            tests,
		Timeout:          timeout,
		Upload:           upload,
		Anonymous:        anonymous,
		UploadStatus:     uploadStatus,
		UploadStatusOK:   uploadStatusOK,
		Question:         selectedQuestion,
		QuestionSource:   wizardQuestionSource(selectedQuestion),
		Summary:          summary,
	}, lang, color)
	report.PrintSuccess(out, l.S("wizard_done"), color)
	return nil
}

func printExtractedRunConfig(out io.Writer, l i18n.Localizer, color bool, backend runner.Backend, source, baseURL string, format runner.APIFormat, credentialSource, authPath string) {
	report.PrintInfo(out, l.S("wizard_extracted_config"), firstNonEmpty(source, wizardBackendSummary(l, backend)), color)
	report.PrintInfo(out, l.S("wizard_extracted_backend"), wizardBackendSummary(l, backend), color)
	if strings.TrimSpace(baseURL) != "" {
		report.PrintInfo(out, l.S("wizard_extracted_base_url"), baseURL, color)
	}
	if backend == runner.BackendAPI && format != "" {
		report.PrintInfo(out, l.S("wizard_extracted_api_format"), wizardAPIFormatLabel(l, format), color)
	}
	if backend == runner.BackendAuthJSON && strings.TrimSpace(authPath) != "" {
		report.PrintInfo(out, l.S("wizard_extracted_auth_json"), authPath, color)
	}
	if strings.TrimSpace(credentialSource) != "" {
		report.PrintInfo(out, l.S("wizard_extracted_credential"), credentialSource, color)
	}
}

func printCodexConfigLocation(out io.Writer, l i18n.Localizer, color bool, ccSwitchDetected bool) {
	info, err := system.CodexConfigInfo()
	if err != nil {
		report.PrintWarning(out, l.S("wizard_codex_config_read_failed", err), color)
		return
	}
	if strings.TrimSpace(info.ConfigPath) != "" {
		report.PrintInfo(out, l.S("wizard_codex_config_path"), info.ConfigPath, color)
	}
	if ccSwitchDetected {
		return
	}
	if strings.TrimSpace(info.ProviderBaseURL) != "" {
		label := info.ProviderBaseURL
		if strings.TrimSpace(info.ModelProvider) != "" {
			label = info.ModelProvider + " -> " + label
		}
		report.PrintInfo(out, l.S("wizard_codex_config_provider"), label, color)
	}
	report.PrintWarning(out, l.S("wizard_codex_config_notice"), color)
}

func wizardBackendSummary(l i18n.Localizer, backend runner.Backend) string {
	switch backend {
	case runner.BackendAPI:
		return l.S("wizard_record_backend_api")
	case runner.BackendAuthJSON:
		return l.S("wizard_record_backend_auth_json")
	default:
		return l.S("wizard_record_backend_codex")
	}
}

func applyCCSwitchResolutionPrompt(r *bufio.Reader, out io.Writer, l i18n.Localizer, color bool, resolution system.CCSwitchResolution) {
	if resolution.ProviderBaseURL == "" {
		return
	}
	if strings.TrimSpace(resolution.LocalBaseURL) != "" {
		report.PrintWarning(out, l.S("wizard_cc_switch_note", resolution.LocalBaseURL, resolution.ConfigDir), color)
	} else {
		report.PrintWarning(out, l.S("wizard_cc_switch_installed_note", resolution.ConfigDir), color)
	}
	yes, err := promptBool(r, out, l, l.S("wizard_cc_switch_use"), true)
	if err != nil {
		return
	}
	if yes {
		_ = os.Unsetenv("LD_GPT_CHECK_DISABLE_CC_SWITCH")
		report.PrintSuccess(out, l.S("wizard_cc_switch_using", resolution.ProviderBaseURL), color)
		return
	}
	_ = os.Setenv("LD_GPT_CHECK_DISABLE_CC_SWITCH", "1")
	report.PrintWarning(out, l.S("wizard_cc_switch_skipped"), color)
}

func wizardQuestionSource(q questions.Question) string {
	if q.ID == questions.DefaultSuite {
		return "classic"
	}
	return "remote"
}

func promptQuestion(ctx context.Context, r *bufio.Reader, out io.Writer, l i18n.Localizer, color bool, apiBase string) (questions.Question, error) {
	choices := append([]questions.Question(nil), questions.Builtin()...)
	seen := make(map[string]bool, len(choices))
	for _, q := range choices {
		seen[q.ID] = true
	}
	remoteURL := strings.TrimRight(apiBase, "/") + "/api/v1/questions"
	remote, err := loadRemoteQuestions(ctx, remoteURL, true)
	if err != nil {
		report.PrintWarning(out, l.S("wizard_questions_remote_failed", err), color)
	} else {
		for _, q := range remote {
			if seen[q.ID] {
				continue
			}
			seen[q.ID] = true
			choices = append(choices, q)
		}
	}

	for i, q := range choices {
		if i == 0 {
			fmt.Fprintln(out, report.Muted(l.S("wizard_question_classic", i+1, q.Title, q.ID), color))
			continue
		}
		fmt.Fprintln(out, report.Muted(l.S("wizard_question_remote", i+1, q.Title, q.ID), color))
	}
	for {
		choice, err := promptString(r, out, l, l.S("wizard_question"), "1")
		if err != nil {
			return questions.Question{}, err
		}
		idx, err := strconv.Atoi(choice)
		if err == nil && idx >= 1 && idx <= len(choices) {
			return choices[idx-1], nil
		}
		fmt.Fprintln(out, l.S("wizard_question_invalid", len(choices)))
	}
}

func promptModel(r *bufio.Reader, out io.Writer, l i18n.Localizer, color bool) (string, error) {
	configured, err := system.CodexConfiguredModel()
	if err == nil && system.ConcreteCodexModel(configured) {
		report.PrintSuccess(out, l.S("model_detected", configured), color)
		return promptOptionalString(r, out, l.S("wizard_model"), configured)
	}
	if err != nil {
		report.PrintWarning(out, err.Error(), color)
	}
	report.PrintWarning(out, l.S("model_choose"), color)
	fmt.Fprintln(out, report.Muted(l.S("model_choice_55"), color))
	fmt.Fprintln(out, report.Muted(l.S("model_choice_54"), color))
	fmt.Fprintln(out, report.Muted(l.S("model_choice_other"), color))
	for {
		choice, err := promptString(r, out, l, l.S("wizard_model"), "1")
		if err != nil {
			return "", err
		}
		switch strings.ToLower(strings.TrimSpace(choice)) {
		case "1", "gpt-5.5", "gpt 5.5":
			return "gpt-5.5", nil
		case "2", "gpt-5.4", "gpt 5.4":
			return "gpt-5.4", nil
		case "3", "other", "custom", "其他", "自定义":
			return promptString(r, out, l, l.S("model_custom"), "gpt-5.5")
		default:
			if system.ConcreteCodexModel(choice) {
				return strings.TrimSpace(choice), nil
			}
			fmt.Fprintln(out, l.S("prompt_non_empty"))
		}
	}
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

func authJSONUsableInWizard(auth *codexauth.CodexAuth, status string) bool {
	if auth == nil {
		return false
	}
	if status != "missing" && status != "expired" {
		return true
	}
	return strings.TrimSpace(auth.RefreshToken) != ""
}

func promptBackend(r *bufio.Reader, out io.Writer, l i18n.Localizer, color, codexAvailable, authJSONAvailable bool) (runner.Backend, error) {
	if codexAvailable {
		fmt.Fprintln(out, report.Muted(l.S("wizard_backend_codex"), color))
		nextAPIChoice := "2"
		if authJSONAvailable {
			fmt.Fprintln(out, report.Muted(l.S("wizard_backend_auth_json", 2), color))
			nextAPIChoice = "3"
		}
		fmt.Fprintln(out, report.Muted(l.S("wizard_backend_api_numbered", nextAPIChoice), color))
		for {
			choice, err := promptString(r, out, l, l.S("wizard_backend"), "1")
			if err != nil {
				return "", err
			}
			switch strings.ToLower(strings.TrimSpace(choice)) {
			case "1", "codex", "local", "cli":
				return runner.BackendCodex, nil
			case "2", "auth-json", "auth_json", "authjson":
				if authJSONAvailable {
					return runner.BackendAuthJSON, nil
				}
				return runner.BackendAPI, nil
			case "3", "api", "http":
				return runner.BackendAPI, nil
			default:
				if backend, ok := runner.NormalizeBackend(runner.Backend(choice)); ok && backend != runner.BackendAuto {
					if backend == runner.BackendAuthJSON && !authJSONAvailable {
						fmt.Fprintln(out, l.S("wizard_backend_invalid"))
						continue
					}
					return backend, nil
				}
				fmt.Fprintln(out, l.S("wizard_backend_invalid"))
			}
		}
	}

	if authJSONAvailable {
		fmt.Fprintln(out, report.Muted(l.S("wizard_backend_auth_json", 1), color))
		fmt.Fprintln(out, report.Muted(l.S("wizard_backend_api_numbered", "2"), color))
		fmt.Fprintln(out, report.Muted(l.S("wizard_backend_exit_numbered", "3"), color))
	} else {
		fmt.Fprintln(out, report.Muted(l.S("wizard_backend_api_primary"), color))
		fmt.Fprintln(out, report.Muted(l.S("wizard_backend_exit"), color))
	}
	for {
		choice, err := promptString(r, out, l, l.S("wizard_backend"), "1")
		if err != nil {
			return "", err
		}
		switch strings.ToLower(strings.TrimSpace(choice)) {
		case "1", "auth-json", "auth_json", "authjson":
			if authJSONAvailable {
				return runner.BackendAuthJSON, nil
			}
			return runner.BackendAPI, nil
		case "2", "api", "http":
			if authJSONAvailable {
				return runner.BackendAPI, nil
			}
			return runner.BackendCodex, nil
		case "3", "exit", "quit", "skip", "取消", "退出":
			return runner.BackendCodex, nil
		default:
			fmt.Fprintln(out, l.S("wizard_backend_invalid"))
		}
	}
}

func promptRunModel(r *bufio.Reader, out io.Writer, l i18n.Localizer, color bool, backend runner.Backend) (string, error) {
	if backend == runner.BackendAPI || backend == runner.BackendAuthJSON {
		return promptAPIModel(r, out, l, color)
	}
	return promptModel(r, out, l, color)
}

func promptAPIFormat(r *bufio.Reader, out io.Writer, l i18n.Localizer, color bool) (runner.APIFormat, error) {
	fmt.Fprintln(out, report.Muted(l.S("wizard_api_format_chat"), color))
	fmt.Fprintln(out, report.Muted(l.S("wizard_api_format_responses"), color))
	fmt.Fprintln(out, report.Muted(l.S("wizard_api_format_anthropic"), color))
	for {
		choice, err := promptString(r, out, l, l.S("wizard_api_format"), "1")
		if err != nil {
			return "", err
		}
		switch strings.ToLower(strings.TrimSpace(choice)) {
		case "1", "openai-chat", "chat", "completion", "completions", "chat-completions":
			return runner.APIFormatOpenAIChat, nil
		case "2", "openai-responses", "responses", "response":
			return runner.APIFormatOpenAIResponses, nil
		case "3", "anthropic", "anthropic-messages", "messages":
			return runner.APIFormatAnthropic, nil
		default:
			if format, ok := runner.NormalizeAPIFormat(runner.APIFormat(choice)); ok {
				return format, nil
			}
			fmt.Fprintln(out, l.S("runner_api_format_invalid", choice))
		}
	}
}

func promptAPIModel(r *bufio.Reader, out io.Writer, l i18n.Localizer, color bool) (string, error) {
	report.PrintInfo(out, l.S("wizard_api_model_hint"), l.S("wizard_api_model_hint_value"), color)
	fmt.Fprintln(out, report.Muted(l.S("model_choice_55"), color))
	fmt.Fprintln(out, report.Muted(l.S("model_choice_54"), color))
	fmt.Fprintln(out, report.Muted(l.S("model_choice_other"), color))
	for {
		choice, err := promptString(r, out, l, l.S("wizard_model"), "2")
		if err != nil {
			return "", err
		}
		switch strings.ToLower(strings.TrimSpace(choice)) {
		case "1", "gpt-5.5", "gpt 5.5":
			return "gpt-5.5", nil
		case "2", "gpt-5.4", "gpt 5.4":
			return "gpt-5.4", nil
		case "3", "other", "custom", "其他", "自定义":
			return promptString(r, out, l, l.S("model_custom"), "gpt-5.4")
		default:
			if system.ConcreteCodexModel(choice) {
				return strings.TrimSpace(choice), nil
			}
			fmt.Fprintln(out, l.S("prompt_non_empty"))
		}
	}
}

func defaultAPIBaseURL(format runner.APIFormat) string {
	switch format {
	case runner.APIFormatAnthropic:
		return "https://api.anthropic.com/v1"
	default:
		return "https://api.openai.com/v1"
	}
}

func wizardAPIFormatLabel(l i18n.Localizer, format runner.APIFormat) string {
	switch format {
	case runner.APIFormatOpenAIResponses:
		return l.S("wizard_record_api_format_responses")
	case runner.APIFormatAnthropic:
		return l.S("wizard_record_api_format_anthropic")
	default:
		return l.S("wizard_record_api_format_chat")
	}
}

func promptMaskedString(r *bufio.Reader, out io.Writer, l i18n.Localizer, label string, in io.Reader) (string, error) {
	for {
		fmt.Fprintf(out, "%s: ", label)
		var s string
		var err error
		if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
			s, err = readMaskedTerminal(f, out)
		} else {
			s, err = readLine(r)
		}
		if err != nil {
			return "", err
		}
		if s = strings.TrimSpace(s); s != "" {
			return s, nil
		}
		fmt.Fprintln(out, l.S("prompt_non_empty"))
	}
}

func readMaskedTerminal(in *os.File, out io.Writer) (string, error) {
	fd := int(in.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = term.Restore(fd, oldState)
	}()

	var b []byte
	buf := make([]byte, 1)
	for {
		n, err := in.Read(buf)
		if err != nil {
			fmt.Fprintln(out)
			return "", err
		}
		if n == 0 {
			continue
		}
		switch c := buf[0]; c {
		case '\r', '\n':
			fmt.Fprintln(out)
			return string(b), nil
		case 3:
			fmt.Fprintln(out)
			return "", fmt.Errorf("interrupted")
		case 4:
			if len(b) == 0 {
				fmt.Fprintln(out)
				return "", io.EOF
			}
		case 8, 127:
			if len(b) > 0 {
				b = b[:len(b)-1]
				fmt.Fprint(out, "\b \b")
			}
		case 21:
			for len(b) > 0 {
				b = b[:len(b)-1]
				fmt.Fprint(out, "\b \b")
			}
		default:
			if c >= 32 && c != 127 {
				b = append(b, c)
				fmt.Fprint(out, "*")
			}
		}
	}
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

func promptOptionalString(r *bufio.Reader, out io.Writer, label, placeholder string) (string, error) {
	fmt.Fprintf(out, "%s [%s]: ", label, placeholder)
	s, err := readLine(r)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(s), nil
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
