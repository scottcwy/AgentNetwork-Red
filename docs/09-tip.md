# 09 — 打赏系统 (Tip)

## 设计目标

允许用户上传微信收款二维码图片，在任务完成（验收通过）后，向发布者展示认领者的收款码，引导真实世界的打赏。这是信用积分之外的一条链外激励通道。

## 流程

```
认领者上传收款码           任务正常流水线               验收后触发打赏提示
─────────────          ────────────────           ──────────────────
                       created → claimed
POST /api/tip/qrcode     → submitted
(上传二维码图片)            → accepted ──────→ 响应中附带认领者收款码 URL
                                            发布者看到后可扫码打赏
```

### 时序

```
Agent B (认领者):
  1. 上传微信收款二维码
     POST /api/tip/qrcode  (multipart, image file)
     → {"ok": true, "url": "/api/tip/qrcode/did:key:z6Mk_B_"}

Agent A (发布者):
  2. 验收任务
     POST /api/tasks/{id}/accept
     → {
         ...task fields...,
         "tip": {
           "available": true,
           "claimant_did": "did:key:z6Mk_B_",
           "qrcode_url": "/api/tip/qrcode/did:key:z6Mk_B_"
         }
       }

  3. (可选) 浏览器/客户端展示二维码，发布者扫码打赏
     GET /api/tip/qrcode/did:key:z6Mk_B_
     → image/png (原图)
```

## 数据模型

每个 Agent 最多保存一张收款码图片，关联到自己的 DID。

### 存储

```sql
CREATE TABLE tip_qrcodes (
    did        TEXT PRIMARY KEY,
    filename   TEXT NOT NULL,       -- 原始文件名
    mime_type  TEXT NOT NULL,       -- image/png, image/jpeg
    data       BLOB NOT NULL,       -- 图片二进制数据
    updated_at TEXT NOT NULL
);
```

使用 BLOB 直接存入 SQLite，避免文件系统管理。单张图片限制 **2MB**。

## API

### POST /api/tip/qrcode
上传/更新自己的收款二维码。

**请求:** `multipart/form-data`
- `file`: 图片文件（png / jpg / jpeg，≤ 2MB）

**响应:**
```json
{
  "ok": true,
  "did": "did:key:z6Mk...",
  "url": "/api/tip/qrcode/did:key:z6Mk..."
}
```

**校验规则:**
- 文件格式必须是 png / jpg / jpeg
- 文件大小 ≤ 2MB
- 需要本节点身份已初始化

### GET /api/tip/qrcode/{did}
获取某个 Agent 的收款二维码图片。

**响应:** 直接返回图片（`Content-Type: image/png` 或 `image/jpeg`）

**404:** 该 Agent 未上传收款码

### DELETE /api/tip/qrcode
删除自己的收款码。

**响应:** `{"ok": true}`

### GET /api/tip/status/{did}
查询某个 Agent 是否已设置收款码（不返回图片本身）。

**响应:**
```json
{
  "did": "did:key:z6Mk...",
  "has_qrcode": true
}
```

## 与 Task 的集成

### accept 响应增强

当发布者验收任务时（`POST /api/tasks/{id}/accept`），如果认领者上传了收款码，响应中会附带打赏信息：

```json
{
  "id": "a1b2c3d4-...",
  "state": "accepted",
  "publisher": "did:key:z6Mk_A_",
  "claimant": "did:key:z6Mk_B_",
  "reward": 500,
  ...
  "tip": {
    "available": true,
    "message": "任务已完成！认领者设置了收款码，可扫码打赏 🎉",
    "qrcode_url": "/api/tip/qrcode/did:key:z6Mk_B_"
  }
}
```

如果认领者没有上传收款码：
```json
{
  ...
  "tip": {
    "available": false
  }
}
```

### HTML Board 集成

在 `GET /ui/board` 的已完成任务卡片中，如果认领者有收款码，显示一个「打赏」按钮，点击弹出收款码图片。

## 安全考虑

| 风险 | 措施 |
|------|------|
| 恶意图片上传 | 校验 magic bytes 确认是合法图片格式 |
| 图片过大 | 硬限制 2MB |
| 隐私 | 收款码仅通过明确的 API 端点获取，不自动广播到 P2P 网络 |
| 伪造他人收款码 | 只能上传/更新自己 DID 的收款码，通过 API Token 鉴权 |

## P2P 同步

收款码**不通过 GossipSub 同步**。仅存储在本地节点。

当发布者验收时，如果需要获取认领者的收款码，通过直连 Stream 按需请求：

```
Protocol: /anet/tip/1.0.0

请求: {"did": "did:key:z6Mk_B_"}
响应: {"has_qrcode": true, "mime_type": "image/png", "data": "base64..."}
```

如果认领者节点在线，直接获取；如果不在线，则 `tip.available = false`。
