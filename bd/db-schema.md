# Raw-First DB Schema

这期数据库目标不是直接做标准化的小红书业务库，而是先把“种子”和“抓取快照”稳定落住。

## 设计原则

- GitHub 里的 `xhs_seeds.jsonl` 是人工维护的入口
- DB 先承接 raw-first 数据，不要求爬虫一次性写出标准化 creator/post 表
- 先保证“能抓、能存、能回放”，再做结构化抽取

## 表 1: `xhs_seed_accounts`

一条 GitHub 种子对应一条 seed account 记录。

核心字段：

- `seed_id`
- `original_profile_url`
- `normalized_profile_url`
- `cover_photo`
- `source_event`
- `captured_at`
- `collector`
- `display_name`
- `team_name`
- `team_scope`
- `note`

## 表 2: `xhs_crawl_snapshots`

一条抓取任务可以产生多次 snapshot。

核心字段：

- `snapshot_id`
- `seed_id`
- `fetched_at`
- `crawler`
- `payload_json`
- `parse_status`
- `source_hash`

## 推荐流程

1. 线下同学把链接和封面图写进 `xhs_seeds.jsonl`
2. 其他开发者把种子同步进 `xhs_seed_accounts`
3. 爬虫抓取主页或相关公开资料
4. 原始结果先按 `payload_json` 落进 `xhs_crawl_snapshots`
5. 真正需要做 creator/post 标准化时，再从 snapshot 派生新的表

## 为什么先 raw-first

- 更少前置决策
- 对爬虫变更更宽容
- 不会因为标准字段还没想清楚就卡住采集

SQL 见 [db-schema.sql](/Users/zeena/GitHub/AgentNetwork-Red/bd/db-schema.sql)。
