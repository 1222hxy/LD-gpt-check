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
    "username": "alice"
  }
}
```

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
  "upload_schema_version": 2,
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
  "os": "linux",
  "arch": "amd64",
  "codex_version": "codex 0.1.0",
  "codex_model_source": "explicit",
  "codex_model_provider": "openai",
  "codex_provider_host": "api.openai.com",
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
      "answer_chars": 2
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
| `upload_schema_version` | integer | 上传 payload 版本；当前 Go CLI 使用 `2` |
| `question_count` | integer | 本次包含的问题数量 |
| `attempt_count` | integer | 本次总尝试次数 |
| `correct` | integer | 正确尝试次数 |
| `accuracy` | number | 正确率百分比，范围 `0..100` |
| `avg_input_tokens` | number | 平均输入 tokens |
| `avg_output_tokens` | number | 平均输出 tokens |
| `avg_reason_tokens` | number | 平均 reasoning tokens |
| `avg_time_seconds` | number | 平均耗时秒数 |
| `avg_tps` | number | 平均 tokens per second |
| `os` | string | 客户端操作系统 |
| `arch` | string | 客户端架构 |
| `codex_version` | string | 本地 Codex CLI 版本 |
| `codex_model_source` | string | 模型来源：`explicit`、`codex_config` 或 `unknown` |
| `codex_model_provider` | string | Codex 配置中的 provider 名称，可能为空 |
| `codex_provider_host` | string | provider `base_url` 的 host，不含协议、路径或 query |
| `codex_sandbox` | string | CLI 本次实际使用的 Codex sandbox，当前为 `read-only` |
| `codex_disabled_features` | array | 本次禁用的 Codex 功能摘要，当前包含 `memories` |
| `questions` | array | 每道题的汇总结果 |
| `attempts` | array | 每轮尝试结果，当前服务端最多保存 500 条 |

隐私约束：

- 不上传 `full_answer`。
- 不上传完整 prompt，只上传 `prompt_hash`。
- `answer_preview` 用于调试和展示，服务端最多保存 300 字符。
- 可上传 `thread_id`、event type 列表、cached input tokens 等诊断摘要，但不上传原始 JSONL event。
- 不上传本地文件路径、环境变量、Codex 数据库或完整 prompt 历史。

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
      "created_at": "2026-06-28T08:00:00.000Z"
    }
  ]
}
```

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
