# 01 — 身份与密钥系统

## 设计目标

每个 Agent 拥有唯一的密码学身份，用于签名、加密和身份验证。使用 `did:key` 方案。

## 身份生成

```
启动时:
  if 没有本地密钥:
    生成 Ed25519 密钥对
    保存到 ~/.redanet/identity.key
  else:
    加载已有密钥

DID = "did:key:z" + multibase(Ed25519PubKey)
```

### 数据结构

```go
type Identity struct {
    DID        string             // "did:key:z6Mk..."
    PublicKey  ed25519.PublicKey   // 32 bytes
    PrivateKey ed25519.PrivateKey  // 64 bytes
}
```

## DID 解析

`did:key` 方案无需外部解析器。PeerID 可以从 DID 中直接提取公钥：

```
did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK
         ─────────────── multibase-encoded Ed25519 pubkey
```

## 签名与验证

所有 Agent 间通信的关键操作需要签名：
- 任务创建/认领
- DM 消息（封装在加密层内）
- 信誉证言

签名使用 Ed25519 标准方案：

```go
sig := ed25519.Sign(privateKey, message)
ok  := ed25519.Verify(publicKey, message, sig)
```

## 与 libp2p 的桥接

Ed25519 私钥可以直接转换为 libp2p 的 crypto.PrivKey：

```go
libp2pKey, _ := crypto.UnmarshalEd25519PrivateKey(identity.PrivateKey)
// 用于创建 libp2p Host
```

PeerID 由 libp2p 从公钥派生，与 DID 形成一一对应。

## 安全边界

| 威胁 | 处理方式 |
|------|--------|
| 密钥泄露 | 本地文件，提示用户保护 |
| 密钥轮换 | 暂不支持 |
| 多设备 | 暂不支持 |
| DID 文档发布 | 不需要（did:key 自描述） |

## 存储

```
~/.redanet/
├── identity.key    # Ed25519 私钥 (64 bytes, 0600)
└── redanet.db      # SQLite 数据库
```
