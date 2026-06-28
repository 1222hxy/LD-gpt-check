# LD-gpt-check API Reference

本文档描述 LD-gpt-check Worker API。CLI 登录仍保留部分 legacy `/api/...` 路径；benchmark 上传使用 `/api/v1/submissions`。OpenAPI 草案见 [openapi.yaml](./openapi.yaml)。

## 通用约定

Base URL 示例：

```text
https://codexgo.yhklab.com
```

公共 JSON API 使用：

```http
Accept: application/json
Content-Type: application/json
```

受保护端点使用：

```http
Authorization: Bearer ldgc_...
```

当前错误体兼容旧 CLI，保留顶层 `error` 字符串，并增加机器码和 request id：

```json
{
  "error": "unauthorized",
  "code": "unauthorized",
  "request_id": "req_or_cf_ray"
}
```

后续如果不再需要兼容旧 CLI，可以演进为嵌套错误体：

```json
{
  "error": {
    "code": "unauthorized",
    "message": "Unauthorized",
    "request_id": "optional"
  }
}
```

## GET /health

健康检查。

认证：不需要。

响应：

```json
{
  "ok": true
}
```

常见用途：

```bash
curl https://codexgo.yhklab.com/health
```

## GET /api/v1/questions

获取当前启用的题库 JSON。CLI 默认会尝试拉取该端点；如果拉取失败，会回退到本地内置 `candy_21` 原题。

Legacy alias：`GET /api/questions`。

认证：不需要。

Query 参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `slug` | string | 可选。指定题库 slug；不传时返回默认或当前启用题库 |

响应：

```json
{
  "schema_version": "1",
  "questions": [
    {
      "id": "candy_21",
      "version": "1",
      "title": "糖果形状口味保证题",
      "prompt": "不使用任何外部工具回答以下问题：...",
      "tags": ["math", "pigeonhole"],
      "grader": {
        "type": "number",
        "expected": "21",
        "independent_match": true
      }
    }
  ]
}
```

题目字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | suite id，CLI 用 `--suite` 选择 |
| `version` | string | 题目版本 |
| `title` | string | 展示名称 |
| `prompt` | string | 发送给 Codex 的完整题面 |
| `tags` | string[] | 可选标签 |
| `grader` | object | 判题配置 |

grader 类型：

| type | 必填字段 | 说明 |
| --- | --- | --- |
| `number` | `expected` | 抽取数字并比较；`independent_match: true` 表示答案里出现独立数字即可 |
| `exact` | `expected` | 全文匹配，可用 `trim_space`、`case_sensitive` 调整 |
| `regex` | `pattern` | 正则匹配；有捕获组时记录第一个捕获组 |

错误：

- `404 not_found`：指定的 `slug` 不存在或未启用。

## GET /admin

管理后台入口页。只有管理员可访问；管理员登录后也会在 `/account` 页面看到该入口，非管理员隐藏入口。

认证：Web session cookie。未登录会显示 Linux.do 登录入口；已登录但不在管理员列表时返回 `403`。

## GET /admin/questions

题目管理页面。该页面和 API Worker 同源，页面文件由 `frontend/public/admin/questions/index.html` 构建到静态资源；具体读写通过受保护 JSON API 完成。

认证：Web session cookie。未登录会跳转 Linux.do OAuth；已登录但不在管理员列表时返回 `403`。

管理员判断：

- Linux.do 用户 UID 在 `ADMIN_LINUXDO_IDS` 中。
- 当前默认管理员 UID：`29368`。

功能：

- `GET /admin`：查看管理后台入口。
- `GET /admin/questions`：查看当前题库 JSON 编辑页。
- `GET /admin/bridges`：查看中转站 base URL 映射管理页。
- `GET /api/v1/admin/questions`：读取当前可编辑题库，要求管理员 Web session。
- `POST /api/v1/admin/questions`：保存题库到 D1 表 `question_banks.questions_json`，要求管理员 Web session 和同源请求。
- `GET /api/v1/admin/bridges`：读取全局中转站和 base URL 映射。
- `POST /api/v1/admin/bridges`：保存一个中转站及其多个 base URL；服务端会规范化 URL 并阻止同一 base URL 归属多个中转站。

