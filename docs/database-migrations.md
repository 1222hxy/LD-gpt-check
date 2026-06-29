# D1 数据库迁移指南

本文档说明 LD-gpt-check 后端 D1 数据库的迁移规范。目标是：每次后端或数据库结构有变动时，都能可靠迁移到远程 D1，避免 Worker 已部署但数据库字段、表或索引没同步。

## 核心原则

- `worker/schema.sql` 是“新建空库”的完整目标 schema。
- `worker/migrations/` 是“已有数据库升级”的增量脚本目录。
- 任何 schema 变更必须同时更新 `schema.sql` 和新增 migration 文件。
- 生产远程 D1 不要只执行 `schema.sql` 来升级。已有库升级应使用 migration。
- Worker 代码依赖新字段时，先迁移远程 D1，再部署 Worker。
- 所有远程迁移前先导出备份。
- 不要手动改生产数据，除非是在本文“补救流程”里明确列出的迁移记录修复。

## 当前项目状态

当前仓库已有：

```text
worker/schema.sql
worker/migrations/0001_add_submission_diagnostics.sql
worker/migrations/0002_add_upload_v3_and_linuxdo_profile.sql
worker/migrations/0003_add_question_banks.sql
worker/migrations/0004_add_anonymous_submissions.sql
worker/migrations/0005_add_provider_bridges.sql
worker/migrations/0006_bridge_suggestions_ai_icons.sql
```

注意：`0001` 到 `0006` 是早期手动升级脚本，假设基础表已经存在。它们不能直接用于一个全新的空库；空库初始化应执行 `schema.sql`。

从下一次 schema 变更开始，新增迁移应使用 Wrangler 的 D1 migrations 流程，让 D1 记录已应用的 migration，避免重复或漏执行。

## Wrangler 配置要求

`worker/wrangler.toml` 和 `worker/wrangler.toml.example` 的 D1 binding 应包含 `migrations_dir`：

```toml
[[d1_databases]]
binding = "DB"
database_name = "ld-gpt-check"
database_id = "YOUR_DATABASE_ID"
migrations_dir = "migrations"
```

所有命令默认在 `worker/` 目录执行：

```bash
cd worker
```

确认 Wrangler 版本和登录状态：

```bash
npx wrangler --version
npx wrangler whoami
```

## 新建空库

只在新 D1 数据库初始化时执行：

```bash
npx wrangler d1 execute ld-gpt-check --remote --file=./schema.sql
```

然后验证关键表：

```bash
npx wrangler d1 execute ld-gpt-check --remote --command="SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;"
```

至少应看到：

```text
access_tokens
benchmark_attempts
benchmark_question_results
benchmark_submissions
bridge_base_urls
bridge_suggestions
bridges
device_sessions
oauth_states
question_banks
rate_limits
users
web_sessions
```

新建空库执行 `schema.sql` 后，`0001` 到 `0006` 不需要再执行，因为这些变更已经包含在完整 schema 中。

## 现有远程库一次性基线修复

如果生产库已经手动执行过 `0001` 到 `0006`，但 Wrangler 不知道这些 migration 已应用，`npx wrangler d1 migrations list ld-gpt-check --remote` 会仍然列出 `0001` 到 `0006`。这会导致后续 `migrations apply --remote` 尝试重复执行旧脚本并失败。

先检查远程库是否已经具备当前 schema 的关键字段和表：

```bash
npx wrangler d1 execute ld-gpt-check --remote --command="PRAGMA table_info(benchmark_submissions);"
npx wrangler d1 execute ld-gpt-check --remote --command="PRAGMA table_info(bridges);"
npx wrangler d1 execute ld-gpt-check --remote --command="SELECT name FROM sqlite_master WHERE type='table' AND name IN ('question_banks','bridge_base_urls','bridge_suggestions');"
```

确认存在这些字段：

```text
benchmark_submissions.upload_schema_version
benchmark_submissions.is_anonymous
benchmark_submissions.codex_provider_base_url
benchmark_submissions.codex_channel
benchmark_submissions.codex_bridge_id
bridges.icon_url
bridges.homepage_url
```

