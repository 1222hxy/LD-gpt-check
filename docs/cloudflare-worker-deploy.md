# Deploy LD-gpt-check to Cloudflare Workers

本文档部署 LD-gpt-check 到同一个 Cloudflare Worker。产品前端不是 Cloudflare Pages 项目；前端先由 Vite 构建到 `frontend/dist/`，再通过 Worker Static Assets 随 `worker/` 一起部署。后端负责 Linux.do OAuth、CLI 设备登录、D1 存储和 benchmark submission 上传接口。

必须遵守：

- 生产环境只部署 `worker/`。
- 不要把 `frontend/` 单独部署到 Cloudflare Pages。
- 每次改前端后，先运行 `cd frontend && npm run build`，再运行 `cd ../worker && ../frontend/node_modules/.bin/wrangler deploy`。
- `worker/wrangler.toml` 必须包含 `[assets] directory = "../frontend/dist"`。

相关文件：

- `frontend/`：Vite 静态首页源码
- `frontend/dist/`：构建后的 Worker Static Assets 输入目录
- `worker/src/index.ts`：Worker API 实现
- `worker/schema.sql`：D1 数据库 schema
- `worker/wrangler.toml.example`：Wrangler 配置模板
- `docs/api-reference.md`：API 端点说明

## 1. 前置条件

安装本地工具：

```bash
npm install -g wrangler
wrangler login
```

需要准备：

- Cloudflare 账号，并启用 Workers 和 D1。
- Linux.do OAuth 应用。
- 一个 Worker 域名，例如 `https://ld-gpt-check.example.workers.dev`。

Linux.do OAuth callback URL 设置为：

```text
https://YOUR_WORKER_DOMAIN/auth/linuxdo/callback
```

## 2. 创建 D1 数据库

```bash
wrangler d1 create ld-gpt-check
```

记录输出中的 `database_id`，后续写入 `worker/wrangler.toml`。

## 3. 创建 Worker 配置

从模板复制配置：

```bash
cp worker/wrangler.toml.example worker/wrangler.toml
```

编辑 `worker/wrangler.toml`：

```toml
name = "ld-gpt-check"
main = "src/index.ts"
compatibility_date = "2026-06-28"

[assets]
directory = "../frontend/dist"
binding = "ASSETS"
not_found_handling = "none"
run_worker_first = ["/account", "/admin", "/health", "/api/*", "/device", "/auth/*", "/logout"]

[vars]
BASE_URL = "https://YOUR_WORKER_DOMAIN"
LINUXDO_AUTH_URL = "https://connect.linux.do/oauth2/authorize"
LINUXDO_TOKEN_URL = "https://connect.linux.do/oauth2/token"
LINUXDO_USERINFO_URL = "https://connect.linux.do/api/user"
ALLOWED_ORIGINS = "https://YOUR_WORKER_DOMAIN"
ADMIN_LINUXDO_IDS = "29368"

[[d1_databases]]
binding = "DB"
database_name = "ld-gpt-check"
database_id = "YOUR_DATABASE_ID"
```

说明：

- `BASE_URL` 必须和实际 Worker 公网地址一致，不能带结尾 `/`。
- `[assets]` 让 Worker 同时服务产品首页；`run_worker_first` 中的路径继续交给后端逻辑处理。
- Linux.do OAuth URL 请以你的 Linux.do 开发者后台为准。
- `ALLOWED_ORIGINS` 是可选 CORS 白名单，多个 origin 用英文逗号分隔；`BASE_URL` 的 origin 会自动允许。
- `ADMIN_LINUXDO_IDS` 是允许进入 `/admin` 管理后台的 Linux.do 用户 UID，多个 UID 用英文逗号分隔。当前默认管理员 UID 是 `29368`。
- `wrangler.toml` 可能包含环境差异，生产项目可不提交该文件，只提交 `wrangler.toml.example`。

## 4. 设置 secrets

在 `worker/` 目录执行：

```bash
cd worker
wrangler secret put LINUXDO_CLIENT_ID
wrangler secret put LINUXDO_CLIENT_SECRET
wrangler secret put TOKEN_SECRET
```

`TOKEN_SECRET` 用于 hash CLI token、web session 和 OAuth state。建议使用 32 字节以上随机值：

```bash
openssl rand -base64 32
```

安全要求：

- 不要把 OAuth client secret 或 `TOKEN_SECRET` 写入仓库。
- 不要在日志中打印 access token、device code、cookie 或 OAuth code。
- 如果泄露了 `TOKEN_SECRET`，应轮换 secret，并让用户重新登录。

可选：启用 Cloudflare Turnstile 保护浏览器授权页。

1. 在 Cloudflare Turnstile 创建站点，域名填写 Worker 域名。
2. 将 site key 写入 `worker/wrangler.toml`：