## POST /api/device/start

目标路径：`POST /api/v1/device-authorizations`。当前 Worker 已支持该 v1 alias。

创建设备授权会话。CLI 登录时首先调用该接口，然后打开 `verification_uri_complete` 或提示用户访问 `verification_uri` 并输入 `user_code`。

认证：不需要。

请求体：空。

响应：

```json
{
  "device_code": "dc_xxx",
  "user_code": "123-456-789",
  "verification_uri": "https://codexgo.yhklab.com/device",
  "verification_uri_complete": "https://codexgo.yhklab.com/device?code=123-456-789",
  "expires_in": 600,
  "interval": 3
}
```

字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `device_code` | string | CLI 轮询使用的机密 code，只展示给本地 CLI，不展示在网页 |
| `user_code` | string | 用户在浏览器授权页输入的短验证码 |
| `verification_uri` | string | 授权页地址 |
| `verification_uri_complete` | string | 带验证码的授权页地址 |
| `expires_in` | number | 授权会话有效秒数 |
| `interval` | number | CLI 推荐轮询间隔秒数 |

错误：

- `500 internal_error`：数据库写入失败或随机值生成失败。

## POST /api/device/poll

目标路径：`POST /api/v1/device-authorizations/token`。当前 Worker 已支持该 v1 alias。

CLI 轮询设备授权状态。用户在浏览器授权成功后，该接口返回 access token。

认证：不需要。

请求体：

```json
{
  "device_code": "dc_xxx"
}
```

响应：等待授权。

```json
{
  "status": "pending"
}
```

响应：轮询过快。

```json
{
  "status": "slow_down"
}
```

响应：过期。

```json
{
  "status": "expired"
}
```

响应：授权完成。

```json
{
  "status": "authorized",
  "access_token": "ldgc_xxx",
  "user": {
    "id": "6f1a...",
    "username": "alice",
    "login": "alice",
    "name": "Alice",
    "email": "alice@privaterelay.linux.do",
    "avatar_url": "https://cdn.ldstatic.com/user_avatar/...",
    "avatar_template": "https://cdn.ldstatic.com/user_avatar/...",
    "active": true,
    "trust_level": 2,
    "silenced": false
  }
}
```

Linux.do Connect 实测 `GET /api/user` 可返回 `id`、`sub`、`username`、`login`、`name`、`email`、`avatar_template`、`avatar_url`、`active`、`trust_level`、`silenced`、`external_ids`、`api_key`。服务端保存除 `api_key` 外的资料字段；`api_key` 属于凭证类字段，禁止落库、输出或转发。

状态说明：

| status | CLI 行为 |
| --- | --- |
| `pending` | 继续按 `interval` 轮询 |
| `slow_down` | 增加轮询间隔后继续 |
| `expired` | 停止登录并提示重新开始 |
| `authorized` | 保存 `access_token` 和 `user` |

错误：

- `400 bad_request`：缺少或无效 `device_code`。
- `500 internal_error`：数据库读写失败。

## GET /device

浏览器授权页面。用户登录 Linux.do 后在该页面输入 `user_code` 并授权 CLI。

认证：Web session cookie。未登录时页面提供 Linux.do 登录入口。

Query 参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `code` | string | 可选，预填用户验证码 |

响应：

- `200 text/html`：授权页面。

## POST /api/device/approve

目标路径：`POST /api/v1/device-authorizations/approve`。当前 Worker 已支持该 v1 alias，HTML 表单仍使用 legacy 路径。

提交用户验证码，将待授权设备会话标记为 approved。

认证：Web session cookie。

请求体可以是 HTML 表单：

```text
user_code=123-456-789
```

也可以是 JSON：

```json
{
  "user_code": "123-456-789"
}
```

响应：JSON 请求。

```json
{
  "ok": true
}
```

