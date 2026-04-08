# Schema

本库使用一个极简的 `SeedRecord` 作为唯一核心结构。

每一行都存放在 [xhs_seeds.jsonl](/Users/zeena/GitHub/AgentNetwork-Red/bd/xhs_seeds.jsonl) 中，格式为一行一个 JSON 对象。

## `SeedRecord`

| 字段 | 必填 | 说明 |
|------|------|------|
| `seed_id` | 是 | 格式固定为 `seed-YYYYMMDD-NNN` |
| `xhs_profile_url` | 是 | 小红书主页 URL，校验后按规范化结果去重 |
| `cover_photo` | 是 | 仓库内相对路径，当前必须指向 `bd/photos/` 下文件 |
| `source_event` | 是 | 这条线索来自哪个线下场景 |
| `captured_at` | 是 | RFC3339 UTC |
| `collector` | 否 | 缺省时按 `Zeena` |
| `display_name` | 否 | 这个人是谁 |
| `team_name` | 否 | 他们组的名字 |
| `team_scope` | 否 | 他们组在做什么 |
| `note` | 否 | 其他补充说明 |

最小合法样例：

```json
{
  "seed_id": "seed-20260408-001",
  "xhs_profile_url": "https://www.xiaohongshu.com/user/profile/sample_linwei",
  "cover_photo": "bd/photos/seed-20260408-001-cover.svg",
  "source_event": "shanghai-demo-day",
  "captured_at": "2026-04-08T12:05:00Z"
}
```

## URL 规范化规则

去重时不直接用原始 URL，而是先做规范化：

1. 必须是 `https://`
2. host 一律转小写
3. 去掉 query
4. 去掉 fragment
5. 去掉末尾 `/`

例如以下两条会被视为同一个种子：

- `https://www.xiaohongshu.com/user/profile/abc/`
- `https://www.xiaohongshu.com/user/profile/abc?x=1`

## 去重规则

1. 以规范化后的 `xhs_profile_url` 作为唯一主键来源。
2. 不再按姓名、团队或照片做主键。
3. 如果 URL 重复，必须合并或跳过，不能新增第二条种子。

## 封面图规则

- 每条种子必须有一张封面图
- 当前只要求 1 张，不做多图相册
- `cover_photo` 必须指向仓库中的 `bd/photos/` 文件
- 推荐格式：`jpg`、`jpeg`、`png`、`webp`
- 示例和占位允许使用 `svg`

## 命名规则

- 种子 ID：`seed-YYYYMMDD-NNN`
- 图片文件：`seed-YYYYMMDD-NNN-cover.<ext>`
- `source_event` 推荐使用稳定短名，例如 `shanghai-demo-day`

## 不做的事

- 不在 GitHub 种子文件里保存复杂交互历史
- 不在 GitHub 里直接维护抓取后的标准化 creator/post 表
- 不要求每条记录都有姓名、团队或备注