```toml
TURNSTILE_SITE_KEY = "0x4AAAA..."
```

3. 将 secret key 写入 Worker secret：

```bash
wrangler secret put TURNSTILE_SECRET_KEY
```

同时配置 `TURNSTILE_SITE_KEY` 和 `TURNSTILE_SECRET_KEY` 后，`/device` 授权页会显示人机校验，后端会在批准设备授权前校验 token。

## 5. 初始化数据库 schema

在 `worker/` 目录执行：

```bash
wrangler d1 execute ld-gpt-check --file=./schema.sql --remote
```

如果是在已有 submission 表的远程 D1 上升级，按顺序执行迁移：

```bash
wrangler d1 execute ld-gpt-check --file=./migrations/0001_add_submission_diagnostics.sql --remote
wrangler d1 execute ld-gpt-check --file=./migrations/0002_add_upload_v3_and_linuxdo_profile.sql --remote
wrangler d1 execute ld-gpt-check --file=./migrations/0003_add_question_banks.sql --remote
```

验证表是否创建成功：

```bash
wrangler d1 execute ld-gpt-check --remote --command="SELECT name FROM sqlite_master WHERE type='table';"
```

应看到 `users`、`device_sessions`、`access_tokens`、`benchmark_submissions`、`benchmark_question_results`、`benchmark_attempts`、`oauth_states`、`web_sessions`、`question_banks`、`bridges`、`bridge_base_urls` 等表。

如果是从旧版 `runs/run_cases` schema 升级到当前 MVP，建议新建 D1 数据库或手动迁移数据后再执行 schema。当前 schema 使用新的 submission 表，不再依赖旧表。

## 6. 本地开发验证

在 `worker/` 目录启动本地 Worker：

```bash
wrangler dev
```

健康检查：

```bash
curl http://localhost:8787/health
```

预期响应：

```json
{"ok":true}
```

创建设备登录会话：

```bash
curl -X POST http://localhost:8787/api/device/start
```

本地 OAuth 回调需要 Linux.do 应用允许本地 callback；如果没有配置本地 callback，可以只验证健康检查和 API 结构，然后在部署后验证完整登录流程。

## 7. 部署到 Cloudflare

先构建前端静态资源：

```bash
cd frontend
npm run build
```

再部署同一个 Worker：

```bash
cd ../worker
../frontend/node_modules/.bin/wrangler deploy
```

部署完成后验证：

```bash
curl https://YOUR_WORKER_DOMAIN/
curl https://YOUR_WORKER_DOMAIN/health
curl -X POST https://YOUR_WORKER_DOMAIN/api/device/start
```

再用 CLI 验证登录：

```bash
LD_GPT_CHECK_API_BASE_URL=https://YOUR_WORKER_DOMAIN bin/ld-gpt-check login
bin/ld-gpt-check whoami
bin/ld-gpt-check run -m gpt-5.5 -r xhigh -n 1 --upload
```

浏览器验证：

```text
https://YOUR_WORKER_DOMAIN/
https://YOUR_WORKER_DOMAIN/account
https://YOUR_WORKER_DOMAIN/admin
```

## 8. 远程题目管理

管理后台入口是 Worker 渲染页面，题目管理是后台中的一个静态前端模块，读写通过受保护 JSON API 完成。访问：

```text
https://YOUR_WORKER_DOMAIN/admin
```

要求：

- 浏览器已通过 Linux.do 登录。
- 当前 Linux.do 用户 UID 在 `ADMIN_LINUXDO_IDS` 中，例如 `29368`。

页面会调用 `GET/POST /api/v1/admin/questions`。保存后的题库写入 D1 表 `question_banks.questions_json`。CLI 默认拉取：

```text
https://YOUR_WORKER_DOMAIN/api/v1/questions
```

### 中转站映射管理

管理员可以打开：

```text
https://YOUR_WORKER_DOMAIN/admin/bridges
```

页面会调用 `GET/POST /api/v1/admin/bridges`。保存后的映射写入 D1 表 `bridges` 和 `bridge_base_urls`。上传时 Worker 会把 `codex_provider_base_url` 自动分类为 `official`、`bridge` 或 `unknown_bridge`。

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

用户也可以完全不使用远程题库，直接在本地运行：

```bash
bin/ld-gpt-check run --question-file ./questions.json --list-suites
bin/ld-gpt-check run -m gpt-5.5 --question-file ./questions.json --suite custom_1 -n 5
```

首页应显示产品前端；`/account` 未登录时会显示 Linux.do 登录入口。登录成功后会回到账号页，展示当前用户、CLI 配置命令、最近上传记录，并提供网页退出登录按钮。

## 8. 生产检查清单

部署前确认：

