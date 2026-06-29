package i18n

import (
	"fmt"
	"os"
	"strings"
)

type Lang string

const (
	ZH Lang = "zh-CN"
	EN Lang = "en"
)

func Normalize(v string) Lang {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "en", "en-us", "en_us":
		return EN
	case "zh", "zh-cn", "zh_cn", "cn", "chinese", "中文", "":
		return ZH
	default:
		return ZH
	}
}

func Detect(configLang string) Lang {
	if v := os.Getenv("LD_GPT_CHECK_LANG"); strings.TrimSpace(v) != "" {
		return Normalize(v)
	}
	return Normalize(configLang)
}

type Localizer struct {
	Lang Lang
}

func New(lang Lang) Localizer {
	return Localizer{Lang: Normalize(string(lang))}
}

func (l Localizer) S(key string, args ...any) string {
	msgs := zhMessages
	if l.Lang == EN {
		msgs = enMessages
	}
	format, ok := msgs[key]
	if !ok {
		format, ok = zhMessages[key]
	}
	if !ok {
		format = key
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

func (l Localizer) BoolSuffix(def bool) string {
	if l.Lang == EN {
		if def {
			return "Y/n"
		}
		return "y/N"
	}
	if def {
		return "是/否，默认是"
	}
	return "是/否，默认否"
}

func ParseBoolInput(input string, def bool) (bool, bool) {
	s := strings.ToLower(strings.TrimSpace(input))
	if s == "" {
		return def, true
	}
	switch s {
	case "y", "yes", "true", "1", "是", "好", "确认", "可以":
		return true, true
	case "n", "no", "false", "0", "否", "不", "取消":
		return false, true
	default:
		return false, false
	}
}

var zhMessages = map[string]string{
	"error_prefix":                          "错误：%v",
	"unknown_command":                       "未知命令 %q",
	"unexpected_args":                       "存在无法识别的参数：%v",
	"flag_model":                            "模型名称；不填则使用 Codex 本机配置中的模型",
	"flag_effort":                           "推理强度：low、medium、high、xhigh",
	"flag_backend":                          "测试后端：auto、codex、api",
	"flag_api_format":                       "API 调用格式：openai-chat、openai-responses、anthropic-messages",
	"flag_model_api_base":                   "模型 API Base URL，例如 https://api.example.com/v1",
	"flag_model_api_key":                    "模型 API Key；推荐改用环境变量 LD_GPT_CHECK_MODEL_API_KEY",
	"flag_codex_args":                       "额外 Codex 启动参数字符串；上传时会写入结果摘要",
	"flag_tests":                            "测试次数，默认 5",
	"flag_timeout":                          "每轮超时时间，例如 10m 或 30s",
	"flag_upload":                           "上传结果",
	"flag_anonymous":                        "匿名展示上传结果，公开页面隐藏 Linux.do 身份信息",
	"flag_json":                             "输出 JSON",
	"flag_api_base":                         "Worker API 地址",
	"flag_lang":                             "界面语言：zh-CN 或 en",
	"flag_update_yes":                       "发现新版本时不询问，直接更新",
	"flag_update_check_only":                "只检查是否有新版本，不下载或替换",
	"timeout_positive":                      "timeout 必须是正数",
	"canceled":                              "操作已取消",
	"flag_count_positive":                   "%s 必须是正整数",
	"uploaded_run":                          "已上传运行结果：%s",
	"uploaded_run_no_id":                    "已上传运行结果。",
	"uploaded_provider":                     "渠道 / 中转站：%s",
	"codex_args_upload_notice":              "为了确保公平，自定义 Codex 启动参数会随上传结果一同提交；不要在其中放入密钥或私密信息。",
	"login_success":                         "已登录：%s (%s)",
	"credential_saved":                      "凭证已保存到：%s",
	"config_path":                           "配置文件：%s",
	"config_api_base":                       "API 地址：%s",
	"config_language":                       "语言：%s",
	"config_not_logged_in":                  "登录状态：未登录",
	"config_logged_in":                      "登录状态：已登录",
	"model_local_config":                    "Codex 本机配置",
	"model_detected":                        "检测到 Codex 模型：%s",
	"model_choose":                          "无法确认 Codex 当前模型，请选择模型",
	"model_choice_55":                       "1) GPT 5.5",
	"model_choice_54":                       "2) GPT 5.4",
	"model_choice_other":                    "3) 其他模型",
	"model_custom":                          "请输入模型名称",
	"logout_success":                        "已退出登录。",
	"credential_removed":                    "本地凭证已从配置文件中移除：%s",
	"usage":                                 "LD-gpt-check\n\n用法：\n  ld-gpt-check setup [--lang zh-CN|en]\n  ld-gpt-check login [--api-base-url URL] [--lang zh-CN|en]\n  ld-gpt-check run [--backend auto|codex|api] [-m MODEL] [--api-format openai-chat|openai-responses|anthropic-messages] [--model-api-base-url URL] [--model-api-key KEY] [--codex-args ARGS] [--suite candy_21] [--question-file FILE] [--question-url URL] [--no-remote-questions] [-r low|medium|high|xhigh] [-n TESTS] [--timeout 30m] [--upload] [--json]\n  ld-gpt-check run --list-suites [--question-file FILE] [--question-url URL] [--no-remote-questions]\n  ld-gpt-check update [--yes|--check-only]\n  ld-gpt-check config\n  ld-gpt-check whoami\n  ld-gpt-check logout",
	"update_check_failed":                   "检查更新失败：%v",
	"update_available_noninteractive":       "发现新版本 %s（当前 %s）。可运行 `ld-gpt-check update --yes` 自动更新，或下载：%s",
	"update_available_prompt":               "发现新版本 %s（当前 %s），是否现在更新",
	"update_available":                      "发现新版本 %s（当前 %s）：%s",
	"update_none":                           "已是最新版本：%s",
	"update_declined":                       "已取消更新。",
	"update_downloading":                    "正在下载并校验版本 %s……",
	"update_install_error":                  "更新失败：%v",
	"update_install_failed":                 "更新失败",
	"update_success_restart":                "已更新到 %s。请重新运行命令。",
	"update_windows_pending":                "已下载版本 %s，将在当前进程退出后替换程序。请稍后重新运行命令。",
	"auth_missing_device_code":              "设备登录响应缺少 device_code",
	"auth_missing_user_code":                "设备登录响应缺少 user_code",
	"auth_missing_verification_uri":         "设备登录响应缺少验证链接",
	"auth_opening_browser":                  "正在打开浏览器完成 Linux.do 登录……",
	"auth_manual_open":                      "如果浏览器没有自动打开，请访问：",
	"auth_user_code":                        "验证码：",
	"auth_expired":                          "设备登录已过期，请重新运行 login",
	"auth_missing_access_token":             "设备授权成功但响应缺少 access_token",
	"auth_missing_user_id":                  "设备授权成功但响应缺少用户 ID",
	"auth_unexpected_status":                "未知登录状态：%s",
	"wizard_title":                          "LD-gpt-check 向导",
	"wizard_subtitle":                       "本向导会配置登录、运行测试，并可选择上传结果。",
	"wizard_step_config":                    "检查配置",
	"wizard_step_login":                     "登录状态",
	"wizard_step_run":                       "运行测试",
	"wizard_step_upload":                    "上传结果",
	"wizard_config_file":                    "配置文件",
	"wizard_credentials_file":               "凭证文件",
	"wizard_api_label":                      "API",
	"wizard_api_using":                      "正在使用 API：%s",
	"wizard_api_base":                       "Worker API 地址",
	"wizard_language":                       "界面语言 (zh-CN/en)",
	"wizard_existing_credential":            "当前已有登录凭证：%s",
	"wizard_unknown_user":                   "未知用户",
	"wizard_api_changed":                    "API 地址已变化，建议重新登录以获取对应后端的凭证。",
	"wizard_relogin":                        "是否重新登录",
	"wizard_login_now":                      "是否现在登录 Linux.do（上传结果需要登录）",
	"wizard_run_now":                        "是否现在运行一次测试",
	"wizard_done_next":                      "向导完成。之后可运行：ld-gpt-check run -r xhigh -n 5",
	"wizard_backend":                        "测试方式",
	"wizard_backend_codex":                  "1) 本机 Codex CLI",
	"wizard_backend_api":                    "2) API 模式（无需本机 Codex）",
	"wizard_backend_api_primary":            "1) API 模式（无需本机 Codex）",
	"wizard_backend_exit":                   "2) 暂不测试，退出向导",
	"wizard_backend_invalid":                "请输入有效测试方式：1 或 2。",
	"wizard_model":                          "模型",
	"wizard_model_default":                  "Codex 本机配置",
	"wizard_effort":                         "推理强度",
	"wizard_tests":                          "测试次数",
	"wizard_timeout":                        "每轮超时",
	"wizard_upload":                         "跑完后上传结果",
	"wizard_anonymous_upload":               "匿名展示上传结果",
	"wizard_anonymous_note":                 "匿名展示会隐藏公开页面中的 Linux.do 用户名、头像和主页链接；测试摘要仍会提交并参与统计。",
	"wizard_upload_needs_login":             "上传需要先登录；请重新运行向导或执行 ld-gpt-check login",
	"wizard_done":                           "向导完成。",
	"wizard_ready_run":                      "准备开始本地 Codex 测试。",
	"wizard_codex_found":                    "检测到本机 Codex：%s",
	"wizard_codex_missing":                  "未检测到本机 Codex；可以改用 API 模式测试。",
	"wizard_use_api":                        "是否使用 API 模式测试",
	"wizard_need_api_without_codex":         "没有 Codex 环境时需要使用 API 模式才能继续测试。",
	"wizard_ready_api_run":                  "准备开始 API 测试。",
	"wizard_api_format":                     "API 调用格式",
	"wizard_api_format_chat":                "1) OpenAI Chat Completions",
	"wizard_api_format_responses":           "2) OpenAI Responses",
	"wizard_api_format_anthropic":           "3) Anthropic Messages",
	"wizard_api_model_hint":                 "API 模型",
	"wizard_api_model_hint_value":           "请填写你的 API 服务实际支持的模型名称；可选预置 GPT-5.5 / GPT-5.4，或输入自定义模型。",
	"wizard_model_api_base":                 "模型 API Base URL",
	"wizard_model_api_key":                  "模型 API Key",
	"wizard_api_key_warning":                "建议创建一个新的临时 API Key，本次测试用完后立即销毁，避免争议和泄露风险。",
	"wizard_codex_startup_args":             "Codex 额外启动参数",
	"wizard_codex_startup_args_empty":       "留空",
	"wizard_skip_login":                     "已跳过登录；仍可本地运行测试。",
	"wizard_upload_skipped":                 "未上传，本次结果仅保存在终端输出中。",
	"wizard_questions_remote_failed":        "远程题目拉取失败，将使用经典题：%v",
	"wizard_question_classic":               "%d) 经典题：%s（%s）",
	"wizard_question_remote":                "%d) 远程题：%s（%s）",
	"wizard_question":                       "选择测试题目",
	"wizard_question_invalid":               "请输入 1 到 %d 之间的题目序号。",
	"prompt_non_empty":                      "请输入有效内容。",
	"prompt_bool":                           "请输入 yes/no 或 是/否。",
	"prompt_effort":                         "可选值：low, medium, high, xhigh。",
	"prompt_int_range":                      "请输入 %d 到 %d 之间的整数。",
	"prompt_duration":                       "请输入有效时长，例如 30s、10m、1h。",
	"api_not_logged_in":                     "尚未登录；请先运行 ld-gpt-check login",
	"api_base_empty":                        "API 地址为空；请设置 LD_GPT_CHECK_API_BASE_URL 或运行 login --api-base-url URL",
	"api_base_invalid":                      "无效的 API 地址 %q",
	"api_base_bad_scheme":                   "无效的 API 地址协议 %q",
	"api_http_nil":                          "HTTP client 为空",
	"api_request_failed":                    "%s %s 请求失败：%v",
	"api_status_failed":                     "请求失败：HTTP %d：%s",
	"api_decode_failed":                     "解析 %s 响应失败：%v",
	"api_empty_device_code":                 "device_code 不能为空",
	"api_upload_id_required":                "上传失败：upload_id 不能为空",
	"api_upload_model_required":             "上传失败：无法确认模型名称；请使用 -m 指定，例如 -m gpt-5.5",
	"api_upload_provider_base_url_required": "上传失败：无法确认 Codex provider base URL；请升级或检查 Codex 配置",
	"api_upload_tests_invalid":              "上传失败：attempt_count 必须大于 0",
	"api_upload_cases_mismatch":             "上传失败：attempts 数量必须等于 attempt_count",
	"api_upload_questions_mismatch":         "上传失败：questions 数量必须等于 question_count",
	"run_status_start":                      "开始运行：model=%s，reasoning=%s，共 %d 轮",
	"run_status_case_start":                 "▶ [%d/%d] 正在运行 %s #%d",
	"run_status_case_done":                  "• [%d/%d] %s，耗时 %.1fs，TPS %.1f",
	"run_status_case_error":                 "✗ [%d/%d] 运行失败：%v",
	"run_status_question":                   "测试题目：%s",
	"run_status_failed_question":            "失败题目：%s",
	"report_panel_title":                    "运行概览",
	"report_metric_accuracy":                "正确率",
	"report_metric_correct":                 "正确",
	"report_metric_time":                    "耗时",
	"report_metric_tps":                     "TPS",
	"report_metric_input":                   "输入",
	"report_metric_reason":                  "推理",
	"report_summary":                        "\n正确：%d/%d (%.1f%%)，平均耗时 %.1fs，平均 TPS %.1f\n",
	"runner_model_required":                 "缺少模型名称；请使用 -m 或 --model 指定",
	"runner_backend_invalid":                "无效的测试后端 %q；可选值：auto、codex、api",
	"runner_api_format_invalid":             "无效的 API 调用格式 %q；可选值：openai-chat、openai-responses、anthropic-messages",
	"runner_api_key_required":               "API 模式缺少 Key；请设置 %s 或在向导中输入",
	"runner_api_base_invalid":               "无效的模型 API Base URL %q",
	"runner_api_timeout":                    "API 请求超时：%s",
	"runner_api_failed":                     "API 请求失败：%v",
	"runner_api_status_failed":              "API 请求失败：HTTP %d：%s",
	"runner_api_auth_failed":                "API 认证失败：HTTP %d：%s。请检查模型 API Key 是否输入错误、是否已过期，或 Base URL 是否对应正确服务。",
	"runner_api_retry_exhausted":            "%v；检测到网络断流或临时连接问题，已自动重试仍失败",
	"runner_api_status_retry_exhausted":     "API 请求失败：HTTP %d：%s。检测到限流或服务端临时错误，已自动重试仍失败。",
	"runner_api_empty_response":             "API 响应为空",
	"runner_api_decode_failed":              "解析 API 响应失败：%v",
	"runner_api_empty_answer":               "API 响应中没有可解析的助手回答",
	"runner_bad_effort":                     "无效的推理强度 %q；可选值：low、medium、high、xhigh",
	"runner_tests_max":                      "测试次数不能超过 %d",
	"runner_tests_positive":                 "测试次数必须是正整数",
	"runner_codex_missing":                  "未在 PATH 中找到 codex 可执行文件",
	"runner_codex_timeout":                  "codex exec 超时：%s",
	"runner_codex_failed":                   "codex exec 失败：%s",
	"runner_event_too_large":                "codex 输出事件过大，无法解析",
	"runner_tool_used":                      "模型尝试使用工具，已中止本轮测试",
}

var enMessages = map[string]string{
	"error_prefix":                          "Error: %v",
	"unknown_command":                       "unknown command %q",
	"unexpected_args":                       "unexpected arguments: %v",
	"flag_model":                            "model name; omit to use the model from Codex local config",
	"flag_effort":                           "reasoning effort: low, medium, high, xhigh",
	"flag_backend":                          "test backend: auto, codex, api",
	"flag_api_format":                       "API call format: openai-chat, openai-responses, anthropic-messages",
	"flag_model_api_base":                   "model API base URL, for example https://api.example.com/v1",
	"flag_model_api_key":                    "model API key; using LD_GPT_CHECK_MODEL_API_KEY is recommended",
	"flag_codex_args":                       "extra Codex startup argument string; included in uploaded summaries",
	"flag_tests":                            "number of tests, default 5",
	"flag_timeout":                          "per-test timeout, for example 10m or 30s",
	"flag_upload":                           "upload result",
	"flag_anonymous":                        "show uploaded results anonymously by hiding Linux.do identity in public views",
	"flag_json":                             "print JSON",
	"flag_api_base":                         "Worker API base URL",
	"flag_lang":                             "UI language: zh-CN or en",
	"flag_update_yes":                       "update without prompting when a new version is available",
	"flag_update_check_only":                "only check for a new version without downloading or replacing",
	"timeout_positive":                      "timeout must be positive",
	"canceled":                              "operation canceled",
	"flag_count_positive":                   "%s must be a positive integer",
	"uploaded_run":                          "Uploaded run: %s",
	"uploaded_run_no_id":                    "Uploaded run.",
	"uploaded_provider":                     "Channel / bridge: %s",
	"codex_args_upload_notice":              "For fairness, custom Codex startup arguments are submitted with uploaded results; do not put keys or private data in them.",
	"login_success":                         "Logged in: %s (%s)",
	"credential_saved":                      "Credentials saved to: %s",
	"config_path":                           "Config file: %s",
	"config_api_base":                       "API base URL: %s",
	"config_language":                       "Language: %s",
	"config_not_logged_in":                  "Login status: not logged in",
	"config_logged_in":                      "Login status: logged in",
	"model_local_config":                    "Codex local config",
	"model_detected":                        "Detected Codex model: %s",
	"model_choose":                          "Could not confirm the current Codex model; choose a model",
	"model_choice_55":                       "1) GPT 5.5",
	"model_choice_54":                       "2) GPT 5.4",
	"model_choice_other":                    "3) Other model",
	"model_custom":                          "Enter model name",
	"logout_success":                        "Logged out.",
	"credential_removed":                    "Local credentials removed from config file: %s",
	"usage":                                 "LD-gpt-check\n\nUsage:\n  ld-gpt-check setup [--lang zh-CN|en]\n  ld-gpt-check login [--api-base-url URL] [--lang zh-CN|en]\n  ld-gpt-check run [--backend auto|codex|api] [-m MODEL] [--api-format openai-chat|openai-responses|anthropic-messages] [--model-api-base-url URL] [--model-api-key KEY] [--codex-args ARGS] [--suite candy_21] [--question-file FILE] [--question-url URL] [--no-remote-questions] [-r low|medium|high|xhigh] [-n TESTS] [--timeout 30m] [--upload] [--json]\n  ld-gpt-check run --list-suites [--question-file FILE] [--question-url URL] [--no-remote-questions]\n  ld-gpt-check update [--yes|--check-only]\n  ld-gpt-check config\n  ld-gpt-check whoami\n  ld-gpt-check logout",
	"update_check_failed":                   "Update check failed: %v",
	"update_available_noninteractive":       "New version %s is available (current %s). Run `ld-gpt-check update --yes` to update, or download: %s",
	"update_available_prompt":               "New version %s is available (current %s). Update now",
	"update_available":                      "New version %s is available (current %s): %s",
	"update_none":                           "Already on the latest version: %s",
	"update_declined":                       "Update canceled.",
	"update_downloading":                    "Downloading and verifying version %s...",
	"update_install_error":                  "Update failed: %v",
	"update_install_failed":                 "update failed",
	"update_success_restart":                "Updated to %s. Re-run the command.",
	"update_windows_pending":                "Downloaded version %s. The executable will be replaced after this process exits; re-run the command shortly.",
	"auth_missing_device_code":              "device start response missing device_code",
	"auth_missing_user_code":                "device start response missing user_code",
	"auth_missing_verification_uri":         "device start response missing verification URI",
	"auth_opening_browser":                  "Opening browser to complete Linux.do login...",
	"auth_manual_open":                      "If the browser does not open automatically, visit:",
	"auth_user_code":                        "User code:",
	"auth_expired":                          "device login expired; run login again",
	"auth_missing_access_token":             "device poll authorized without access_token",
	"auth_missing_user_id":                  "device poll authorized without user id",
	"auth_unexpected_status":                "unexpected login status: %s",
	"wizard_title":                          "LD-gpt-check setup wizard",
	"wizard_subtitle":                       "This wizard configures login, runs a test, and can upload the result.",
	"wizard_step_config":                    "Check configuration",
	"wizard_step_login":                     "Login status",
	"wizard_step_run":                       "Run test",
	"wizard_step_upload":                    "Upload result",
	"wizard_config_file":                    "Config file",
	"wizard_credentials_file":               "Credential file",
	"wizard_api_label":                      "API",
	"wizard_api_using":                      "Using API: %s",
	"wizard_api_base":                       "Worker API base URL",
	"wizard_language":                       "UI language (zh-CN/en)",
	"wizard_existing_credential":            "Existing login credential: %s",
	"wizard_unknown_user":                   "unknown user",
	"wizard_api_changed":                    "The API base URL changed; logging in again is recommended.",
	"wizard_relogin":                        "Log in again",
	"wizard_login_now":                      "Log in with Linux.do now (required for upload)",
	"wizard_run_now":                        "Run a test now",
	"wizard_done_next":                      "Setup complete. You can later run: ld-gpt-check run -r xhigh -n 5",
	"wizard_backend":                        "Test mode",
	"wizard_backend_codex":                  "1) Local Codex CLI",
	"wizard_backend_api":                    "2) API mode (no local Codex required)",
	"wizard_backend_api_primary":            "1) API mode (no local Codex required)",
	"wizard_backend_exit":                   "2) Skip test and exit the wizard",
	"wizard_backend_invalid":                "Enter a valid test mode: 1 or 2.",
	"wizard_model":                          "Model",
	"wizard_model_default":                  "Codex local config",
	"wizard_effort":                         "Reasoning effort",
	"wizard_tests":                          "Test count",
	"wizard_timeout":                        "Per-test timeout",
	"wizard_upload":                         "Upload result after run",
	"wizard_anonymous_upload":               "Show uploaded result anonymously",
	"wizard_anonymous_note":                 "Anonymous display hides your Linux.do username, avatar, and profile link in public views; the benchmark summary is still submitted and included in statistics.",
	"wizard_upload_needs_login":             "upload requires login; run the wizard again or execute ld-gpt-check login",
	"wizard_done":                           "Setup complete.",
	"wizard_ready_run":                      "Ready to start the local Codex test.",
	"wizard_codex_found":                    "Detected local Codex: %s",
	"wizard_codex_missing":                  "Local Codex was not detected; API mode can be used instead.",
	"wizard_use_api":                        "Use API mode for this test",
	"wizard_need_api_without_codex":         "API mode is required to continue testing without a local Codex environment.",
	"wizard_ready_api_run":                  "Ready to start the API test.",
	"wizard_api_format":                     "API call format",
	"wizard_api_format_chat":                "1) OpenAI Chat Completions",
	"wizard_api_format_responses":           "2) OpenAI Responses",
	"wizard_api_format_anthropic":           "3) Anthropic Messages",
	"wizard_api_model_hint":                 "API model",
	"wizard_api_model_hint_value":           "Enter the model name supported by your API service; choose GPT-5.5 / GPT-5.4 presets or enter a custom model.",
	"wizard_model_api_base":                 "Model API base URL",
	"wizard_model_api_key":                  "Model API key",
	"wizard_api_key_warning":                "Create a fresh temporary API key if possible, and destroy it right after this test to reduce leakage and dispute risk.",
	"wizard_codex_startup_args":             "Extra Codex startup arguments",
	"wizard_codex_startup_args_empty":       "empty",
	"wizard_skip_login":                     "Login skipped; local tests still work.",
	"wizard_upload_skipped":                 "Upload skipped; this result only appears in terminal output.",
	"wizard_questions_remote_failed":        "Remote questions could not be fetched; using the classic question: %v",
	"wizard_question_classic":               "%d) Classic: %s (%s)",
	"wizard_question_remote":                "%d) Remote: %s (%s)",
	"wizard_question":                       "Choose question",
	"wizard_question_invalid":               "Enter a question number from 1 to %d.",
	"prompt_non_empty":                      "Enter a non-empty value.",
	"prompt_bool":                           "Enter yes/no.",
	"prompt_effort":                         "Allowed values: low, medium, high, xhigh.",
	"prompt_int_range":                      "Enter an integer from %d to %d.",
	"prompt_duration":                       "Enter a valid duration, for example 30s, 10m, or 1h.",
	"api_not_logged_in":                     "not logged in; run ld-gpt-check login first",
	"api_base_empty":                        "api base URL is empty; set LD_GPT_CHECK_API_BASE_URL or run login --api-base-url URL",
	"api_base_invalid":                      "invalid api base URL %q",
	"api_base_bad_scheme":                   "invalid api base URL scheme %q",
	"api_http_nil":                          "http client is nil",
	"api_request_failed":                    "%s %s failed: %v",
	"api_status_failed":                     "request failed: HTTP %d: %s",
	"api_decode_failed":                     "decode response from %s failed: %v",
	"api_empty_device_code":                 "device_code is required",
	"api_upload_id_required":                "upload failed: upload_id is required",
	"api_upload_model_required":             "upload failed: could not confirm model name; pass -m, for example -m gpt-5.5",
	"api_upload_provider_base_url_required": "upload failed: could not determine Codex provider base URL; upgrade or check Codex config",
	"api_upload_tests_invalid":              "upload failed: attempt_count must be greater than 0",
	"api_upload_cases_mismatch":             "upload failed: attempts length must equal attempt_count",
	"api_upload_questions_mismatch":         "upload failed: questions length must equal question_count",
	"run_status_start":                      "Starting run: model=%s, reasoning=%s, %d total cases",
	"run_status_case_start":                 "▶ [%d/%d] Running %s #%d",
	"run_status_case_done":                  "• [%d/%d] %s, time %.1fs, TPS %.1f",
	"run_status_case_error":                 "✗ [%d/%d] failed: %v",
	"run_status_question":                   "Question: %s",
	"run_status_failed_question":            "Failed question: %s",
	"report_panel_title":                    "Run overview",
	"report_metric_accuracy":                "Accuracy",
	"report_metric_correct":                 "Correct",
	"report_metric_time":                    "Time",
	"report_metric_tps":                     "TPS",
	"report_metric_input":                   "Input",
	"report_metric_reason":                  "Reason",
	"report_summary":                        "\nCorrect: %d/%d (%.1f%%), avg time %.1fs, avg TPS %.1f\n",
	"runner_model_required":                 "model is required; pass -m or --model",
	"runner_backend_invalid":                "invalid test backend %q; use auto, codex, or api",
	"runner_api_format_invalid":             "invalid API call format %q; use openai-chat, openai-responses, or anthropic-messages",
	"runner_api_key_required":               "API mode requires a key; set %s or enter it in the wizard",
	"runner_api_base_invalid":               "invalid model API base URL %q",
	"runner_api_timeout":                    "API request timed out after %s",
	"runner_api_failed":                     "API request failed: %v",
	"runner_api_status_failed":              "API request failed: HTTP %d: %s",
	"runner_api_auth_failed":                "API authentication failed: HTTP %d: %s. Check whether the model API key is wrong or expired, and whether the base URL points to the correct service.",
	"runner_api_retry_exhausted":            "%v; detected a broken stream or temporary connection issue, and automatic retries still failed",
	"runner_api_status_retry_exhausted":     "API request failed: HTTP %d: %s. Rate limiting or a temporary server error was detected, and automatic retries still failed.",
	"runner_api_empty_response":             "API response is empty",
	"runner_api_decode_failed":              "decode API response failed: %v",
	"runner_api_empty_answer":               "API response did not contain a parseable assistant answer",
	"runner_bad_effort":                     "invalid reasoning effort %q; use low, medium, high, or xhigh",
	"runner_tests_max":                      "tests must be <= %d",
	"runner_tests_positive":                 "tests must be a positive integer",
	"runner_codex_missing":                  "codex executable not found in PATH",
	"runner_codex_timeout":                  "codex exec timed out after %s",
	"runner_codex_failed":                   "codex exec failed: %s",
	"runner_event_too_large":                "codex output event is too large to parse",
	"runner_tool_used":                      "model attempted to use a tool; test aborted",
}
