# LD-gpt-check API 设计说明

本文档定义 LD-gpt-check 服务端 API 的目标设计。当前 Worker 保留设备登录 legacy 路径，benchmark 上传使用 `/api/v1/submissions`。

相关文档：

- [API Reference](./api-reference.md)
- [OpenAPI contract](./openapi.yaml)

## 设计目标

API 服务承担三类职责：

1. 为 CLI 提供设备授权登录。
2. 接收 CLI 上传的测试结果。
3. 为已登录用户查询自己的账号和历史结果。

设计原则：

- **CLI 优先**：请求和响应应稳定、短小、易解析；错误信息要能直接展示给终端用户。
- **版本化演进**：公共 JSON API 使用 `/api/v1` 前缀；浏览器 OAuth 页面可以继续使用非 API 路由。
- **隐私默认收敛**：上传只接受统计指标和 answer preview，不接受完整回答、prompt 历史或本地 Codex 数据库。
- **兼容可迁移**：设备登录 legacy 端点保留；旧 `/api/runs` 已废弃并返回 `410 Gone`。
- **资源化命名**：路径表达资源和动作边界，避免把实现细节暴露为路径名。
- **默认防护**：状态变更接口要有输入大小限制、同源校验、限流和不泄露内部异常的错误处理。

## 当前实现与目标契约

当前 Worker 入口在 `worker/src/index.ts`，Go 客户端在 `internal/api/client.go`。

| 功能 | 当前路径 | 目标 v1 路径 |
| --- | --- | --- |
| 健康检查 | `GET /health` | `GET /health` |
| 创建设备登录 | `POST /api/device/start` | `POST /api/v1/device-authorizations` |
| 轮询设备登录 | `POST /api/device/poll` | `POST /api/v1/device-authorizations/token` |
| 授权页面 | `GET /device` | `GET /device` |
| 提交页面授权 | `POST /api/device/approve` | `POST /api/v1/device-authorizations/approve` 或保留 HTML 表单路由 |
| Linux.do OAuth 开始 | `GET /auth/linuxdo/start` | `GET /auth/linuxdo/start` |
| Linux.do OAuth 回调 | `GET /auth/linuxdo/callback` | `GET /auth/linuxdo/callback` |
| 当前用户 | `GET /api/me` | `GET /api/v1/me` |
| 上传测试结果 | 已废弃，返回 410 | `POST /api/v1/submissions` |
| 查询测试结果 | 已废弃，返回 410 | `GET /api/v1/submissions` |
| 退出登录 | `POST /api/logout` | `POST /api/v1/sessions/logout` |

迁移建议：

1. Go 客户端上传已切到 `/api/v1/submissions`。
2. legacy 设备登录端点继续可用，便于 CLI 登录。
3. README 和部署文档使用 v1 submission 示例。
4. 如需删除更多 legacy 路由，先在 release note 中明确说明。

## 认证模型

### CLI Bearer Token

CLI 通过设备授权流程获得平台 access token。后续受保护端点使用：

```http
Authorization: Bearer ldgc_...
```

服务端只存储 token hash，不存储明文 token。token 可通过 logout 撤销。

受保护端点：

- `GET /api/v1/me`
- `POST /api/v1/submissions`
- `GET /api/v1/submissions`
- `POST /api/v1/sessions/logout`

### Web Session Cookie

浏览器授权页面使用 `ldgc_session` HttpOnly cookie。该 cookie 仅用于网页授权流程，不作为公共 JSON API 的认证方式。

Cookie 要求：

- `HttpOnly`
- `SameSite=Lax`
- HTTPS 环境启用 `Secure`
- 设置明确过期时间

### OAuth State

Linux.do OAuth 使用一次性 `state`。服务端应保存 state hash、回跳路径、过期时间和 used 标记。回调时必须同时校验 query state 和 cookie state。

## 响应格式

成功响应默认使用 JSON object。除非端点天然是 HTML 页面或 302 重定向，不返回裸数组或纯文本。

示例：

```json
{
  "user": {
    "id": "usr_123",
    "username": "alice"
  }
}
```

当前实现为了兼容旧 Go CLI，错误响应保留顶层 `error` 字符串，同时提供稳定机器码和 request id：

```json
{
  "error": "unauthorized",
  "code": "unauthorized",
  "request_id": "req_01J..."
}
```

字段说明：

- `error`：可读错误信息，可直接展示给 CLI 用户。
- `code`：稳定机器码，使用 `snake_case`。
- `request_id`：请求追踪 ID；优先使用 Cloudflare `cf-ray`，否则由 Worker 生成。

建议错误码：

| HTTP 状态码 | code | 场景 |
| --- | --- | --- |
| 400 | `bad_request` | JSON 无效、字段类型错误、验证码无效 |
| 401 | `unauthorized` | 缺少 token、token 无效或已撤销 |
| 404 | `not_found` | 路径或资源不存在 |
| 409 | `conflict` | 幂等键冲突或状态冲突 |
| 422 | `validation_failed` | 请求格式正确但业务字段不合法 |
| 429 | `rate_limited` | 轮询太快或触发限流 |
| 500 | `internal_error` | 未预期服务端错误 |
| 502 | `upstream_error` | OAuth token 或 userinfo 上游失败 |

## 资源模型

### User

表示 Linux.do 账号在本服务中的用户记录。

