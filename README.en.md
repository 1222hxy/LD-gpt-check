# 🍬 LD-gpt-check

> 🧪 A lightweight Codex / GPT benchmark for the Linux.do community.  
> Run `ld-gpt-check`, follow the guided flow, and get a reproducible local benchmark in minutes.

中文文档：[README.md](README.md) · Website: [codexgo.yhklab.com](https://codexgo.yhklab.com) · Commands: [docs/commands.md](docs/commands.md) · Release workflow: [docs/release-workflow.md](docs/release-workflow.md)

## ✨ What It Is

LD-gpt-check runs benchmark questions through your local Codex CLI, then records correctness, token usage, latency, TPS, model metadata, and optional community uploads.

The main experience is intentionally simple:

1. 📦 Install the CLI.
2. 🪄 Run `ld-gpt-check`.
3. ✅ Follow the wizard to choose a model, run tests, log in, and upload results if you want.

For daily use, the wizard is the recommended path.

## 🚀 Quick Start

You do not need Go and you do not need to compile anything. Make sure Codex CLI is installed and logged in, then download the right binary:

- 🪟 Windows Intel / AMD: [`ld-gpt-check_windows_amd64.zip`](https://download.yhklab.com/ld-gpt-check/latest/ld-gpt-check_windows_amd64.zip)
- 🐧 Most Linux servers / VPS: [`ld-gpt-check_linux_amd64.tar.gz`](https://download.yhklab.com/ld-gpt-check/latest/ld-gpt-check_linux_amd64.tar.gz)
- 🐧 Linux ARM64: [`ld-gpt-check_linux_arm64.tar.gz`](https://download.yhklab.com/ld-gpt-check/latest/ld-gpt-check_linux_arm64.tar.gz)

If the mirror is unavailable, download the same file from [GitHub Releases](https://github.com/1222hxy/LD-gpt-check/releases/latest).

Start the guided flow:

```bash
ld-gpt-check
```

The wizard helps with model selection, reasoning effort, test count, Linux.do login, upload choice, and result display.

Binary download notes, detailed commands, JSON output, local question files, and self-hosting options are documented in [docs/commands.md](docs/commands.md).

## 🍬 Default Question

The default benchmark is the original `candy_21` question. The expected answer is `21`; validation only requires the final answer to contain an independent `21`.

Remote and local question banks are also supported, but regular users can ignore them and use the wizard.

## 📊 What It Records

- ✅ Correctness
- 🔢 Input, output, and reasoning tokens
- ⏱️ Runtime and TPS
- 🤖 Model, reasoning effort, and Codex version
- 🧬 Normalized provider base URL metadata

Uploaded results appear in the community dashboard for recent runs, trends, model comparisons, and statistical checks.

## 🔐 Privacy

Uploads are optional. LD-gpt-check only uploads benchmark summaries and a short answer preview.

It does not upload local Codex databases, full prompt history, full model responses, raw Codex JSONL events, OpenAI keys, or Linux.do OAuth secrets.

## ☁️ Self Hosting

The backend uses Cloudflare Workers + D1. The frontend and backend are kept separate:

- `worker/`: Worker API, OAuth, D1 schema, migrations
- `frontend/`: static website and admin pages
- `dashboard/`: statistics dashboard
- `cmd/`, `internal/`: Go CLI

Most users do not need to self-host. Deployment instructions are available in [docs/cloudflare-worker-deploy.md](docs/cloudflare-worker-deploy.md).

## 🤝 Open Source

The CLI, Worker backend, D1 schema, question management, frontend, and dashboard are open source. Production secrets are not stored in the repository.
