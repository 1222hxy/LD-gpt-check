# 题库配置格式

远程题库由 Worker 的公开接口提供：

```text
GET /api/v1/questions
```

当前生产地址：

```text
https://codexgo.yhklab.com/api/v1/questions
```

CLI 和向导会读取这个 JSON，并把用户选择的题目发送给模型。题库只保存题面和判题规则，不保存模型回答。

## 顶层结构

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

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | string | 否 | 当前只支持 `"1"`；建议显式填写。 |
| `questions` | array | 是 | 题目数组，数量必须是 `1..50`。 |

## 题目字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 是 | 稳定短 ID。CLI 的 `--suite` 和上传统计都使用它。 |
| `version` | string | 是 | 题目版本。题面或答案改变时递增。 |
| `title` | string | 是 | 展示名称。 |
| `prompt` | string | 是 | 发送给模型的完整题面，非空且不超过 12000 字符。 |
| `tags` | string[] | 否 | 标签数组，最多 20 个，每个不超过 64 字符。 |
| `grader` | object | 是 | 判题规则。 |

`id` 在同一个题库内不能重复。建议只使用小写字母、数字、下划线或短横线，例如 `math_001`。

## Grader 类型

### number

抽取数字并和 `expected` 比较。`expected` 必须是可解析的数字字符串。

```json
{
  "type": "number",
  "expected": "21",
  "tolerance": 0,
  "independent_match": true
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `expected` | string | 是 | 标准数字答案。 |
| `tolerance` | number | 否 | 允许误差，必须非负。默认 `0`。 |
| `independent_match` | boolean | 否 | 为 `true` 时，只要回答中出现独立的 `expected` 数字就判对。 |

适合答案唯一的数学、计数、选择题。当前经典题 `candy_21` 使用此类型。

### exact

全文匹配 `expected`。

```json
{
  "type": "exact",
  "expected": "A",
  "trim_space": true,
  "case_sensitive": false
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `expected` | string | 是 | 标准答案，不能为空。 |
| `trim_space` | boolean | 否 | 比较前去掉首尾空白。 |
| `case_sensitive` | boolean | 否 | 是否区分大小写；默认不区分。 |

适合很短、格式稳定的答案。不建议用于长文本开放题。

### regex

用正则匹配模型回答。`pattern` 必须是合法正则。

```json
{
  "type": "regex",
  "pattern": "答案[:：]\\s*(\\d+)"
}
```

如果正则包含捕获组，系统记录第一个捕获组为 extracted answer；否则记录完整匹配文本。只要匹配成功就判对。

## 推荐规范

- `prompt` 写完整题面和作答要求，不要把标准答案暴露在题面里。
- 题目语义、数据、答案或判题规则有变化时，递增 `version`。
- 同一道题不要复用旧 `id` 表示不同题目；这会污染历史统计。
- 优先使用 `number` 或短答案 `exact`，需要兼容多种表达时再使用 `regex`。
- 先用 `ld-gpt-check run --question-file ./questions.json --list-suites` 验证本地 JSON 能解析，再保存到远程题库。