```json
{
  "id": "6f1a...",
  "username": "alice"
}
```

公开 API 不暴露 provider access token、provider user id、session hash 或 token hash。

### Device Authorization

表示 CLI 发起的一次设备登录。

状态：

- `pending`：等待用户在浏览器授权。
- `slow_down`：CLI 轮询过快，应增加间隔。
- `authorized`：授权完成，响应包含 access token。
- `expired`：授权码过期或已消费。

目标响应：

```json
{
  "device_code": "dc_xxx",
  "user_code": "123-456-789",
  "verification_uri": "https://api.example.com/device",
  "verification_uri_complete": "https://api.example.com/device?code=123-456-789",
  "expires_in": 600,
  "interval": 3
}
```

### Submission

表示一次 CLI benchmark 上传。一次 submission 可以包含多道题，每道题可以运行多次。

核心字段：

- `client_version`
- `upload_id`
- `model`
- `reasoning_effort`
- `question_count`
- `attempt_count`
- `correct`
- `accuracy`
- `avg_input_tokens`
- `avg_output_tokens`
- `avg_reason_tokens`
- `avg_time_seconds`
- `avg_tps`
- `os`
- `arch`
- `codex_version`
- `questions`
- `attempts`

隐私要求：

- `attempts[].answer_preview` 最大 300 字符。
- `attempts[].full_answer` 和完整 prompt 不得上传。
- `questions[].prompt_hash` 可用于题目版本排查，不保存 prompt 正文。
- 服务端限制 `questions` 和 `attempts` 数量。

## 分页、过滤和排序

当前 `GET /api/v1/submissions` 返回最近 submission，默认 50 条。后续可增加 cursor：

```http
GET /api/v1/submissions?limit=50&cursor=opaque_cursor
```

响应：

```json
{
  "submissions": [],
  "next_cursor": "opaque_cursor_or_null"
}
```

规则：

- `limit` 默认 50，最大 100。
- `cursor` 必须是不透明字符串，客户端不解析。
- 默认排序为 `created_at DESC`。
- 未来过滤条件使用 query 参数，例如 `model`、`reasoning_effort`、`from`、`to`。

## 幂等性

当前上传使用 `upload_id` 做幂等键：

- 仅对 `POST /api/v1/submissions` 生效。
- 同一用户、同一 `upload_id` 重复提交时返回同一个 submission id，并带 `duplicate: true`。
- CLI 每次上传前生成新的随机 `upload_id`。

## 验证规则

服务端应做输入收敛，不能依赖客户端永远发送正确数据。

建议规则：

- `question_count`：整数，范围 `1..50`。
- `attempt_count`：整数，范围 `1..500`。
- `correct`：整数，范围 `0..attempt_count`。
- `accuracy`：数字，范围 `0..100`，与 CLI 当前百分比输出一致。
- token 指标和耗时指标：非负数字。
- `model`、`reasoning_effort`、`os`、`arch`、`codex_version`：字符串长度上限 128。
- `questions`：长度等于 `question_count`。
- `attempts`：长度等于 `attempt_count`，最多 500 条。
- `attempts[].answer_preview`：最多 300 字符。

当前实现会把非法数字转为 0。v1 实现应优先返回 `422 validation_failed`，这样 CLI 可以及时发现数据结构错误。

## 限流建议

设备登录轮询已有 `interval`。目标行为：

- CLI 必须按 `interval` 秒轮询。
- 服务端发现过快轮询时返回 `429 rate_limited` 或业务状态 `slow_down`。
- 收到 `slow_down` 后 CLI 应增加轮询间隔，最大 30 秒。
- 上传 submission 可按 token 做轻量限流，防止脚本误刷。
- 浏览器设备授权、OAuth start/callback 和 submission 上传均应有服务端限流。
- 限流命中返回 `429 rate_limited`，并保留 `x-request-id` 便于排查。

## 安全与隐私

- 所有生产环境必须使用 HTTPS。
- token、session、OAuth state 只存 hash。
- 不在日志中记录 access token、device code、session cookie。
- OAuth 回跳路径必须限制为站内路径，避免开放重定向。
- 错误信息不能泄露 hash、SQL、secret 或上游 token。
- Worker secrets 使用 Wrangler secrets，不写入仓库。
- 浏览器授权提交必须校验同源请求，避免 CSRF。
- 生产环境建议启用 Cloudflare Turnstile 保护 `/device` 授权页。
- CORS 使用白名单，不使用 `*` 允许带凭据或授权头的浏览器调用。

## 扩展约定

后续新增接口时遵守以下约定：

- 新公共 JSON API 放在 `/api/v1`。
- 新字段默认向后兼容；删除或重命名字段必须升版本。
- 响应 object 可以新增字段，客户端必须忽略未知字段。
- 枚举新增值需要先更新文档和客户端容错逻辑。
- 新资源优先使用复数名词，例如 `/submissions`、`/tokens`。
- 非资源动作放在子路径末尾，例如 `/sessions/logout`。

## 实施优先级

建议按以下顺序落地：

1. 为 submission 上传保持显式校验。
2. 为 `GET /api/v1/submissions` 增加 cursor 分页。
3. 后续需要公开榜单时新增独立只读资源，不复用上传接口。
4. legacy 路由保留并记录废弃计划。
