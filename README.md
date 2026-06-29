# 🍬 LD-gpt-check

> 🧪 一个面向 Linux.do 社区的 Codex / GPT Bench。  
> 打开终端运行 `ld-gpt-check`，跟着向导走，就能完成测试、登录和上传。

🌐 [English](README.en.md) · 🏠 [在线页面](https://codexgo.yhklab.com) · 📘 [命令参考](docs/commands.md) · ☁️ [部署指南](docs/cloudflare-worker-deploy.md)

## ✨ 这是什么

LD-gpt-check 是一个轻量但完整的社区 Bench，用本机 Codex CLI 跑固定题集，记录模型是否答对、token 消耗、耗时和 TPS，并可选择上传到社区 Dashboard 做统计观察。

除了本机 Codex CLI，也支持 API 模式。API 模式目前覆盖 OpenAI Chat Completions、OpenAI Responses 和 Anthropic Messages 三种常见协议，所以大多数模型服务和中转站都能接入：OpenAI / Codex、Claude / Claude Code、DeepSeek、国产兼容 OpenAI 协议的模型服务等，都可以通过填写 Base URL、API Key 和模型名来测试。

它不是复杂平台，也不是只能开发者使用的脚本。核心体验是：

1. 📦 安装 CLI。
2. 🪄 运行 `ld-gpt-check`。
3. ✅ 按向导选择模型、测试次数、是否登录上传。

除了第一次安装，日常使用基本只需要记住一个命令。

## 🚀 最简单的开始

不需要安装 Go，也不需要自己编译。先确保已经安装并登录 Codex CLI，然后直接下载适合你系统的二进制文件：

- 🪟 Windows Intel / AMD：[下载 `ld-gpt-check_windows_amd64.exe`](https://github.com/1222hxy/LD-gpt-check/releases/latest/download/ld-gpt-check_windows_amd64.exe)
- 🐧 Linux 普通服务器 / VPS：[下载 `ld-gpt-check_linux_amd64`](https://github.com/1222hxy/LD-gpt-check/releases/latest/download/ld-gpt-check_linux_amd64)
- 🐧 Linux ARM64：[下载 `ld-gpt-check_linux_arm64`](https://github.com/1222hxy/LD-gpt-check/releases/latest/download/ld-gpt-check_linux_arm64)

更多系统版本在 [GitHub Releases](https://github.com/1222hxy/LD-gpt-check/releases/latest) 的 Assets 里。

下载后就能直接用。Windows 可以直接运行 `.exe`；Linux/macOS 下载后执行 `chmod +x ld-gpt-check_*`，然后运行对应文件即可。

之后直接运行：

```bash
ld-gpt-check
```

CLI 会进入向导，自动带你完成常见操作：

- 🤖 选择或识别 Codex 使用的模型。
- ⚡ 自动识别 CC Switch 当前上游配置，用真实中转站信息跑测试和归一统计。
- 🔌 没有本机 Codex 时，可改用 API 模式，支持 OpenAI / Anthropic 兼容接口。
- 🧠 选择 reasoning effort。
- 🔢 设置测试次数，默认 5 次。
- 🔐 通过 Linux.do 设备码登录。
- ☁️ 选择是否上传结果。
- 📊 查看彩色终端结果和统计摘要。

需要二进制下载说明、手动参数、JSON 输出、本地题库、自托管后端等高级用法时，再看 [命令参考](docs/commands.md)。

## ⚡ CC Switch 支持

如果你在用 CC Switch，LD-gpt-check 会尽量自动识别它当前给 Codex 使用的上游配置。这个能力很实用：很多时候 Codex 本机配置里只会显示 `127.0.0.1` 之类的本地代理地址，而工具会继续读取 CC Switch 的配置或数据库，找到真正的 API Base URL、API 格式和当前 Codex provider，避免上传统计被错误归到本机代理。

这对使用 Codex App 搭配 CC Switch 的用户尤其友好：通常不需要手动复制 Base URL 或 Key，向导会在检测到可用配置后提示是否沿用。

向导里的体验是：

- 先检测本机 Codex 环境。
- 再检测 CC Switch 是否存在，以及是否有 Codex 专用 provider。
- 如果 Codex 可用，优先继续用本机 Codex 跑测试，同时提示是否用 CC Switch 当前上游做统计归一。
- 如果 Codex 不可用，但 CC Switch 里有 Codex API 配置，可以询问是否沿用它作为 API 模式继续测试。
- 终端会明确提示本次提取到的配置来源，例如 `CC Switch 当前上游配置`、`OpenAI 官方登录配置` 或 `本机 Codex CLI 配置`。

安全边界也做了处理：API Key 只在本次进程内存里使用，不会写入 `ld-gpt-check.toml`，也不会上传到服务器或打印到终端。

默认会识别常见位置：

- Windows：`C:\Users\<你>\.cc-switch`
- macOS：`~/.cc-switch`、`~/Library/Application Support/CC Switch` 等常见应用目录
- Linux：`~/.cc-switch`、`~/.config/cc-switch`

如果你的 CC Switch 数据库放在特殊位置，可以用环境变量显式指定：

```bash
LD_GPT_CHECK_CC_SWITCH_DB=/path/to/cc-switch.db ld-gpt-check
```

### 🤖 让 Agent 帮你下载并跑一轮

仓库根目录提供了 `ld-gpt-check-binary-test` skill，支持 Windows、macOS 和 Linux。想让 Agent 自动下载适合当前系统的最新二进制，并用内置题库跑 1 轮本地测试时，直接复制这句给 Agent：

```text
请使用项目根目录的 ld-gpt-check-binary-test skill，帮我在当前系统（Windows/macOS/Linux）下载适合的 LD-gpt-check 最新二进制，并用内置题库跑 1 轮本地测试，不要上传结果；最后告诉我二进制位置和测试摘要。
```

## 🍬 默认测试题

默认题是原始糖果题 `candy_21`，正确答案为 `21`。判定规则非常直接：最终回答中只要出现独立的 `21` 就算通过，`121` 这种连在其他数字里的内容不算。

CLI 也支持远程题库和本地题库。普通用户可以完全不用管这些，向导会自动使用可用题目；想自己创建题目的用户可以查看 [命令参考](docs/commands.md#本地题库)。

## 📊 你会看到什么

每次测试会记录：

- ✅ 正确 / 错误
- 🔢 input tokens、output tokens、reasoning tokens
- ⏱️ 耗时和 TPS
- 🤖 模型、reasoning effort、Codex 版本
- 🧬 provider base URL 的规范化识别结果

上传后可以在 Dashboard 查看最近结果、趋势、模型对比和统计检查。

## 🔐 登录与上传

登录走设备码流程，CLI 不会接触 Linux.do OAuth client secret：

```text
CLI -> Cloudflare Worker -> Linux.do OAuth -> Worker 生成设备 token -> CLI 保存设备授权 secret
```

设备授权 secret 会持久化到当前目录的 `ld-gpt-check.toml`。它等同登录凭证，必须保密。仓库保留 `ld-gpt-check.example.toml` 作为模板，方便自托管或迁移配置。

## 🛡️ 隐私边界

上传是可选的。默认只上传 benchmark 需要的摘要数据和短 answer preview。

不会上传：

- 🚫 本地 Codex 数据库
- 🚫 完整 prompt 历史
- 🚫 完整模型回答
- 🚫 原始 Codex JSONL event
- 🚫 OpenAI key 或 Linux.do OAuth secret

用户可以在账号页面删除自己的 benchmark 数据。

## ☁️ 自托管

后端使用 Cloudflare Workers + D1，前端与后端代码分离：

- `worker/`：Worker API、OAuth、D1 schema、migrations
- `frontend/`：静态首页和管理页
- `dashboard/`：统计 Dashboard
- `cmd/`、`internal/`：Go CLI

如果只是使用工具，不需要部署后端；直接运行向导即可。想自己部署时，按 [Cloudflare Worker 部署指南](docs/cloudflare-worker-deploy.md) 操作。

## 📚 文档

- 📘 [命令参考](docs/commands.md)
- 📖 [API Reference](docs/api-reference.md)
- 🧩 [API 设计](docs/api-design.md)
- ☁️ [Cloudflare Worker 部署](docs/cloudflare-worker-deploy.md)
- 🚀 [GitHub Release 工作流](docs/release-workflow.md)
- 📝 [Linux.do 发帖草稿](docs/linuxdo-post.md)

## 🤝 开源

项目完整开源，包括 CLI、Worker 后端、D1 schema、题库管理、静态前端和 Dashboard。生产环境 secrets 不进入仓库。