响应：HTML 表单请求。

```html
<p>已授权。可以回到终端继续。</p>
```

错误：

- `400 bad_request`：缺少验证码、验证码无效或已过期。
- `401 unauthorized`：未登录网页 session。
- `403 forbidden`：跨站提交被拒绝。
- `429 rate_limited`：同一 IP 提交过快。

## GET /auth/linuxdo/start

开始 Linux.do OAuth 登录。

认证：不需要。

Query 参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `next` | string | 可选，OAuth 完成后回跳的站内路径 |

响应：

- `302`：跳转到 Linux.do OAuth 授权地址。

安全要求：

- `next` 必须是站内路径。
- 服务端设置一次性 `ldgc_oauth_state` cookie。

## GET /auth/linuxdo/callback

Linux.do OAuth 回调。服务端交换 OAuth token、读取用户信息、创建 Web session，然后跳回授权页面或 `next` 指定路径。

认证：OAuth state cookie。

Query 参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `code` | string | Linux.do 返回的 authorization code |
| `state` | string | OAuth state |

响应：

- `302`：登录成功后跳回站内页面。

错误：

- `400 bad_request`：state 无效或过期。
- `502 upstream_error`：OAuth token exchange 或 userinfo 请求失败。

## GET /api/me

目标路径：`GET /api/v1/me`。当前 Worker 已支持该 v1 alias。

返回当前 Bearer token 对应的用户。

认证：Bearer token。

响应：

```json
{
  "user": {
    "id": "6f1a...",
    "username": "alice"
  }
}
```

错误：

- `401 unauthorized`：缺少 token、token 无效、token 已撤销或 token 已过期。

## POST /api/v1/submissions

上传一次 CLI benchmark submission。CLI 当前通过 `ld-gpt-check run --upload` 调用该接口。旧 `/api/runs` 和 `/api/v1/runs` 返回 `410 Gone`。

认证：Bearer token。

请求体：

```json
{
  "upload_id": "upl_0123456789abcdef0123456789abcdef",
  "upload_schema_version": 4,
  "client_version": "0.1.0",
  "model": "gpt-5.5",
  "reasoning_effort": "xhigh",
  "question_count": 1,
  "attempt_count": 5,
  "correct": 5,
  "accuracy": 100,
  "avg_input_tokens": 120,
  "avg_output_tokens": 12,
  "avg_reason_tokens": 30,
  "avg_time_seconds": 2.4,
  "avg_tps": 5,
  "anonymous": false,
  "started_at": "2026-06-28T08:00:00Z",
  "finished_at": "2026-06-28T08:00:12Z",
  "duration_seconds": 12,
  "question_suite": "candy_21",
  "client_timezone": "+08:00",
  "os": "linux",
  "arch": "amd64",
  "codex_version": "codex 0.1.0",
  "codex_model_source": "explicit",
  "codex_model_provider": "openai",
  "codex_provider_host": "api.openai.com",
  "codex_provider_base_url": "https://api.openai.com/v1",
  "codex_sandbox": "read-only",
  "codex_ephemeral": true,
  "codex_skip_git_repo_check": true,
  "codex_disabled_features": ["memories"],
  "questions": [
    {
      "question_id": "candy_21",
      "question_version": "1",
      "question_title": "糖果形状口味保证题",
      "grader_type": "number",
      "expected_answer": "21",
      "prompt_hash": "sha256_hex",
      "tests": 5,
      "correct": 5,
      "accuracy": 100,
      "avg_input_tokens": 120,
      "avg_output_tokens": 12,
      "avg_reason_tokens": 30,
      "avg_time_seconds": 2.4,
      "avg_tps": 5
    }
  ],
  "attempts": [
    {
      "question_id": "candy_21",
      "question_version": "1",
      "case_index": 1,
      "status": "completed",
      "is_correct": true,
      "expected_answer": "21",
      "extracted_answer": "21",
      "answer_preview": "21",
      "answer_preview_truncated": false,
      "answer_hash": "sha256_hex_of_full_answer",
      "input_tokens": 120,
      "cached_input_tokens": 20,
      "output_tokens": 12,
      "reasoning_tokens": 30,
      "total_tokens": 132,
      "time_seconds": 2.4,
      "tps": 5,
      "codex_thread_id": "019f0e72-d150-7b31-85ae-da9c93f517bd",
      "event_count": 4,
      "event_types": ["thread.started", "turn.completed"],
      "tool_event_detected": false,
      "answer_chars": 2,
      "error_code": "",
      "started_at": "2026-06-28T08:00:00Z",
      "finished_at": "2026-06-28T08:00:02Z",
      "timeout_seconds": 1800
    }
  ]
}
```