确认无误后，备份远程库：

```bash
mkdir -p ../backups
npx wrangler d1 export ld-gpt-check --remote --output=../backups/ld-gpt-check-before-baseline.sql
```

然后只登记旧 migration 为已应用，不重新执行 SQL：

```bash
npx wrangler d1 execute ld-gpt-check --remote --command="
CREATE TABLE IF NOT EXISTS d1_migrations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT UNIQUE,
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);
INSERT OR IGNORE INTO d1_migrations (name) VALUES
  ('0001_add_submission_diagnostics.sql'),
  ('0002_add_upload_v3_and_linuxdo_profile.sql'),
  ('0003_add_question_banks.sql'),
  ('0004_add_anonymous_submissions.sql'),
  ('0005_add_provider_bridges.sql'),
  ('0006_bridge_suggestions_ai_icons.sql');
"
```

再检查是否还有 pending migration：

```bash
npx wrangler d1 migrations list ld-gpt-check --remote
```

预期不再列出 `0001` 到 `0006`。如果仍然列出，先不要继续部署，检查 `d1_migrations.name` 是否和文件名完全一致：

```bash
npx wrangler d1 execute ld-gpt-check --remote --command="SELECT id, name, applied_at FROM d1_migrations ORDER BY id;"
```

## 每次新增 schema 变更的标准流程

### 1. 创建 migration 文件

不要手写编号。使用 Wrangler 生成：

```bash
npx wrangler d1 migrations create ld-gpt-check add_some_feature
```

这会在 `worker/migrations/` 下创建新的编号文件，例如：

```text
0007_add_some_feature.sql
```

命名要求：

- 编号必须递增。
- 文件名用小写英文、数字和下划线。
- 一个 migration 只做一类变更。
- 已经应用到远程的 migration 不要修改；需要修正时新增下一号 migration。

### 2. 编写增量 SQL

常见安全写法：

```sql
ALTER TABLE benchmark_submissions ADD COLUMN new_column TEXT;
CREATE INDEX IF NOT EXISTS idx_table_column ON some_table(column_name);
CREATE TABLE IF NOT EXISTS some_table (
  id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL
);
```

注意事项：

- SQLite/D1 对 `ALTER TABLE` 能力有限。复杂变更要用“新表、复制数据、重命名”的方式。
- 给已有表加 `NOT NULL` 字段时必须有 `DEFAULT`，否则老数据无法满足约束。
- 删除字段、改字段类型、改约束属于高风险变更，先写兼容代码和回填脚本，再分多次迁移。
- 涉及数据回填时，SQL 必须可重复执行或有明确条件，避免重复写坏数据。

### 3. 同步更新完整 schema

同一次提交必须更新 `worker/schema.sql`，让新建空库也能直接得到最终结构。

检查点：

- 新表在 `schema.sql` 里存在。
- 新字段在对应 `CREATE TABLE` 里存在。
- 新索引在 `schema.sql` 里存在。
- 默认值、`NOT NULL`、外键和 migration 一致。

### 4. 本地演练 migration

先用本地 D1 验证 migration 可执行：

```bash
npx wrangler d1 migrations list ld-gpt-check --local
npx wrangler d1 migrations apply ld-gpt-check --local
```

如果本地库是空库且旧 `0001` 到 `0006` 没有做过基线，会失败。这种情况下用临时本地库先导入完整 schema，再做基线记录，然后测试新 migration：

```bash
TMP_D1="$(mktemp -d)"
npx wrangler d1 execute ld-gpt-check --local --persist-to "$TMP_D1" --file=./schema.sql
npx wrangler d1 execute ld-gpt-check --local --persist-to "$TMP_D1" --command="
CREATE TABLE IF NOT EXISTS d1_migrations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT UNIQUE,
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);
INSERT OR IGNORE INTO d1_migrations (name) VALUES
  ('0001_add_submission_diagnostics.sql'),
  ('0002_add_upload_v3_and_linuxdo_profile.sql'),
  ('0003_add_question_banks.sql'),
  ('0004_add_anonymous_submissions.sql'),
  ('0005_add_provider_bridges.sql'),
  ('0006_bridge_suggestions_ai_icons.sql');
"
npx wrangler d1 migrations list ld-gpt-check --local --persist-to "$TMP_D1"
npx wrangler d1 migrations apply ld-gpt-check --local --persist-to "$TMP_D1"
```