- `BASE_URL` 是 HTTPS 公网地址。
- `frontend/dist/` 已由最新前端源码构建。
- `worker/wrangler.toml` 的 `[assets]` 指向 `../frontend/dist`。
- Linux.do callback URL 完全匹配 `/auth/linuxdo/callback`。
- D1 binding 名称是 `DB`。
- D1 schema 已在 remote 数据库执行。
- 三个 secrets 已设置：`LINUXDO_CLIENT_ID`、`LINUXDO_CLIENT_SECRET`、`TOKEN_SECRET`。
- 如果启用 Turnstile，`TURNSTILE_SITE_KEY` 和 `TURNSTILE_SECRET_KEY` 同时存在。
- `/health` 返回 `{"ok":true}`。
- `/device` 页面能正常打开。
- `/account` 页面能正常打开，Linux.do 登录后能显示当前用户信息和退出登录按钮。
- CLI 能完成 `login`、`whoami`、`logout`。
- 上传后，D1 的 `benchmark_submissions`、`benchmark_question_results` 和 `benchmark_attempts` 表有数据，且新上传的 `benchmark_submissions.codex_provider_base_url` 和 `codex_channel` 非空。
- `/account` 默认只展示最近 10 条上传记录，用户可以删除单条记录或清空自己的测试数据。
- 浏览器响应包含 `x-request-id`、`x-content-type-options`、`content-security-policy` 等安全响应头。

## 9. 常见问题

### OAuth callback 提示 invalid oauth state

常见原因：

- 浏览器阻止 cookie。
- `BASE_URL` 和实际访问域名不一致。
- callback URL 配错，导致 state cookie 不在同一站点下。
- 用户从一个域名开始登录，却回调到另一个域名。

处理：

1. 确认 `BASE_URL` 与 Worker 域名完全一致。
2. 确认 Linux.do callback URL 是 `https://YOUR_WORKER_DOMAIN/auth/linuxdo/callback`。
3. 清理浏览器中该域名 cookie 后重试。

### oauth token exchange failed

常见原因：

- `LINUXDO_CLIENT_ID` 或 `LINUXDO_CLIENT_SECRET` 错误。
- Linux.do token endpoint 配置错误。
- OAuth 应用未启用对应 callback URL。

处理：

```bash
cd worker
wrangler secret put LINUXDO_CLIENT_ID
wrangler secret put LINUXDO_CLIENT_SECRET
```

然后重新部署或等待 Worker secret 生效。

### D1 报 no such table

说明 remote D1 没有执行 schema，或绑定了错误的 database id。

处理：

```bash
cd worker
wrangler d1 execute ld-gpt-check --file=./schema.sql --remote
```

同时检查 `worker/wrangler.toml` 中的 `database_id`。

### CLI 提示 unauthorized

常见原因：

- 本地 token 已被 logout 撤销。
- `TOKEN_SECRET` 被更换，旧 token hash 不再匹配。
- CLI 指向了另一个 Worker 环境。

处理：

```bash
bin/ld-gpt-check logout
LD_GPT_CHECK_API_BASE_URL=https://YOUR_WORKER_DOMAIN bin/ld-gpt-check login
```

### API 提示 too many requests

后端对设备登录、OAuth、上传 submission 做了轻量 D1 限流。常见触发原因：

- CLI 或脚本过快轮询设备登录。
- 同一 IP 短时间频繁开始 OAuth 登录。
- 同一用户短时间上传大量 submission。

处理：

1. 等待限流窗口结束后重试。
2. 确认 CLI 遵守设备登录响应中的 `interval`。
3. 如果是生产误伤，可以在 Worker 代码中调整对应 `enforceRateLimit` 或 `enforceUserRateLimit` 的阈值。

### 上传成功但看不到 submission

检查当前用户和 D1 数据：

```bash
bin/ld-gpt-check whoami
cd worker
wrangler d1 execute ld-gpt-check --remote --command="SELECT id, user_id, model, created_at FROM benchmark_submissions ORDER BY created_at DESC LIMIT 5;"
```

确认 CLI 登录的是同一个 Worker 环境。

## 10. 运维建议

- 使用 Cloudflare Dashboard 查看 Worker logs 和错误率。
- 保留 `x-request-id`，用户反馈错误时让用户提供该响应头。
- 定期备份 D1，尤其是公开分享或排行榜功能上线后。
- 增加生产和预览环境时，分别使用不同 D1 数据库和 OAuth 应用。
- schema 变更采用向后兼容迁移，先加字段和索引，再切代码，最后清理旧字段。
- 将 Worker 放在 Cloudflare 默认 DDoS 防护之后，并在 Dashboard 中关注 4xx、5xx、429 的异常变化。
- 不建议开放 `ALLOWED_ORIGINS = "*"`；CLI 不依赖浏览器 CORS，网页调用只允许明确域名。
