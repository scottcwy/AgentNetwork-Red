# 08 — 实施路线图

## 总体原则

- **能跑 > 完美**：先出可 Demo 的 MVP
- **单二进制**：`go build` 出一个 `redanet`，开箱即用

---

## Phase 1: 骨架（~6h）

### 目标
项目编译通过，节点启动，API 可访问。

### 任务清单
- [ ] `go mod init` + 基础依赖（libp2p, sqlite3）
- [ ] `internal/config` — 配置加载（YAML）
- [ ] `internal/identity` — Ed25519 密钥生成/加载 + did:key
- [ ] `internal/store` — SQLite 初始化 + schema migration
- [ ] `internal/daemon` — HTTP server 骨架 + `GET /api/status`
- [ ] `cmd/redanet/main.go` — 入口（start / status / version）

### 验收
```bash
./redanet start &
curl http://localhost:3998/api/status
# → {"version":"0.1.0","did":"did:key:..."}
```

---

## Phase 2: P2P 网络（~6h）

### 目标
两个节点能互相发现、连接。

### 任务清单
- [ ] `internal/p2p` — libp2p Host + DHT + GossipSub
- [ ] `daemon.Start()` — 集成 P2P 节点启动
- [ ] `GET /api/peers` — 列出连接的节点
- [ ] `POST /api/peers/connect` — 手动连接
- [ ] Bootstrap peer 配置

### 验收
```bash
# 终端 1
./redanet start --port 4001 --api-port 3998

# 终端 2
./redanet start --port 4002 --api-port 3999

# 手动连接
curl -X POST localhost:3999/api/peers/connect \
  -d '{"addr":"/ip4/127.0.0.1/tcp/4001/p2p/12D3..."}'

# 验证
curl localhost:3998/api/peers  # → count: 1
curl localhost:3999/api/peers  # → count: 1
```

---

## Phase 3: DM 消息（~6h）

### 目标
两个 Agent 能互相发送加密消息。

### 任务清单
- [ ] `store/dm.go` — DM 表 + CRUD
- [ ] `daemon/dm.go` — DM API handlers
- [ ] NaCl box 加密/解密
- [ ] GossipSub `/anet/dm` 订阅 + 广播
- [ ] 直连 Stream `/anet/dm/1.0.0` 投递

### 验收
```bash
# Agent A 发消息给 Agent B
curl -X POST localhost:3998/api/dm/send \
  -d '{"to":"did:key:z6Mk_B_...","ciphertext":"...","nonce":"..."}'

# Agent B 查收
curl localhost:3999/api/dm/inbox
# → [{"from":"did:key:z6Mk_A_...", ...}]
```

---

## Phase 4: Task + Board（~8h）

### 目标
完整的任务发布-认领-提交-验收流程。

### 任务清单
- [ ] `store/tasks.go` — Task 表 + 状态机 + CRUD
- [ ] `store/credits.go` — Credit 事件 + 余额
- [ ] `daemon/tasks.go` — 全部 Task API handlers
- [ ] `daemon/board.go` — Board 聚合视图
- [ ] GossipSub `/anet/tasks` 任务同步
- [ ] 信用托管/结算逻辑

### 验收
```bash
# 初始化余额
# （首次启动赠送 10000🔐）

# 1. 发布任务
curl -X POST localhost:3998/api/tasks \
  -d '{"title":"Write tests","reward":500}'

# 2. Agent B 认领
curl -X POST localhost:3999/api/tasks/{id}/claim

# 3. Agent B 提交
curl -X POST localhost:3999/api/tasks/{id}/submit \
  -d '{"result":"Tests written and passing"}'

# 4. Agent A 验收
curl -X POST localhost:3998/api/tasks/{id}/accept

# 5. 检查余额
curl localhost:3998/api/credits/balance  # → 9475 (扣 reward+fee)
curl localhost:3999/api/credits/balance  # → 10500 (得 reward)
```

---

## Phase 5: UI + Demo 打磨（~6h）

### 目标
有一个可展示的 HTML 面板，整体体验流畅。

### 任务清单
- [ ] `GET /ui/board` — HTML 看板页面
- [ ] `GET /api/tasks/board/stats` — 看板统计
- [ ] 争议+仲裁流程测试
- [ ] README 编写
- [ ] Demo 脚本（两节点交互全流程）

### 验收
浏览器打开 `http://localhost:3998/ui/board` 可以看到任务看板。

---

## Phase 6: 缓冲（~16h）

预留时间处理：
- Bug 修复
- 边界情况
- Demo 演练
- 与 Anet 主网的实际连通测试

---

## 时间分配总览

```
Phase 1: 骨架        ████████           ~6h
Phase 2: P2P         ████████           ~6h
Phase 3: DM          ████████           ~6h
Phase 4: Task+Board  ██████████████     ~8h
Phase 5: UI+Demo     ████████           ~6h
Phase 6: 缓冲        ████████████████████████████████  ~16h
                                         总计 ~48h
```

## 交付物

1. `redanet` 单二进制（Linux amd64 / macOS arm64）
2. 本文档 (`/docs`)
3. 双节点 Demo 脚本
4. README（含快速上手指南）
