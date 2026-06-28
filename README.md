# 🍬 LD-gpt-check

> 🧪 一个面向 Linux.do 社区的 Codex / GPT Bench。  
> 打开终端运行 `ld-gpt-check`，跟着向导走，就能完成测试、登录和上传。

🌐 [English](README.en.md) · 🏠 [在线页面](https://codexgo.yhklab.com) · 📘 [命令参考](docs/commands.md) · ☁️ [部署指南](docs/cloudflare-worker-deploy.md)

## ✨ 这是什么

LD-gpt-check 是一个轻量但完整的社区 Bench，用本机 Codex CLI 跑固定题集，记录模型是否答对、token 消耗、耗时和 TPS，并可选择上传到社区 Dashboard 做统计观察。

它不是复杂平台，也不是只能开发者使用的脚本。核心体验是：

1. 📦 安装 CLI。
2. 🪄 运行 `ld-gpt-check`。
3. ✅ 按向导选择模型、测试次数、是否登录上传。

除了第一次安装，日常使用基本只需要记住一个命令。

## 🚀 最简单的开始

先确保已经安装并登录 Codex CLI，然后从 [GitHub Releases](https://github.com/1222hxy/LD-gpt-check/releases/latest) 下载适合你系统的二进制文件：

- 🪟 Windows Intel / AMD：下载 `ld-gpt-check_windows_amd64.zip`
- 🐧 Linux 普通服务器 / VPS：下载 `ld-gpt-check_linux_amd64.tar.gz`
- 🐧 Linux ARM64：下载 `ld-gpt-check_linux_arm64.tar.gz`

开发者也可以用 Go 直接安装：

```bash
go install github.com/1222hxy/LD-gpt-check/cmd/ld-gpt-check@latest
```

之后直接运行：

```bash
ld-gpt-check
```

CLI 会进入向导，自动带你完成常见操作：

- 🤖 选择或识别 Codex 使用的模型。
- 🧠 选择 reasoning effort。
- 🔢 设置测试次数，默认 5 次。
- 🔐 通过 Linux.do 设备码登录。
- ☁️ 选择是否上传结果。
- 📊 查看彩色终端结果和统计摘要。

需要二进制下载说明、手动参数、JSON 输出、本地题库、自托管后端等高级用法时，再看 [命令参考](docs/commands.md)。

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
CLI -> Cloudflare Worker -> Linux.do OAuth -> Worker 生成平台 token -> CLI 保存 token
```

凭证会持久化到当前目录的 `ld-gpt-check.toml`。仓库保留 `ld-gpt-check.example.toml` 作为模板，方便自托管或迁移配置。

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
