CREATE TABLE IF NOT EXISTS question_banks (
  id TEXT PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  schema_version TEXT NOT NULL,
  questions_json TEXT NOT NULL,
  is_active INTEGER NOT NULL DEFAULT 1,
  created_by TEXT,
  updated_by TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY(created_by) REFERENCES users(id),
  FOREIGN KEY(updated_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_question_banks_active ON question_banks(is_active, updated_at);

INSERT INTO question_banks
  (id, slug, title, schema_version, questions_json, is_active, created_at, updated_at)
VALUES
  (
    'default-question-bank',
    'default',
    'Default question bank',
    '1',
    '{
  "schema_version": "1",
  "questions": [
    {
      "id": "candy_21",
      "version": "1",
      "title": "糖果形状口味保证题",
      "prompt": "不使用任何外部工具回答以下问题：\n\n在一个黑色的袋子里放有三种口味的糖果，每种糖果有两种不同的形状（圆形和五角星形，不同的形状靠手感可以分辨）。现已知不同口味的糖和不同形状的数量统计如下表。参赛者需要在活动前决定摸出的糖果数目，那么，最少取出多少个糖果才能保证手中同时拥有不同形状的苹果味和桃子味的糖？（同时手中有圆形苹果味匹配五角星桃子味糖果，或者有圆形桃子味匹配五角星苹果味糖果都满足要求）\n\n          苹果味 桃子味 西瓜味\n圆形        7      9      8\n五角星形    7      6      4",
      "tags": ["math", "pigeonhole"],
      "grader": {
        "type": "number",
        "expected": "21",
        "independent_match": true
      }
    }
  ]
}',
    1,
    '2026-06-28T00:00:00.000Z',
    '2026-06-28T00:00:00.000Z'
  )
ON CONFLICT(slug) DO NOTHING;
