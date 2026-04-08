# 10 — 开发待办清单

## 当前判断

- [x] 当前仓库已进入骨架实现阶段；`cmd/`、`internal/`、`go.mod` 已建立，但 P2P、DM、Task、Board 仍未完成
- [ ] 近期目标建议收敛为 `v0.1 Demo`：双节点启动、互联、发 DM、跑通任务闭环、可查看 Board
- [ ] 交付形态保持为单二进制 `redanet`

---

## 0. 先决决策（开工前先统一口径）

### 必须先定

- [ ] 确认 `v0.1` 范围
  建议：`Peers + DM + Task + Board + Credits` 作为 P0，`Tip` 作为 P1
- [ ] 统一任务状态机
  当前文档对 `abandon` 的结果存在冲突：一处写“回到 created”，一处图示接近“进入 expired”
- [ ] 确认 API 鉴权是否首版上线
  `07-api.md` 写了 Bearer Token，但路线图未纳入实现任务
- [x] 确认 `/api/discover` 的 MVP 定义
  当前只有接口形态，没有可直接编码的数据来源、索引策略和排序规则
- [ ] 确认 Tip 的网络模型
  `09-tip.md` 提到本地存储 + `/anet/tip/1.0.0` 按需拉取，但路线图尚未覆盖

### 推荐默认口径

- [ ] `v0.1` 不做复杂发现搜索，`/api/discover` 可先返回已连接 peers 的简化结果，或直接延后
- [ ] `abandon` 作为动作，不作为稳定终态；执行后任务回到 `created`，并记录审计事件
- [ ] Bearer Token 先做“可配置开启”，默认仅绑定 `127.0.0.1`
- [x] Tip 先做本地上传/查询接口，不做复杂同步；当前已补充简化跨节点按需拉取

---

## 1. P0 主线开发待办（必须完成）

### 1.1 工程骨架

- [x] 初始化 `go.mod`
- [x] 建立目录结构：`cmd/redanet`、`internal/config`、`internal/identity`、`internal/p2p`、`internal/store`、`internal/daemon`
- [x] 增加版本信息常量与 `version` 子命令
- [x] 增加 `start`、`status` CLI 入口
- [ ] 统一日志输出格式
- [x] 明确默认数据目录：`~/.redanet/`

### 1.2 配置系统

- [x] 定义配置结构体：监听地址、API 端口、bootstrap peers、数据目录、API token、日志级别
- [x] 支持从 YAML 加载配置
- [x] 支持默认配置回填
- [x] 支持 CLI 参数覆盖关键配置
- [ ] 明确配置文件路径与找不到配置时的行为

### 1.3 身份与密钥

- [x] 实现 Ed25519 密钥生成
- [x] 实现本地密钥加载与 `0600` 权限写入
- [x] 实现 `did:key` 编码
- [ ] 实现 Ed25519 与 libp2p 私钥桥接
- [x] 暴露本节点 `DID`、`PeerID`
- [ ] 为后续签名校验预留接口

### 1.4 存储层

- [x] 接入 SQLite
- [x] 建立数据库初始化流程
- [x] 建立 schema migration 机制
- [x] 创建基础表：`tasks`、`credit_events`、`direct_messages`
- [x] 预留 `tip_qrcodes` migration，但可按 P1 启用
- [x] 统一时间格式为 RFC3339 UTC
- [ ] 增加基础仓储层封装与事务边界

### 1.5 Daemon 与基础 API

- [x] 启动 HTTP server
- [x] 注册路由与中间件
- [x] 实现 `GET /api/status`
- [x] 实现 `POST /api/shutdown`
- [x] 统一 JSON 编码、错误格式、状态码
- [x] 如果启用鉴权，补充 Bearer Token 中间件

### 1.6 P2P 网络

- [x] 创建 libp2p Host
- [x] 开启 TCP 与 QUIC 监听
- [x] 启动 Kademlia DHT
- [x] 启动 GossipSub
- [x] 连接 bootstrap peers
- [x] 维护本节点可公布的 multiaddr 列表
- [x] 注册 `/anet/dm/1.0.0` stream handler
- [x] 订阅 `/anet/dm`
- [x] 订阅 `/anet/tasks`

### 1.7 Peers 能力

- [x] 实现 `GET /api/peers`
- [x] 实现 `POST /api/peers/connect`
- [x] 将 multiaddr 解析为 `peer.AddrInfo`
- [x] 返回当前连接数、peer ID、地址列表
- [x] 明确连接失败时的错误提示

### 1.8 DM 消息

- [x] 定义 `DirectMessage` 模型
- [x] 实现 DM 表 CRUD
- [x] 实现消息 ID 去重
- [x] 实现 NaCl box 加密/解密工具
- [x] 实现 Ed25519 到 Curve25519 的密钥转换
- [x] 实现优先直连、失败回退 GossipSub 的投递策略
- [x] 实现 `POST /api/dm/send`
- [x] 实现 `GET /api/dm/inbox`
- [x] 实现 `GET /api/dm/thread/{peerDID}`
- [x] 明确本地明文缓存是否默认启用

### 1.9 Credits 信用系统

- [ ] 定义余额模型与信用事件模型
- [x] 首次启动发放初始余额 `10000`
- [x] 实现余额查询
- [ ] 实现记账函数，保证单节点内事务一致性
- [x] 实现 `GET /api/credits/balance`
- [x] 实现 `GET /api/credits/events`
- [ ] `POST /api/credits/transfer` 若进入 `v0.1`，需补充转账校验与对账逻辑；否则延后