响应：

```json
{
  "id": "submission_uuid",
  "duplicate": false
}
```

字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `client_version` | string | LD-gpt-check CLI 版本 |
| `model` | string | Codex 使用的模型名 |
| `reasoning_effort` | string | 推理强度，如 `low`、`medium`、`high`、`xhigh` |
| `upload_id` | string | CLI 生成的幂等上传 ID，同一用户重复提交会返回同一 submission |
| `upload_schema_version` | integer | 上传 payload 版本；当前 Go CLI 使用 `4`，服务端要求 v4 及以上 |
| `question_count` | integer | 本次包含的问题数量 |
| `attempt_count` | integer | 本次总尝试次数 |
| `correct` | integer | 正确尝试次数 |
| `accuracy` | number | 正确率百分比，范围 `0..100` |
| `avg_input_tokens` | number | 平均输入 tokens |
| `avg_output_tokens` | number | 平均输出 tokens |
| `avg_reason_tokens` | number | 平均 reasoning tokens |
| `avg_time_seconds` | number | 平均耗时秒数 |
| `avg_tps` | number | 平均 tokens per second |
| `anonymous` | boolean | 可选。为 `true` 时公共/community 展示隐藏 Linux.do 身份，用户占位固定为 `匿名`；测试数据、统计字段和提交记录仍正常保存、返回并参与统计 |
| `started_at` / `finished_at` | string | 本次 benchmark 开始和结束时间，UTC ISO-8601 |
| `duration_seconds` | number | 整次 benchmark wall-clock 耗时 |
| `question_suite` | string | CLI 选择的 suite，例如 `candy_21` |
| `client_timezone` | string | 客户端本地时区偏移，例如 `+08:00` |
| `os` | string | 客户端操作系统 |
| `arch` | string | 客户端架构 |
| `codex_version` | string | 本地 Codex CLI 版本 |
| `codex_model_source` | string | 模型来源：`explicit`、`codex_config` 或 `unknown` |
| `codex_model_provider` | string | Codex 配置中的 provider 名称，可能为空 |
| `codex_provider_host` | string | provider `base_url` 的 host，不含协议、路径或 query |
| `codex_provider_base_url` | string | 规范化后的 HTTPS provider base URL，保留 path，去除 query/fragment；用于区分官方渠道、中转站和未知中转站 |
| `codex_channel` | string | 服务端落库分类：`official`、`bridge`、`unknown_bridge`；上传方无需传入 |
| `codex_bridge_name` | string | 命中管理员配置的中转站映射时返回的中转站名称 |
| `codex_sandbox` | string | CLI 本次实际使用的 Codex sandbox，当前为 `read-only` |
| `codex_disabled_features` | array | 本次禁用的 Codex 功能摘要，当前包含 `memories` |
| `questions` | array | 每道题的汇总结果 |
| `attempts` | array | 每轮尝试结果，当前服务端最多保存 500 条 |
| `attempts[].answer_hash` | string | 完整回答的 SHA-256，用于去重和对比，不暴露原文 |
| `attempts[].started_at` / `attempts[].finished_at` | string | 单轮尝试开始和结束时间 |
| `attempts[].timeout_seconds` | number | 单轮 Codex 执行超时配置 |
| `attempts[].error_code` | string | 失败分类，成功时为空 |

隐私约束：