本地 apply 成功后，检查目标结构：

```bash
npx wrangler d1 execute ld-gpt-check --local --persist-to "$TMP_D1" --command="PRAGMA table_info(YOUR_TABLE);"
npx wrangler d1 execute ld-gpt-check --local --persist-to "$TMP_D1" --command="SELECT name FROM sqlite_master WHERE type='index' ORDER BY name;"
```

### 5. 远程迁移前备份

每次远程迁移前都导出备份：

```bash
mkdir -p ../backups
npx wrangler d1 export ld-gpt-check --remote --output=../backups/ld-gpt-check-$(date +%Y%m%d-%H%M%S).sql
```

备份文件不要提交到 Git。

### 6. 查看远程 pending migrations

```bash
npx wrangler d1 migrations list ld-gpt-check --remote
```

确认列表里只出现你本次新增的 migration。若看到旧的 `0001` 到 `0006`，先做“现有远程库一次性基线修复”。

### 7. 应用远程 migration

```bash
npx wrangler d1 migrations apply ld-gpt-check --remote
```

Wrangler 会列出即将执行的 migration。确认编号和文件名正确后继续。

如果某个 migration 失败，Wrangler 会回滚该 migration；不要立刻部署 Worker。先看错误，修复方式通常是新增下一号 migration 或修正尚未成功应用的 migration 文件。

### 8. 远程验证

迁移后至少执行：

```bash
npx wrangler d1 migrations list ld-gpt-check --remote
npx wrangler d1 execute ld-gpt-check --remote --command="SELECT id, name, applied_at FROM d1_migrations ORDER BY id DESC LIMIT 10;"
npx wrangler d1 execute ld-gpt-check --remote --command="SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;"
```

针对具体改动再查表结构：

```bash
npx wrangler d1 execute ld-gpt-check --remote --command="PRAGMA table_info(YOUR_TABLE);"
npx wrangler d1 execute ld-gpt-check --remote --command="SELECT name, sql FROM sqlite_master WHERE type='index' AND tbl_name='YOUR_TABLE';"
```

### 9. 部署 Worker

只有远程 D1 验证通过后，才部署依赖新 schema 的 Worker：

```bash
cd ../frontend
npm run build

cd ../worker
npx wrangler deploy
```

部署后验证：

```bash
curl https://YOUR_WORKER_DOMAIN/health
```

再验证关键业务路径：

- `GET /api/v1/questions`
- CLI 登录或 `whoami`
- CLI 上传一次测试结果
- Dashboard 或 admin 页面读取新字段

## 变更类型与推荐策略

### 只新增可空字段

推荐：

```sql
ALTER TABLE benchmark_submissions ADD COLUMN some_value TEXT;
```

部署顺序：

1. 迁移。
2. 部署写入新字段的 Worker。
3. 旧客户端仍可继续工作。

### 新增非空字段

必须带默认值：

```sql
ALTER TABLE benchmark_submissions ADD COLUMN source TEXT NOT NULL DEFAULT '';
```

否则已有行无法满足 `NOT NULL`。

### 新增表

使用 `CREATE TABLE IF NOT EXISTS`，并同步创建索引：

```sql
CREATE TABLE IF NOT EXISTS example_table (
  id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_example_table_created ON example_table(created_at);
```

### 重命名或删除字段

不要一步到位。推荐三阶段：

1. 新增新字段，Worker 同时兼容新旧字段。
2. 回填数据，观察一段时间。
3. 新增迁移重建表并清理旧字段。

SQLite 重建表示例必须谨慎编写：

```sql
CREATE TABLE new_table (...);
INSERT INTO new_table (...) SELECT ... FROM old_table;
DROP TABLE old_table;
ALTER TABLE new_table RENAME TO old_table;
CREATE INDEX IF NOT EXISTS ...;
```

