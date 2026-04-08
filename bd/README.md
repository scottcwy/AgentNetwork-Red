# 小红书种子库（BD）

这一层现在是一个极简的“小红书种子库”，不是完整的人物档案系统。

它只负责保存两类内容：

- 可供后续开发者抓取的小红书主页 URL
- 用于 wrap-up display 的封面图和少量补充说明

## 最小规则

- 权威数据源只有一个：[xhs_seeds.jsonl](/Users/zeena/GitHub/AgentNetwork-Red/bd/xhs_seeds.jsonl)
- 每条种子必须有 `xhs_profile_url` 和 `cover_photo`
- `collector` 可省略；省略时默认按 `Zeena`
- `display_name`、`team_name`、`team_scope`、`note` 都是可选加分项
- 一条种子只要求一张封面图；这期不做多图相册

## 快速开始

1. 在 [xhs_seeds.jsonl](/Users/zeena/GitHub/AgentNetwork-Red/bd/xhs_seeds.jsonl) 里追加一行 JSON。
2. 把对应封面图放进 `bd/photos/`。
3. 运行校验：

```bash
go run ./cmd/bdcheck ./bd
```

4. 如果你先把飞书提交整理进本地 `.bd-inbox/submissions.jsonl`，运行同步：

```bash
go run ./cmd/bdsync --inbox ./.bd-inbox ./bd
```

5. 生成 wrap-up 页面：

```bash
go run ./cmd/bdwrap ./bd
```

## 文档入口

- [schema.md](/Users/zeena/GitHub/AgentNetwork-Red/bd/schema.md): `SeedRecord` 字段、URL 规范化和去重规则
- [sop.md](/Users/zeena/GitHub/AgentNetwork-Red/bd/sop.md): 线下最小采集规范
- [lark-bot-input.md](/Users/zeena/GitHub/AgentNetwork-Red/bd/lark-bot-input.md): 飞书里你实际怎么发
- [inbox.md](/Users/zeena/GitHub/AgentNetwork-Red/bd/inbox.md): 本地 inbox 结构与状态
- [db-schema.md](/Users/zeena/GitHub/AgentNetwork-Red/bd/db-schema.md): raw-first DB 设计说明
- [db-schema.sql](/Users/zeena/GitHub/AgentNetwork-Red/bd/db-schema.sql): SQLite 建表 SQL

## 目录结构

```text
bd/
├── db-schema.md
├── db-schema.sql
├── display/
│   └── index.html
├── inbox.md
├── lark-bot-input.md
├── photos/
├── schema.md
├── sop.md
└── xhs_seeds.jsonl
```
