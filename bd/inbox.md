# 本地 Inbox 结构

飞书 bot 不直接写 GitHub 权威文件，而是先把结果整理到本地 `.bd-inbox/`。

## 目录

```text
.bd-inbox/
├── drafts/
│   └── <chat_id>.json
├── files/
│   └── <downloaded-cover-photo>
└── submissions.jsonl
```

`.bd-inbox/` 应该只在本地存在，不提交到仓库。

## drafts

`drafts/<chat_id>.json` 用来保存一个私聊里当前还没 `done` 的草稿。

最小结构：

```json
{
  "chat_id": "oc_xxx",
  "collector": "Zeena",
  "source_event": "shanghai-demo-day",
  "xhs_profile_url": "https://www.xiaohongshu.com/user/profile/sample_linwei",
  "display_name": "林未",
  "team_name": "虾网探针组",
  "team_scope": "做线下参与者发现与资料沉淀",
  "note": "愿意后续配合数据采集",
  "pending_photo": ".bd-inbox/files/om_xxx.jpg",
  "updated_at": "2026-04-08T12:30:00Z"
}
```

## submissions

只有 `done` 成功后，才追加一条 finalized submission 到 `.bd-inbox/submissions.jsonl`。

每行一个 JSON：

```json
{
  "submission_id": "sub-20260408-001",
  "chat_id": "oc_xxx",
  "collector": "Zeena",
  "source_event": "shanghai-demo-day",
  "xhs_profile_url": "https://www.xiaohongshu.com/user/profile/sample_linwei",
  "display_name": "林未",
  "team_name": "虾网探针组",
  "team_scope": "做线下参与者发现与资料沉淀",
  "note": "愿意后续配合数据采集",
  "cover_photo_tmp": ".bd-inbox/files/sub-20260408-001.jpg",
  "submitted_at": "2026-04-08T12:31:00Z",
  "state": "ready"
}
```

## state

- `ready`：等待同步
- `synced`：已经同步进 `bd/xhs_seeds.jsonl`
- `needs_review`：与已有 seed 的可选字段冲突，需要人工看
- `error`：缺字段、图片不存在或 URL 非法
