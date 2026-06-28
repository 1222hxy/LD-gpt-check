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
	"error_prefix":                  "错误：%v",
	"unknown_command":               "未知命令 %q",
	"unexpected_args":               "存在无法识别的参数：%v",
	"flag_model":                    "模型名称",
	"flag_effort":                   "推理强度：low、medium、high、xhigh",
	"flag_tests":                    "测试次数",
	"flag_timeout":                  "每轮超时时间，例如 10m 或 30s",
	"flag_upload":                   "上传结果",
	"flag_json":                     "输出 JSON",
	"flag_api_base":                 "Worker API 地址",
	"flag_lang":                     "界面语言：zh-CN 或 en",
	"timeout_positive":              "timeout 必须是正数",
	"canceled":                      "操作已取消",
	"flag_count_positive":           "%s 必须是正整数",
	"uploaded_run":                  "已上传运行结果：%s",
	"uploaded_run_no_id":            "已上传运行结果。",
	"login_success":                 "已登录：%s (%s)",
	"credential_saved":              "凭证已保存到：%s",
	"config_path":                   "配置文件：%s",
	"config_api_base":               "API 地址：%s",
	"config_language":               "语言：%s",
	"config_not_logged_in":          "登录状态：未登录",
	"config_logged_in":              "登录状态：已登录",
	"logout_success":                "已退出登录。",
	"credential_removed":            "本地凭证已从配置文件中移除：%s",
	"usage":                         "LD-gpt-check 0.1.0\n\n用法：\n  ld-gpt-check setup [--lang zh-CN|en]\n  ld-gpt-check login [--api-base-url URL] [--lang zh-CN|en]\n  ld-gpt-check run -m MODEL [--suite candy_21] [--question-file FILE] [--question-url URL] [-r low|medium|high|xhigh] [-n TESTS] [--timeout 30m] [--upload] [--json]\n  ld-gpt-check run --list-suites [--question-file FILE] [--question-url URL]\n  ld-gpt-check config\n  ld-gpt-check whoami\n  ld-gpt-check logout",
	"auth_missing_device_code":      "设备登录响应缺少 device_code",
	"auth_missing_user_code":        "设备登录响应缺少 user_code",
	"auth_missing_verification_uri": "设备登录响应缺少验证链接",
	"auth_opening_browser":          "正在打开浏览器完成 Linux.do 登录……",
	"auth_manual_open":              "如果浏览器没有自动打开，请访问：",
	"auth_user_code":                "验证码：",
	"auth_expired":                  "设备登录已过期，请重新运行 login",
	"auth_missing_access_token":     "设备授权成功但响应缺少 access_token",
	"auth_missing_user_id":          "设备授权成功但响应缺少用户 ID",
	"auth_unexpected_status":        "未知登录状态：%s",
	"wizard_title":                  "LD-gpt-check 向导",
	"wizard_api_using":              "正在使用 API：%s",
	"wizard_api_base":               "Worker API 地址",
	"wizard_language":               "界面语言 (zh-CN/en)",
	"wizard_existing_credential":    "当前已有登录凭证：%s",
	"wizard_unknown_user":           "未知用户",
	"wizard_api_changed":            "API 地址已变化，建议重新登录以获取对应后端的凭证。",
	"wizard_relogin":                "是否重新登录",
	"wizard_login_now":              "是否现在登录 Linux.do（上传结果需要登录）",
	"wizard_run_now":                "是否现在运行一次测试",
	"wizard_done_next":              "向导完成。之后可运行：ld-gpt-check run -m gpt-5.5 -r xhigh -n 5",
	"wizard_model":                  "模型",
	"wizard_effort":                 "推理强度",
	"wizard_tests":                  "测试次数",
	"wizard_timeout":                "每轮超时",
	"wizard_upload":                 "跑完后上传结果",
	"wizard_upload_needs_login":     "上传需要先登录；请重新运行向导或执行 ld-gpt-check login",
	"wizard_done":                   "向导完成。",
	"prompt_non_empty":              "请输入有效内容。",
	"prompt_bool":                   "请输入 yes/no 或 是/否。",
	"prompt_effort":                 "可选值：low, medium, high, xhigh。",
	"prompt_int_range":              "请输入 %d 到 %d 之间的整数。",
	"prompt_duration":               "请输入有效时长，例如 30s、10m、1h。",
	"api_not_logged_in":             "尚未登录；请先运行 ld-gpt-check login",
	"api_base_empty":                "API 地址为空；请设置 LD_GPT_CHECK_API_BASE_URL 或运行 login --api-base-url URL",
	"api_base_invalid":              "无效的 API 地址 %q",
	"api_base_bad_scheme":           "无效的 API 地址协议 %q",
	"api_http_nil":                  "HTTP client 为空",
	"api_request_failed":            "%s %s 请求失败：%v",
	"api_status_failed":             "请求失败：HTTP %d：%s",
	"api_decode_failed":             "解析 %s 响应失败：%v",
	"api_empty_device_code":         "device_code 不能为空",
	"api_upload_id_required":        "上传失败：upload_id 不能为空",
	"api_upload_model_required":     "上传失败：model 不能为空",
	"api_upload_tests_invalid":      "上传失败：attempt_count 必须大于 0",
	"api_upload_cases_mismatch":     "上传失败：attempts 数量必须等于 attempt_count",
	"api_upload_questions_mismatch": "上传失败：questions 数量必须等于 question_count",
	"report_summary":                "\n正确：%d/%d (%.1f%%)，平均耗时 %.1fs，平均 TPS %.1f\n",
	"runner_model_required":         "缺少模型名称；请使用 -m 或 --model 指定",
	"runner_bad_effort":             "无效的推理强度 %q；可选值：low、medium、high、xhigh",
	"runner_tests_max":              "测试次数不能超过 %d",
	"runner_tests_positive":         "测试次数必须是正整数",
	"runner_codex_missing":          "未在 PATH 中找到 codex 可执行文件",
	"runner_codex_timeout":          "codex exec 超时：%s",
	"runner_codex_failed":           "codex exec 失败：%s",
	"runner_event_too_large":        "codex 输出事件过大，无法解析",
	"runner_tool_used":              "模型尝试使用工具，已中止本轮测试",
}