### 1.10 Task 状态机

- [x] 定义 `Task` 模型
- [x] 实现 Task 表 CRUD
- [x] 明确 reward、fee、deposit 的计算函数
- [x] 实现发布任务：锁定 `reward + fee`
- [x] 实现认领任务：锁定 `deposit`
- [x] 实现提交结果
- [x] 实现验收通过后的结算
- [x] 实现拒绝、争议、仲裁
- [x] 实现取消、放弃、过期处理
- [x] 用显式状态迁移校验非法操作
- [x] 实现 `GET /api/tasks/{id}`
- [x] 实现 `POST /api/tasks`
- [x] 实现 `POST /api/tasks/{id}/claim`
- [x] 实现 `POST /api/tasks/{id}/submit`
- [x] 实现 `POST /api/tasks/{id}/accept`
- [x] 实现 `POST /api/tasks/{id}/reject`
- [x] 实现 `POST /api/tasks/{id}/dispute`
- [x] 实现 `POST /api/tasks/{id}/cancel`
- [x] 实现 `POST /api/tasks/{id}/abandon`
- [x] 实现 `POST /api/tasks/{id}/arbitrate`

### 1.11 Task 广播与同步

- [x] 定义 `/anet/tasks` 消息结构
- [x] 任务创建与状态变更时广播 GossipSub 消息
- [x] 接收远端任务变更并写入本地库
- [ ] 处理重复消息与旧状态覆盖问题
- [ ] 首版可先跳过签名校验，但需预留校验点

### 1.12 Board

- [x] 实现 `GET /api/tasks/board`
- [x] 支持 `state`、`q`、`limit` 查询参数
- [x] 实现 `GET /api/tasks/board/stats`
- [x] 实现服务端渲染 `GET /ui/board`
- [x] 看板按 `created / claimed / submitted / accepted / disputed` 分列展示
- [ ] 明确本地视图与“全网最终一致”之间的说明文案

### 1.13 最低测试与验收

- [ ] 为配置、identity、fee/deposit 计算、状态迁移写单元测试
- [ ] 为 store migration 写初始化测试
- [x] 为 DM 去重写测试
- [x] 为任务结算写表驱动测试
- [x] 跑通双节点手工集成流程
- [ ] 编写可复现 Demo 脚本
- [ ] 补全 README 快速启动步骤

---

## 2. P1 展示增强（建议在 P0 跑通后补）

### 2.1 Tip 打赏

- [x] 创建 `tip_qrcodes` 表
- [x] 实现图片上传校验：格式、大小、magic bytes
- [x] 实现 `POST /api/tip/qrcode`
- [x] 实现 `GET /api/tip/qrcode/{did}`
- [x] 实现 `DELETE /api/tip/qrcode`
- [x] 实现 `GET /api/tip/status/{did}`
- [x] 在 `accept` 响应中追加 tip 信息
- [x] 在 `/ui/board` 已完成卡片中增加“打赏”入口

### 2.2 Demo 体验

- [ ] 增加更友好的 HTML 样式
- [ ] 增加样例配置文件
- [ ] 增加本地双节点一键演示脚本
- [ ] 增加常见错误排查说明

---

## 3. P2 延后项（不要阻塞首版）

- [x] `/api/discover` 的真正搜索能力
- [x] `Profile / ANS` 本地持久化与 `register / resolve / lookup`
- [ ] 任务与 DM 的签名验证闭环
- [x] Tip 的跨节点拉取协议 `/anet/tip/1.0.0`
- [ ] Peer reputation / `peers` 持久化表
- [ ] 密钥轮换
- [ ] 多设备身份同步
- [ ] 更完整的仲裁机制
- [ ] 后台定时器：自动过期、清理、重试
- [ ] 观察性：结构化日志、metrics、pprof

---

## 4. 推荐实施顺序

- [ ] 第 1 天：工程骨架、配置、身份、SQLite、`/api/status`
- [ ] 第 2 天：P2P、Peers API、基础双节点联通
- [ ] 第 3 天：DM 存储、加密、直连与广播
- [ ] 第 4 天：Credits、Task 状态机、任务广播
- [ ] 第 5 天：Board、HTML 页面、README、Demo
- [ ] 第 6 天：Bug 修复、集成测试、范围收敛

---

## 5. 完成标准

- [ ] `go build -o redanet ./cmd/redanet` 成功
- [ ] 两个节点可互相连接并在 `/api/peers` 中看到彼此
- [ ] Agent A 能发 DM，Agent B 能在 inbox 中查收
- [ ] Agent A 发布任务，Agent B 认领/提交，Agent A 验收成功
- [ ] 双方余额变化符合结算规则
- [ ] 浏览器打开 `/ui/board` 能看到任务看板
- [ ] README 能指导陌生开发者在本地复现上述流程

---

## 6. 实施备注

- [ ] 所有时间字段统一使用 RFC3339，避免本地时区混乱
- [ ] 所有金额计算使用整数，避免浮点误差
- [ ] 所有状态迁移必须经由显式校验函数，不要在 handler 内随意改状态
- [ ] 所有跨节点同步默认按“幂等写入”设计
- [ ] 首版优先保证“能跑通”，再补签名、仲裁细节和高级发现能力
