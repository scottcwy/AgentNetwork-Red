# 06 — Peer 发现与管理

## 设计目标

让 Agent 能够发现网络中的其他 Agent，查看连接状态，手动建立连接。

## Peer 概念

一个 Peer 就是网络中的一个 Agent 节点，由以下属性标识：

```go
type PeerInfo struct {
    PeerID string   `json:"peer_id"` // libp2p PeerID (12D3Koo...)
    DID    string   `json:"did"`     // did:key:z6Mk...
    Addrs  []string `json:"addrs"`   // multiaddr 列表
}
```

## 发现机制

### 1. Bootstrap（启动时连接种子节点）

```
启动:
  for seed in config.bootstrap_peers:
    go connect(seed)
    
  // 连接成功后，通过 DHT 自动发现更多节点
```

### 2. DHT（Kademlia 分布式哈希表）

```
加入 DHT 后:
  - 自动被其他节点发现
  - 可以通过 peer.ID 查找节点地址
  - DHT 持续在后台 crawling
```

### 3. 手动连接

```
POST /api/peers/connect
{"addr": "/ip4/1.2.3.4/tcp/4001/p2p/12D3KooW..."}
```

## API

### 列出连接的 Peers
```
GET /api/peers
→ {
    "count": 3,
    "peers": [
      {
        "peer_id": "12D3KooW...",
        "addrs": ["/ip4/10.0.1.5/tcp/4001"]
      },
      ...
    ]
  }
```

### 手动连接
```
POST /api/peers/connect
{"addr": "/ip4/1.2.3.4/tcp/4001/p2p/12D3KooW..."}
→ {"ok": true, "peer_id": "12D3KooW..."}
```

### 发现/搜索 Agent
```
GET /api/discover?q=translation&skills=nlp&limit=10
→ {
    "results": [
      {
        "name": "agent://translate-bot",
        "peer_id": "12D3KooW...",
        "description": "多语言翻译 Agent",
        "skills": ["nlp", "translation"],
        "score": 0.95
      }
    ],
    "elapsed": "45ms"
  }
```

## Peer 状态

本地维护连接的 Peer 列表（来自 libp2p Peerstore）：

```go
// 获取当前已连接的 peer 列表
func (n *Node) ConnectedPeers() []peer.ID {
    return n.Host.Network().Peers()
}
```

## 存储

简化版不持久化 peer 列表（由 libp2p 管理内存中的 Peerstore）。

后续版本可以添加：
```sql
CREATE TABLE peers (
    did             TEXT PRIMARY KEY,
    peer_id         TEXT,
    last_seen       TEXT,
    reputation      REAL DEFAULT 0,
    connection_count INTEGER DEFAULT 0
);
```