var enMessages = map[string]string{
	"error_prefix":                  "Error: %v",
	"unknown_command":               "unknown command %q",
	"unexpected_args":               "unexpected arguments: %v",
	"flag_model":                    "model name",
	"flag_effort":                   "reasoning effort: low, medium, high, xhigh",
	"flag_tests":                    "number of tests",
	"flag_timeout":                  "per-test timeout, for example 10m or 30s",
	"flag_upload":                   "upload result",
	"flag_json":                     "print JSON",
	"flag_api_base":                 "Worker API base URL",
	"flag_lang":                     "UI language: zh-CN or en",
	"timeout_positive":              "timeout must be positive",
	"canceled":                      "operation canceled",
	"flag_count_positive":           "%s must be a positive integer",
	"uploaded_run":                  "Uploaded run: %s",
	"uploaded_run_no_id":            "Uploaded run.",
	"login_success":                 "Logged in: %s (%s)",
	"credential_saved":              "Credentials saved to: %s",
	"config_path":                   "Config file: %s",
	"config_api_base":               "API base URL: %s",
	"config_language":               "Language: %s",
	"config_not_logged_in":          "Login status: not logged in",
	"config_logged_in":              "Login status: logged in",
	"logout_success":                "Logged out.",
	"credential_removed":            "Local credentials removed from config file: %s",
	"usage":                         "LD-gpt-check 0.1.0\n\nUsage:\n  ld-gpt-check setup [--lang zh-CN|en]\n  ld-gpt-check login [--api-base-url URL] [--lang zh-CN|en]\n  ld-gpt-check run -m MODEL [--suite candy_21] [--question-file FILE] [--question-url URL] [-r low|medium|high|xhigh] [-n TESTS] [--timeout 30m] [--upload] [--json]\n  ld-gpt-check run --list-suites [--question-file FILE] [--question-url URL]\n  ld-gpt-check config\n  ld-gpt-check whoami\n  ld-gpt-check logout",
	"auth_missing_device_code":      "device start response missing device_code",
	"auth_missing_user_code":        "device start response missing user_code",
	"auth_missing_verification_uri": "device start response missing verification URI",
	"auth_opening_browser":          "Opening browser to complete Linux.do login...",
	"auth_manual_open":              "If the browser does not open automatically, visit:",
	"auth_user_code":                "User code:",
	"auth_expired":                  "device login expired; run login again",
	"auth_missing_access_token":     "device poll authorized without access_token",
	"auth_missing_user_id":          "device poll authorized without user id",
	"auth_unexpected_status":        "unexpected login status: %s",
	"wizard_title":                  "LD-gpt-check setup wizard",
	"wizard_api_using":              "Using API: %s",
	"wizard_api_base":               "Worker API base URL",
	"wizard_language":               "UI language (zh-CN/en)",
	"wizard_existing_credential":    "Existing login credential: %s",
	"wizard_unknown_user":           "unknown user",
	"wizard_api_changed":            "The API base URL changed; logging in again is recommended.",
	"wizard_relogin":                "Log in again",
	"wizard_login_now":              "Log in with Linux.do now (required for upload)",
	"wizard_run_now":                "Run a test now",
	"wizard_done_next":              "Setup complete. You can later run: ld-gpt-check run -m gpt-5.5 -r xhigh -n 5",
	"wizard_model":                  "Model",
	"wizard_effort":                 "Reasoning effort",
	"wizard_tests":                  "Test count",
	"wizard_timeout":                "Per-test timeout",
	"wizard_upload":                 "Upload result after run",
	"wizard_upload_needs_login":     "upload requires login; run the wizard again or execute ld-gpt-check login",
	"wizard_done":                   "Setup complete.",
	"prompt_non_empty":              "Enter a non-empty value.",
	"prompt_bool":                   "Enter yes/no.",
	"prompt_effort":                 "Allowed values: low, medium, high, xhigh.",
	"prompt_int_range":              "Enter an integer from %d to %d.",
	"prompt_duration":               "Enter a valid duration, for example 30s, 10m, or 1h.",
	"api_not_logged_in":             "not logged in; run ld-gpt-check login first",
	"api_base_empty":                "api base URL is empty; set LD_GPT_CHECK_API_BASE_URL or run login --api-base-url URL",
	"api_base_invalid":              "invalid api base URL %q",
	"api_base_bad_scheme":           "invalid api base URL scheme %q",
	"api_http_nil":                  "http client is nil",
	"api_request_failed":            "%s %s failed: %v",
	"api_status_failed":             "request failed: HTTP %d: %s",
	"api_decode_failed":             "decode response from %s failed: %v",
	"api_empty_device_code":         "device_code is required",
	"api_upload_id_required":        "upload failed: upload_id is required",
	"api_upload_model_required":     "upload failed: model is required",
	"api_upload_tests_invalid":      "upload failed: attempt_count must be greater than 0",
	"api_upload_cases_mismatch":     "upload failed: attempts length must equal attempt_count",
	"api_upload_questions_mismatch": "upload failed: questions length must equal question_count",
	"report_summary":                "\nCorrect: %d/%d (%.1f%%), avg time %.1fs, avg TPS %.1f\n",
	"runner_model_required":         "model is required; pass -m or --model",
	"runner_bad_effort":             "invalid reasoning effort %q; use low, medium, high, or xhigh",
	"runner_tests_max":              "tests must be <= %d",
	"runner_tests_positive":         "tests must be a positive integer",
	"runner_codex_missing":          "codex executable not found in PATH",
	"runner_codex_timeout":          "codex exec timed out after %s",
	"runner_codex_failed":           "codex exec failed: %s",
	"runner_event_too_large":        "codex output event is too large to parse",
	"runner_tool_used":              "model attempted to use a tool; test aborted",
}