这类迁移必须先在备份或本地复制库上演练。

## 排障

### Worker 报 `no such column`

原因：Worker 已部署，但远程 D1 没有对应字段。

处理：

```bash
cd worker
npx wrangler d1 migrations list ld-gpt-check --remote
npx wrangler d1 migrations apply ld-gpt-check --remote
npx wrangler d1 execute ld-gpt-check --remote --command="PRAGMA table_info(TABLE_NAME);"
```

确认字段存在后再重新测试业务接口。

### `migrations apply` 想执行旧的 0001-0006

原因：旧库手动执行过 SQL，但 `d1_migrations` 没记录。

处理：执行本文“现有远程库一次性基线修复”。不要直接 apply，否则会重复 `ALTER TABLE ADD COLUMN` 或因基础表不存在失败。

### `duplicate column name`

原因：字段已经存在，但 migration 记录缺失，或 migration 被重复执行。

处理：

1. 用 `PRAGMA table_info(TABLE_NAME);` 确认字段存在。
2. 用 `SELECT * FROM d1_migrations;` 确认记录缺失。
3. 如果字段和表结构完全符合预期，只补 `d1_migrations` 记录。
4. 如果结构不一致，先备份，再写新的修复 migration。

### `no such table`

原因：

- 空库直接执行了旧的 `0001` 到 `0006`。
- `worker/wrangler.toml` 绑定到了错误的 D1 database id。
- 新建库未执行 `schema.sql`。

处理：

```bash
npx wrangler d1 execute ld-gpt-check --remote --command="SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;"
```

如果是空库，执行 `schema.sql`。如果表存在但不是预期库，检查 `worker/wrangler.toml` 的 `database_id`。

### 本地成功，远程失败

常见原因：

- 远程有历史数据违反新约束。
- 远程 migration 记录和本地不一致。
- 本地使用的是空库，没覆盖真实数据形态。

处理：

1. 导出远程备份。
2. 在临时本地库导入备份。
3. 对临时库执行 migration。
4. 修正 SQL 后再远程 apply。

```bash
npx wrangler d1 export ld-gpt-check --remote --output=../backups/prod.sql
```

## PR / 提交检查清单

每次数据库相关 PR 必须确认：

- [ ] 新增或修改了 `worker/migrations/NNNN_description.sql`。
- [ ] `worker/schema.sql` 已同步最终结构。
- [ ] `worker/wrangler.toml.example` 的 D1 配置仍正确。
- [ ] 本地执行过 `npx wrangler d1 migrations apply ld-gpt-check --local`，或使用临时基线库演练过。
- [ ] 远程执行前已导出 D1 备份。
- [ ] 远程执行后 `migrations list --remote` 没有 pending migration。
- [ ] 远程 `PRAGMA table_info(...)` 验证过关键字段。
- [ ] 依赖新 schema 的 Worker 是在远程迁移成功后部署的。
- [ ] 如果涉及上传、Dashboard、Admin，至少手动验证一个对应业务路径。

## 命令速查

```bash
cd worker

# 创建迁移
npx wrangler d1 migrations create ld-gpt-check add_feature

# 查看本地/远程待执行迁移
npx wrangler d1 migrations list ld-gpt-check --local
npx wrangler d1 migrations list ld-gpt-check --remote

# 执行本地/远程迁移
npx wrangler d1 migrations apply ld-gpt-check --local
npx wrangler d1 migrations apply ld-gpt-check --remote

# 执行 SQL
npx wrangler d1 execute ld-gpt-check --remote --command="SELECT 1;"
npx wrangler d1 execute ld-gpt-check --remote --file=./schema.sql

# 备份
npx wrangler d1 export ld-gpt-check --remote --output=../backups/ld-gpt-check.sql

# 查看迁移记录
npx wrangler d1 execute ld-gpt-check --remote --command="SELECT id, name, applied_at FROM d1_migrations ORDER BY id;"

# 查看表结构
npx wrangler d1 execute ld-gpt-check --remote --command="PRAGMA table_info(benchmark_submissions);"
```
