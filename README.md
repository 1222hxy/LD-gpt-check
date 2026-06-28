# LD-gpt-check

LD-gpt-check is a minimal Go CLI for checking whether the local Codex CLI can solve benchmark questions. The built-in default suite is the original candy math prompt, `candy_21`, whose expected answer is `21`. It reports token usage, reasoning tokens, elapsed time, TPS, and accuracy. It can also log in through a Cloudflare Worker using Linux.do OAuth and upload privacy-scoped summary results to Cloudflare D1.

## Prerequisites

- Go 1.22+ for local builds.
- Codex CLI installed and available as `codex` on macOS/Linux or `codex.cmd` on Windows.
- A Cloudflare account with Workers and D1 enabled for upload/login features.
- A Linux.do OAuth application. Set its callback URL to:

```text
https://YOUR_WORKER_DOMAIN/auth/linuxdo/callback
```

## Build and Run Locally

Build the CLI:

```bash
go build -o bin/ld-gpt-check ./cmd/ld-gpt-check
```

For first-time use, run the guided setup:

```bash
bin/ld-gpt-check setup
```

The wizard asks for the Worker API URL, helps complete Linux.do login, stores the returned platform token in `ld-gpt-check.toml`, and can run/upload a test result.

The CLI uses Chinese by default. To use English for the current process:

```bash
LD_GPT_CHECK_LANG=en bin/ld-gpt-check help
```

You can also choose and persist the UI language during setup:

```bash
bin/ld-gpt-check setup --lang en
```

Run the benchmark:

```bash
bin/ld-gpt-check run -m gpt-5.5 -r xhigh -n 5
```

List available question suites:

```bash
bin/ld-gpt-check run --list-suites
```

Run selected suites:

```bash
bin/ld-gpt-check run -m gpt-5.5 --suite candy_21 -n 5
```

Load additional questions from a JSON bank:

```bash
bin/ld-gpt-check run -m gpt-5.5 --question-file ./questions.json --suite candy_21,custom_1
bin/ld-gpt-check run -m gpt-5.5 --question-url https://example.com/questions.json --suite custom_1
```

The built-in candy prompt must remain unchanged. Its grader only requires an independent `21` in the final answer, so `21` passes but `121` does not. Prompts tell Codex not to use external tools; the runner also starts Codex with ignored user config/rules in a temporary read-only workspace and fails a run if Codex emits tool-call events.

Print machine-readable output:

```bash
bin/ld-gpt-check run -m gpt-5.5 -r xhigh -n 5 --json
```

Reasoning effort supports `low`, `medium`, `high`, and `xhigh`. The default effort is `medium`; the default test count is `1`. Each Codex run has a default timeout of `30m`; override it with `--timeout 10m` or `--timeout 90s`.

## Login and Upload

Set the Worker API URL before the first login, or pass it with `--api-base-url`:

```bash
export LD_GPT_CHECK_API_BASE_URL="https://YOUR_WORKER_DOMAIN"
bin/ld-gpt-check login
```

The CLI opens a browser for Linux.do login. On SSH, WSL, or remote servers, copy the printed URL and 9-digit code manually.

Check the logged-in user:

```bash
bin/ld-gpt-check whoami
```

Show the local config path and login status:

```bash
bin/ld-gpt-check config
```

Upload a run:

```bash
bin/ld-gpt-check run -m gpt-5.5 -r xhigh -n 5 --upload
```

Log out:

```bash
bin/ld-gpt-check logout
```

Local config is stored as TOML in the same directory as the executable:

```text
ld-gpt-check.toml
```

Override the path when needed:

```bash
LD_GPT_CHECK_CONFIG=/path/to/ld-gpt-check.toml bin/ld-gpt-check config
```

Use `ld-gpt-check.example.toml` as the template. The real config stores the Worker API URL, selected language, access token, and basic user profile:

```toml
api_base_url = "https://YOUR_WORKER_DOMAIN"
access_token = "..."
language = "zh-CN"

[user]
id = "..."
username = "..."
```

## Cloudflare Worker Deployment

This is the backend API deployment. It is separate from the static frontend in `frontend/`.

- `worker/` contains the Cloudflare Worker API, D1 schema, OAuth flow, and backend secrets.
- `frontend/` contains the Vite static site and deploys to Cloudflare Pages.
- Do not put D1 bindings or OAuth client secrets in `frontend/`.

Run backend commands from `worker/`:

```bash
cd worker
wrangler deploy
```

Run frontend commands from `frontend/`:

```bash
cd frontend
npm run build
npm run deploy
```

For a detailed backend deployment and troubleshooting guide, see [`docs/cloudflare-worker-deploy.md`](docs/cloudflare-worker-deploy.md).

Install Wrangler:

```bash
npm install -g wrangler
wrangler login
```

Create a D1 database:

```bash
wrangler d1 create ld-gpt-check
```

Copy the returned `database_id` into `worker/wrangler.toml`:

```bash
cp worker/wrangler.toml.example worker/wrangler.toml
```

Apply the schema:

```bash
cd worker
wrangler d1 execute ld-gpt-check --file=./schema.sql --remote
```

Configure Worker variables in `worker/wrangler.toml`:

```toml
[vars]
BASE_URL = "https://YOUR_WORKER_DOMAIN"
LINUXDO_AUTH_URL = "https://connect.linux.do/oauth2/authorize"
LINUXDO_TOKEN_URL = "https://connect.linux.do/oauth2/token"
LINUXDO_USERINFO_URL = "https://connect.linux.do/api/user"
```

The Linux.do URLs above are placeholders; confirm the exact OAuth endpoints in your Linux.do developer settings.

Set secrets:

```bash
wrangler secret put LINUXDO_CLIENT_ID
wrangler secret put LINUXDO_CLIENT_SECRET
wrangler secret put TOKEN_SECRET
```

Deploy:

```bash
wrangler deploy
```

After deployment, test:

```bash
curl https://YOUR_WORKER_DOMAIN/health
LD_GPT_CHECK_API_BASE_URL=https://YOUR_WORKER_DOMAIN bin/ld-gpt-check login
```

## API Documentation

The Worker API is documented in:

- [`docs/api-design.md`](docs/api-design.md) for design principles, versioning, error handling, privacy boundaries, and extension rules.
- [`docs/api-reference.md`](docs/api-reference.md) for endpoint-level request and response details.
- [`docs/openapi.yaml`](docs/openapi.yaml) for the target OpenAPI 3.1 contract.

The current Worker keeps legacy login paths such as `/api/device/start`, while uploads use `POST /api/v1/submissions`. Legacy `/api/runs` now returns `410 Gone`.

## Project Layout

```text
cmd/ld-gpt-check/     CLI entrypoint
internal/api/         Worker API client and upload payloads
internal/auth/        Device login flow
internal/config/      Local config path and persistence
internal/questions/   Built-in and external benchmark question banks
internal/runner/      Codex execution and JSON event parsing
internal/report/      Terminal table rendering
internal/system/      OS helpers
frontend/             Vite static frontend, deployed to Cloudflare Pages
worker/src/           Cloudflare Worker TypeScript source
worker/schema.sql     D1 database schema
```

## Privacy Notes

Uploads include only summary metrics and answer previews. The CLI does not upload local Codex databases, full prompt history, or full model responses.
