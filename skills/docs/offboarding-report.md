# 12 — 离职交接报告

## 背景

本次交接覆盖 `codex-skills-lightweight-directory` 分支上的两条工作：

1. `Skills`：把仓库从“收集外链”推进到“第三方 Skill 种子库 + 最小 manager”
2. `BD`：把极简小红书种子库、同步层和 wrap-up display 收口到可运行状态

其中，`Skills` 是当前最建议继续推进的主线。

## 本阶段完成内容

### A. Skills

#### 1. 目标切换

已经明确放弃“只观察、不 vendor”的旧思路，转为：

- 直接抓取第三方公开 `SKILL.md`
- 入库到本仓库
- 为每个 Skill 补最小元数据
- 用 manager 提供懒人式挑选和安装

当前主线 issue：

- [#8 Skills: 入库第三方 Skill.md，并实现傻瓜式挑选安装](https://github.com/chatchat-space/AgentNetwork-Red/issues/8)

说明：

- [#9](https://github.com/chatchat-space/AgentNetwork-Red/issues/9) 已经关闭并被 `#8` 吸收
- `#8` 里最早一条关于 “stars 门槛 / 实验区” 的评论属于历史讨论，不再是当前执行口径

#### 2. 文档重定向

已经更新：

- `skills/README.md`
- `skills/DESIGN.md`

文档当前表达的是：

- 第三方 Skill 入库
- 最小元数据
- manager 消费 `catalog.json`
- 当前状态区分 `skill-md-only` / `unverified`

#### 3. 第三方 Skill 种子库

当前统一落点：

- `skills/third_party/github/...`
- `skills/catalog.json`

当前已入库 20 个 `SKILL.md`。

来源分布：

- `HughYau/qiushi-skill`：11 个
- `mliu98/awesome-human-distillation` 里挑出的 5 个高活跃仓库：5 个
- `VoltAgent/awesome-openclaw-skills` 指向的 `openclaw/skills`：4 个

每个叶子目录包含：

- `SKILL.md`
- `metadata.json`

当前元数据最少覆盖：

- `id`
- `provider`
- `local_path`
- `source_repo_url`
- `source_path`
- `source_raw_url`
- `discovery_source`
- `imported_at`
- `import_mode`
- `status`

当前默认值：

- `import_mode = skill-md-only`
- `status = unverified`

#### 4. 最小 manager

当前新增：

- `cmd/skillmgr/main.go`
- `internal/skills/catalog.go`
- `internal/skills/install.go`
- `internal/skills/catalog_test.go`

当前能力：

- `list`
- `search`
- `show`
- `install`

当前行为：

- `list/search/show` 直接消费 `skills/catalog.json`
- `show` 会读取 `SKILL.md` frontmatter 补出 `name/description/argument-hint`
- `install` 目前只复制 `SKILL.md + metadata.json`
- `install` 默认写入 `~/.codex/skills/`
- 支持 `-target` 覆盖目标目录
- 支持 flag 写在 query 前后，避免 Go 默认 flag 顺序坑掉可用性

已实测命令：

```bash
go run ./cmd/skillmgr list -limit 3
go run ./cmd/skillmgr search colleague
go run ./cmd/skillmgr show github/titanwings/colleague-skill
go run ./cmd/skillmgr install github/titanwings/colleague-skill -target /tmp/skillmgr-install-test
```

### B. BD

#### 1. 当前目标

这一层现在是“小红书种子库 + 本地 inbox/sync + wrap-up display”，不是完整人物档案系统。

当前主 epic：

- [#2 Epic: 小红书种子库 + Wrap-Up Display MVP](https://github.com/chatchat-space/AgentNetwork-Red/issues/2)

#### 2. 当前结果

已经新增：

- `bd/README.md`
- `bd/schema.md`
- `bd/sop.md`
- `bd/lark-bot-input.md`
- `bd/inbox.md`
- `bd/db-schema.md`
- `bd/db-schema.sql`
- `bd/xhs_seeds.jsonl`
- `bd/display/index.html`
- `bd/photos/*`
- `cmd/bdcheck`
- `cmd/bdsync`
- `cmd/bdwrap`
- `internal/bd/*`

当前能力：

- 校验种子库
- 从 `.bd-inbox/submissions.jsonl` 同步到 `bd/xhs_seeds.jsonl`
- 检查冲突字段并打 `needs_review`
- 复制封面图到 `bd/photos/`
- 生成 wrap-up HTML

#### 3. 当前可运行命令

```bash
go run ./cmd/bdcheck ./bd
go run ./cmd/bdsync --inbox ./.bd-inbox ./bd
go run ./cmd/bdwrap ./bd
```

## 验证情况

已通过：

- `go test ./...`
- `go vet ./...`

Skills 额外验证：

- `go test ./cmd/skillmgr ./internal/skills`
- `go vet ./cmd/skillmgr ./internal/skills`
- `skillmgr list/search/show/install`

BD 额外验证：

- `go test ./internal/bd ./cmd/bdcheck ./cmd/bdsync ./cmd/bdwrap`

## 提交链

### Skills

- `a566001` 把 Skills 目录从重设计稿收口成可执行入口
- `37a0cd2` 把 Skills 主线改成第三方 Skill 入库与安装
- `a6c8ab6` 把第三方 Skill 种子库拉进仓库
- `9987301` 给 Skills 种子库补上最小 manager

### BD

- `2cb0307` Lock the lightweight XHS seed capture flow before bot integration

## 已知边界与风险

### Skills

- 当前已入库不等于已验证可直接运行
- 当前 install 只复制 `SKILL.md + metadata.json`
- 尚未处理第三方 Skill 的 `scripts/`、`assets/`、`references/` 等伴随资源
- 尚未做上游同步、版本跟踪、冲突更新策略
- 尚未做 install 后状态回写，例如从 `unverified` 过渡到 `installed` / `verified`

### BD

- 当前还是极简种子库，不是完整人物档案系统
- `.bd-inbox/` 属于本地工作目录，已通过 `.gitignore` 忽略
- 同步流程当前依赖 JSONL 和本地文件布局，尚未接真实 bot 自动流

## 给接手人的建议顺序

### 如果继续做 Skills

1. 先继续围绕 `#8` 做，不要重新开平行主线
2. 优先补 `skillmgr` 的状态管理：
   - `imported`
   - `installed`
   - `verified`
3. 再决定是否支持伴随资源复制：
   - `scripts/`
   - `assets/`
   - `references/`
4. 最后再考虑上游同步和版本治理

最重要的一条：

先把“已入库但未验证”和“可直接安装使用”严格区分，不要把 catalog 做成误导用户的假成功列表。

### 如果继续做 BD

1. 继续沿 `#2` 的 issue 树推进
2. 优先把 bot 采集和本地 inbox 的桥接做稳
3. 保持 `bdcheck -> bdsync -> bdwrap` 这条链路简洁可复现

## 分支与远端

- 本地分支：`codex-skills-lightweight-directory`
- 远端分支：`origin/codex-skills-lightweight-directory`

当前可以直接把这条分支交给其他人继续开发，不需要再额外整理上下文。
