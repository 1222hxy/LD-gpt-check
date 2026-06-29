# 📘 LD-gpt-check 命令参考

这里保留完整命令说明。普通用户建议先运行 `ld-gpt-check` 进入向导。

## 📦 安装

推荐普通用户直接下载对应系统的二进制文件，不需要安装 Go，也不需要自己编译。Windows 下载 `.exe` 后直接运行；Linux/macOS 下载后执行 `chmod +x ld-gpt-check_*`，再运行对应文件。

二进制命名规则：

```text
ld-gpt-check_SYSTEM_ARCH
```

常见选择：

- 🪟 Windows Intel/AMD：[ld-gpt-check_windows_amd64.exe](https://github.com/1222hxy/LD-gpt-check/releases/latest/download/ld-gpt-check_windows_amd64.exe)
- 🪟 Windows ARM：[ld-gpt-check_windows_arm64.exe](https://github.com/1222hxy/LD-gpt-check/releases/latest/download/ld-gpt-check_windows_arm64.exe)
- 🍎 macOS Apple Silicon：[ld-gpt-check_darwin_arm64](https://github.com/1222hxy/LD-gpt-check/releases/latest/download/ld-gpt-check_darwin_arm64)
- 🍎 macOS Intel：[ld-gpt-check_darwin_amd64](https://github.com/1222hxy/LD-gpt-check/releases/latest/download/ld-gpt-check_darwin_amd64)
- 🐧 Linux Intel/AMD：[ld-gpt-check_linux_amd64](https://github.com/1222hxy/LD-gpt-check/releases/latest/download/ld-gpt-check_linux_amd64)
- 🐧 Linux ARM64：[ld-gpt-check_linux_arm64](https://github.com/1222hxy/LD-gpt-check/releases/latest/download/ld-gpt-check_linux_arm64)
- 🧩 树莓派 32 位：[ld-gpt-check_linux_armv7](https://github.com/1222hxy/LD-gpt-check/releases/latest/download/ld-gpt-check_linux_armv7) 或 [ld-gpt-check_linux_armv6](https://github.com/1222hxy/LD-gpt-check/releases/latest/download/ld-gpt-check_linux_armv6)

所有二进制文件和校验文件都在 [GitHub Releases](https://github.com/1222hxy/LD-gpt-check/releases/latest)。

测试后端：

- 本机 Codex：macOS/Linux 下 `codex` 在 PATH 中，Windows 下可找到 `codex.cmd`。
- API 模式：不需要本机 Codex，按向导输入模型 API Base URL 和 Key。

## 🪄 向导模式

```bash
ld-gpt-check
```

向导会先检测本机是否有 Codex。没有 Codex 时会询问是否改用 API 模式；有 Codex 时也可以选择 API 模式。

API 模式会提示输入 Base URL 和 Key。建议创建新的临时 API Key，测试完成后立即销毁。

## 🧪 直接运行

```bash
ld-gpt-check run -r xhigh -n 5
ld-gpt-check run -m gpt-5.5 -r xhigh -n 5
LD_GPT_CHECK_MODEL_API_KEY="你的临时 API Key" ld-gpt-check run --backend api --api-format openai-chat --model-api-base-url "https://api.krill-ai.com/codex/v1" -m gpt-5.4 -n 1
```

如果不传 `-m`，CLI 会尽量读取本机 Codex 配置中的具体模型。识别不到时，会让用户选择 GPT 5.5、GPT 5.4 或自定义模型。

API 模式必须提供模型名，可以用 `-m` 或 `--model`。

常用参数：

- `-m, --model`：指定模型
- `--backend`：`auto`、`codex`、`api`，默认 `auto`
- `--api-format`：`openai-chat`、`openai-responses`、`anthropic-messages`
- `--model-api-base-url`：模型 API Base URL，例如 `https://api.openai.com/v1` 或中转站地址
- `--model-api-key`：模型 API Key；更推荐用 `LD_GPT_CHECK_MODEL_API_KEY`
- `--codex-args`：本机 Codex 模式的额外启动参数字符串，例如 `--codex-args '-c model_provider=my_provider'`
- `-r, --reasoning-effort`：`low`、`medium`、`high`、`xhigh`
- `-n, --tests`：测试次数，默认 5
- `--upload`：上传结果
- `--anonymous`：匿名展示上传结果，社区页面会隐藏 Linux.do 用户名、头像和主页链接；测试摘要仍会提交并参与统计
- `--json`：输出 JSON
- `--timeout`：单轮 Codex 超时，默认 `30m`

示例：

```bash
ld-gpt-check run -m gpt-5.5 -r xhigh -n 5 --upload
ld-gpt-check run -m gpt-5.5 -r xhigh -n 5 --upload --anonymous
ld-gpt-check run -m gpt-5.5 -r xhigh -n 5 --json
ld-gpt-check run -r high -n 10 --timeout 10m
ld-gpt-check run -r xhigh -n 5 --codex-args '-c model_provider=my_provider'
```

为了确保公平，`--codex-args` 会写入上传摘要里的 `codex_invocation`，随结果一同提交。不要在启动参数里放 API Key、token 或其他私密信息。

### API 调用格式

OpenAI Chat Completions：

```bash
LD_GPT_CHECK_MODEL_API_KEY="你的临时 API Key" \
ld-gpt-check run --backend api \
  --api-format openai-chat \
  --model-api-base-url "https://api.krill-ai.com/codex/v1" \
  -m gpt-5.4 -n 1
```

OpenAI Responses：

```bash
LD_GPT_CHECK_MODEL_API_KEY="你的临时 API Key" \
ld-gpt-check run --backend api \
  --api-format openai-responses \
  --model-api-base-url "https://api.openai.com/v1" \
  -m gpt-5.4 -n 1
```

Anthropic Messages：

```bash
LD_GPT_CHECK_MODEL_API_KEY="你的临时 API Key" \
ld-gpt-check run --backend api \
  --api-format anthropic-messages \
  --model-api-base-url "https://api.anthropic.com/v1" \
  -m claude-sonnet-4-5 -n 1
```

可用环境变量：

```text
LD_GPT_CHECK_MODEL_API_KEY
LD_GPT_CHECK_MODEL_API_BASE_URL
LD_GPT_CHECK_API_FORMAT
```

Base URL 可以是 provider base URL，也可以误填完整 endpoint，例如 `.../chat/completions`、`.../responses` 或 `.../messages`。CLI 会避免重复拼接路径；上传摘要里的 provider base URL 会去除 query、fragment 和 user info，方便识别官方渠道或中转站。

## 🔐 登录、上传和退出

```bash
ld-gpt-check login
ld-gpt-check whoami
ld-gpt-check run -m gpt-5.5 -r xhigh -n 5 --upload
ld-gpt-check logout
```

登录流程：

```text
CLI -> Cloudflare Worker -> Linux.do OAuth -> Worker 生成设备 token -> CLI 保存设备授权 secret
```

本地配置文件：

```text
ld-gpt-check.toml
```

配置模板：

```text
ld-gpt-check.example.toml
```

示例：

```toml
api_base_url = "https://codexgo.yhklab.com"
language = "zh-CN"

[device_authorization]
secret = "ldgc_..."
```

`device_authorization.secret` 是设备授权密钥，等同登录凭证，必须保密；不要提交到 Git，也不要贴到日志或工单里。

自托管后端登录：

```bash
LD_GPT_CHECK_API_BASE_URL="https://YOUR_WORKER_DOMAIN" ld-gpt-check login
```

## 🧩 题库

CLI 默认尝试拉取远程题库：

```text
https://codexgo.yhklab.com/api/v1/questions
```

如果远程拉取失败，会自动回退到内置 `candy_21` 原题。

题库 JSON 字段、grader 类型和配置规范见 [题库配置格式](question-bank-format.md)。

列出题目：

```bash
ld-gpt-check run --list-suites
```

只使用内置或本地题库：

```bash
ld-gpt-check run --no-remote-questions --list-suites
```

运行指定题目：

```bash
ld-gpt-check run --suite candy_21 -n 5
```

### 本地题库

题库 JSON 示例：

```json
{
  "schema_version": "1",
  "questions": [
    {
      "id": "custom_1",
      "version": "1",
      "title": "简单数字题",
      "prompt": "不使用任何外部工具回答：10 + 11 等于多少？",
      "tags": ["math"],
      "grader": {
        "type": "number",
        "expected": "21",
        "independent_match": true
      }
    }
  ]
}
```

使用本地题库：

```bash
ld-gpt-check run --question-file ./questions.json --list-suites
ld-gpt-check run -m gpt-5.5 --question-file ./questions.json --suite custom_1 -n 5
```

支持的 grader：

- `number`：抽取数字并和 `expected` 比较；`independent_match: true` 表示只要出现独立数字即可。
- `exact`：全文匹配；可用 `trim_space`、`case_sensitive` 控制严格程度。
- `regex`：正则匹配；如果有捕获组，记录第一个捕获组为 extracted answer。

## ☁️ 部署相关命令

详细步骤见 [Cloudflare Worker 部署指南](cloudflare-worker-deploy.md)。常用命令如下：

```bash
cp worker/wrangler.toml.example worker/wrangler.toml
wrangler d1 create ld-gpt-check
cd worker
# 新建空库初始化：
npx wrangler d1 execute ld-gpt-check --remote --file=./schema.sql
# 已有远程库升级请按 docs/database-migrations.md 执行 migration，不要重复执行 schema.sql。
wrangler secret put LINUXDO_CLIENT_ID
wrangler secret put LINUXDO_CLIENT_SECRET
wrangler secret put TOKEN_SECRET
wrangler secret put TURNSTILE_SECRET_KEY
```

构建前端并部署同一个 Worker：

```bash
cd frontend
npm run build
cd ../worker
../frontend/node_modules/.bin/wrangler deploy
```