- 不上传 `full_answer`。
- 可上传 `answer_hash`，但 hash 不能反推出完整回答。
- 不上传完整 prompt，只上传 `prompt_hash`。
- `answer_preview` 用于调试和展示，服务端最多保存 300 字符。
- 可上传 `thread_id`、event type 列表、cached input tokens 等诊断摘要，但不上传原始 JSONL event。
- 不上传本地文件路径、环境变量、Codex 数据库或完整 prompt 历史。
- `anonymous` 只影响身份展示，不会隐藏模型、准确率、题目、耗时、token 等测试数据。

错误：

- `401 unauthorized`：Bearer token 无效。
- `422 validation_failed`：字段校验失败。
- `429 rate_limited`：同一用户上传过快。
- `500 internal_error`：数据库写入失败。

## GET /api/v1/submissions

查询当前用户最近上传的 submission。当前实现支持 `limit`，默认 50，最大 100。旧 `/api/runs` 和 `/api/v1/runs` 返回 `410 Gone`。

认证：Bearer token。

| 参数 | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `limit` | integer | `50` | 返回数量，最大 100 |

响应：

```json
{
  "submissions": [
    {
      "id": "submission_uuid",
      "upload_id": "upl_0123456789abcdef0123456789abcdef",
      "model": "gpt-5.5",
      "reasoning_effort": "xhigh",
      "question_count": 1,
      "attempt_count": 5,
      "correct_count": 5,
      "accuracy": 100,
      "avg_time_seconds": 2.4,
      "avg_tps": 5,
      "codex_provider_base_url": "https://api.openai.com/v1",
      "codex_channel": "official",
      "codex_bridge_id": null,
      "codex_bridge_name": "",
      "anonymous": true,
      "user": {
        "anonymous": true,
        "display_name": "匿名",
        "username": "",
        "avatar_url": "",
        "linuxdo_url": ""
      },
      "created_at": "2026-06-28T08:00:00.000Z"
    }
  ]
}
```

`user` 是展示用安全对象。公开身份时会包含 `display_name`、`username`、`avatar_url` 和 `linuxdo_url`；匿名提交时固定返回 `display_name: "匿名"`，不会返回真实用户名或头像。

错误：

- `401 unauthorized`：Bearer token 无效。

## DELETE /api/v1/submissions/{id}

删除当前用户自己的一条 benchmark submission，同时删除其 question results 和 attempts。不会删除账号、网页 session 或 CLI access token。

认证：Bearer token。

响应：

```json
{
  "ok": true,
  "deleted": 1
}
```

如果记录不存在或不属于当前用户，返回 `deleted: 0`，避免泄露其他用户数据。

## DELETE /api/v1/submissions

删除当前用户自己的全部 benchmark submission 数据。不会删除账号、网页 session 或 CLI access token。

认证：Bearer token。

响应：

```json
{
  "ok": true,
  "deleted": 12
}
```

## POST /api/logout

目标路径：`POST /api/v1/sessions/logout`。当前 Worker 已支持该 v1 alias。

撤销当前 Bearer token。CLI 当前通过 `ld-gpt-check logout` 调用该接口，并同时删除本地 token。

认证：Bearer token。当前实现即使 token 无效也返回成功。

响应：

```json
{
  "ok": true
}
```

错误：

- 目标 v1 可继续保持幂等成功，避免 logout 因 token 已过期而失败。

## 客户端行为建议

CLI 登录：

1. 调用创建设备授权接口。
2. 打开 `verification_uri_complete`。
3. 按 `interval` 调用轮询接口。
4. `pending` 时继续等待。
5. `slow_down` 时增加间隔。
6. `authorized` 时保存 token 和 user。
7. `expired` 时提示重新登录。

CLI 上传：

1. 本地生成 summary。
2. 构造 upload payload，删除 full answer。
3. 调用 `/api/v1/submissions`。
4. 展示返回的 submission id。

CLI 查询身份：

1. 读取本地 config。
2. 调用 `/api/v1/me`。
3. token 无效时提示重新登录。
