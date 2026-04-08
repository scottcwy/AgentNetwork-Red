# 11 — 交接摘要

> 分支：`codex-skills-lightweight-directory`
>
> 远端：`origin/codex-skills-lightweight-directory`

## 一句话结论

这个分支目前包含两条已经可交接的工作流：

1. `Skills`：已经从“外链观察名单”升级为“第三方 `SKILL.md` 入库 + 最小 manager”
2. `BD`：已经补出极简小红书种子库、校验命令、同步入口和 wrap-up display

如果只交接一条，优先交接 `Skills`。

## 当前重点

### Skills

- 主 issue：[#8](https://github.com/chatchat-space/AgentNetwork-Red/issues/8)
- 当前状态：`已入库 + 可浏览 + 可搜索 + 可 show + 可 install`
- 已导入 20 个第三方 `SKILL.md`
- 当前统一落点：
  - `skills/third_party/github/...`
  - `skills/catalog.json`
- 当前 manager：
  - `cmd/skillmgr`
  - `internal/skills`

### BD

- 主 epic：[#2](https://github.com/chatchat-space/AgentNetwork-Red/issues/2)
- 当前状态：`种子库结构 + inbox/sync + wrap-up display 已打通`
- 当前入口：
  - `bd/README.md`
  - `cmd/bdcheck`
  - `cmd/bdsync`
  - `cmd/bdwrap`

## 关键提交

### Skills 线

- `a566001` 把 Skills 目录从重设计稿收口成可执行入口
- `37a0cd2` 把 Skills 主线改成第三方 Skill 入库与安装
- `a6c8ab6` 把第三方 Skill 种子库拉进仓库
- `9987301` 给 Skills 种子库补上最小 manager

### BD 线

- `2cb0307` Lock the lightweight XHS seed capture flow before bot integration

## 现在能直接跑的命令

### Skills

```bash
go run ./cmd/skillmgr list
go run ./cmd/skillmgr search colleague
go run ./cmd/skillmgr show github/titanwings/colleague-skill
go run ./cmd/skillmgr install github/titanwings/colleague-skill
```

说明：

- `install` 默认安装到 `~/.codex/skills/`
- 当前导入的 Skill 默认都是 `skill-md-only` + `unverified`
- 现在的 install 只复制 `SKILL.md + metadata.json`

### BD

```bash
go run ./cmd/bdcheck ./bd
go run ./cmd/bdsync --inbox ./.bd-inbox ./bd
go run ./cmd/bdwrap ./bd
```

## 当前验证状态

- `go test ./...` 通过
- `go vet ./...` 通过
- `skillmgr list/search/show/install` 已实测

## 接手人最先该知道的事

- `#8` 的早期评论里有一段“stars 门槛 / 只观察不 vendor”的旧讨论，那已经不是当前主线；以 issue 正文和后续进展评论为准。
- 当前 Skills 库是“先入库后验证”的策略，不要把 `imported` 误解成“已经可直接运行”。
- `skillmgr` 现在只解决最小闭环：`list/search/show/install`，还没做完整验证、同步和版本管理。
- 这个分支里 BD 和 Skills 同时存在，但两条线可以拆开维护，不要求一个人同时推进两边。
